package nodes

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db"
	"github.com/noa-santo/tagfs/internal/db/gen"
	"github.com/oklog/ulid/v2"
)

var rootLogger = log.New(os.Stdout, "ROOT NODE: ", log.LstdFlags|log.Lmicroseconds)

type RootNode struct {
	fs.Inode
	inboxNode *inboxNode
}

func (n *RootNode) Init(ctx context.Context) {
	rootLogger.Printf("Initializing passthrough directories...")
	n.initPassthrough(ctx)
	rootLogger.Printf("Initializing inbox...")
	n.initInbox(ctx)
	rootLogger.Printf("Initializing dynamic directories...")
	n.initStaticDirectories(ctx, &n.Inode, config.Get().StoragePath, config.Get().Directories, 0)
	rootLogger.Printf("Initialization complete")
}

func (n *RootNode) initPassthrough(ctx context.Context) {
	for _, dirName := range config.Get().PassthroughDirs {
		path := filepath.Join(config.Get().StoragePath, dirName)
		childNode := &passthroughNode{Path: path}
		childINode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR})
		n.AddChild(dirName, childINode, true)
	}
}

func (n *RootNode) initInbox(ctx context.Context) {
	n.inboxNode = &inboxNode{}
	childINode := n.NewPersistentInode(ctx, n.inboxNode, fs.StableAttr{Mode: syscall.S_IFDIR})
	n.AddChild(config.Get().InboxDir, childINode, true)
}

func (n *RootNode) initStaticDirectories(ctx context.Context, parentInode *fs.Inode, parentPath string, dirConfigs []config.DirectoryConfig, level int) {
	for _, dirConf := range dirConfigs {
		dirPath := filepath.Join(parentPath, dirConf.Name)
		dirConf.Tags = append(dirConf.Tags, fmt.Sprintf("level:%d", level))
		childNode := &staticDirectoryNode{
			nodeConfig: dirConf,
		}
		childInode := parentInode.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR})
		parentInode.AddChild(dirConf.Name, childInode, true)
		if len(dirConf.Subdirectories) > 0 {
			n.initStaticDirectories(ctx, childInode, dirPath, dirConf.Subdirectories, level+1)
		}
	}
}

func (n *RootNode) Create(ctx context.Context, name string, flags uint32, mode uint32, _ *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	rootLogger.Printf("Ingesting new file at root: %s", name)
	fileID := ulid.Make().String()
	dataPath := filepath.Join(config.Get().StoragePath, ".data", fileID)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	physicalPath := filepath.Join(dataPath, name)
	f, err := os.OpenFile(physicalPath, int(flags)|os.O_CREATE, os.FileMode(mode))
	if err != nil {
		rootLogger.Printf("Error creating physical file: %v", err)
		return nil, nil, 0, fs.ToErrno(err)
	}
	err = db.Get().Queries.InsertFile(ctx, gen.InsertFileParams{
		ID:       fileID,
		OrigName: name,
		Mode:     int64(mode),
	})
	if err != nil {
		err := f.Close()
		if err != nil {
			rootLogger.Printf("Error closing physical file: %v", err)
			return nil, nil, 0, 0
		}
		err = os.Remove(physicalPath)
		if err != nil {
			rootLogger.Printf("Error removing physical file: %v", err)
			return nil, nil, 0, 0
		}
		rootLogger.Printf("Error inserting file into DB: %v", err)
		return nil, nil, 0, syscall.EIO
	}
	childNode := &passthroughNode{Path: physicalPath}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: mode})
	wrappedFh := &rootFileHandle{
		file:     f,
		rootNode: n,
		name:     name,
		fileID:   fileID,
	}
	return childInode, wrappedFh, 0, 0
}

func (n *RootNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rootLogger.Printf("Ingesting new directory at root: %s", name)
	dirID := ulid.Make().String()
	err := db.Get().Queries.InsertDynamicDirectory(ctx, gen.InsertDynamicDirectoryParams{
		ID:       dirID,
		ParentID: sql.NullString{Valid: false},
		Name:     name,
	})
	if err != nil {
		rootLogger.Printf("Error inserting directory into DB: %v", err)
		return nil, syscall.EIO
	}

	childNode := &fs.Inode{}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR | mode})

	out.EntryValid = 0
	out.AttrValid = 0
	go func() {
		n.RmChild(name)
		errno := n.NotifyEntry(name)
		if errno != 0 {
			rootLogger.Printf("Error notifying entry %q: %v", name, errno)
		}
	}()

	return childInode, 0
}

type rootFileHandle struct {
	file     *os.File
	rootNode *RootNode
	name     string
	fileID   string
}

func (fh *rootFileHandle) Write(_ context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	written, err := fh.file.WriteAt(data, off)
	if err != nil {
		return uint32(written), fs.ToErrno(err)
	}
	return uint32(written), 0
}

func (fh *rootFileHandle) Read(_ context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	readBytes, err := fh.file.ReadAt(dest, off)
	if err != nil && err.Error() != "EOF" {
		return nil, fs.ToErrno(err)
	}
	return fuse.ReadResultData(dest[:readBytes]), 0
}

func (fh *rootFileHandle) Flush(_ context.Context) syscall.Errno {
	err := fh.file.Sync()
	if err != nil {
		return fs.ToErrno(err)
	}
	return 0
}

func (fh *rootFileHandle) Release(ctx context.Context) syscall.Errno {
	rootLogger.Printf("Releasing ingested file: %s (ID: %s)", fh.name, fh.fileID)

	info, err := fh.file.Stat()
	if err != nil {
		rootLogger.Printf("Error reading stats for file %s: %v", fh.name, err)
	} else {
		dbErr := db.Get().Queries.UpdateFileStats(ctx, gen.UpdateFileStatsParams{
			Mode: int64(info.Mode()),
			ID:   fh.fileID,
		})
		if dbErr != nil {
			rootLogger.Printf("Error updating DB stats for file %s: %v", fh.name, dbErr)
		}
	}
	closeErr := fh.file.Close()

	go func() {
		fh.rootNode.RmChild(fh.name)
		errno := fh.rootNode.NotifyEntry(fh.name)
		if errno != 0 {
			rootLogger.Printf("Error notifying entry %q after release: %v", fh.name, errno)
		}
	}()

	if closeErr != nil {
		return fs.ToErrno(closeErr)
	}
	return fs.OK
}

var _ = (fs.NodeMkdirer)((*RootNode)(nil))
var _ = (fs.NodeCreater)((*RootNode)(nil))

var _ fs.FileWriter = (*rootFileHandle)(nil)
var _ fs.FileReader = (*rootFileHandle)(nil)
var _ fs.FileFlusher = (*rootFileHandle)(nil)
var _ fs.FileReleaser = (*rootFileHandle)(nil)
