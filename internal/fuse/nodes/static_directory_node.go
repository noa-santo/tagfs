package nodes

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db"
	"github.com/noa-santo/tagfs/internal/db/gen"
	"github.com/noa-santo/tagfs/internal/logic"
	"github.com/oklog/ulid/v2"
)

var staticDirLogger = log.New(os.Stdout, "STATIC_DIR: ", log.LstdFlags)

type staticDirectoryNode struct {
	fs.Inode
	nodeConfig config.DirectoryConfig
}

func (n *staticDirectoryNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags) // todo: also show static dirs in this list
	if err != nil {
		staticDirLogger.Printf("Readdir: GetFilesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, syscall.EIO
	}
	var result []fuse.DirEntry
	for _, node := range nodes {
		result = append(result, fuse.DirEntry{
			Name: node.OrigName,
			Ino:  0,
			Mode: uint32(node.Mode),
		})
	}
	return fs.NewListDirStream(result), fs.OK
}

func (n *staticDirectoryNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Lookup: GetFilesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, syscall.EIO
	}

	for _, node := range nodes {
		if node.OrigName != name {
			continue
		}

		physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)

		var st syscall.Stat_t
		if err := syscall.Stat(physicalPath, &st); err != nil {
			staticDirLogger.Printf("Lookup: stat failed for %s (id=%s): %v", physicalPath, node.ID, err)
			return nil, fs.ToErrno(err)
		}
		out.Attr.FromStat(&st)

		childNode := &passthroughNode{Path: physicalPath}
		childInode := n.NewInode(ctx, childNode, fs.StableAttr{
			Mode: uint32(node.Mode) & syscall.S_IFMT,
		})
		return childInode, fs.OK
	}

	return nil, syscall.ENOENT
}

func (n *staticDirectoryNode) Mkdir(ctx context.Context, name string, mode uint32, _ *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	if !n.nodeConfig.Rules.AllowSubdirCreation {
		return nil, syscall.EPERM
	}
	staticDirLogger.Printf("Ingesting new directory at %s: %s", n.nodeConfig.Name, name)
	dirID := ulid.Make().String()
	dataPath := filepath.Join(config.Get().StoragePath, ".data", dirID)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fs.ToErrno(err)
	}
	physicalPath := filepath.Join(dataPath, name)
	if err := os.Mkdir(physicalPath, os.FileMode(mode)); err != nil {
		return nil, fs.ToErrno(err)
	}
	err := db.Get().Queries.InsertNode(ctx, gen.InsertNodeParams{
		ID:       dirID,
		OrigName: name,
		Mode:     logic.ToStoredMode(mode, true),
	})
	if err != nil {
		staticDirLogger.Printf("Error inserting directory into DB: %v", err)
		return nil, syscall.EIO
	}

	err = db.Get().UpdateTags(dirID, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Error updating tags for directory %s: %v", dirID, err)
		return nil, syscall.EIO
	}

	childNode := &passthroughNode{Path: physicalPath}
	childInode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR | mode})

	return childInode, fs.OK
}

func (n *staticDirectoryNode) Rmdir(ctx context.Context, name string) syscall.Errno {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Rmdir: GetFilesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName != name {
			continue
		}
		physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)
		if err := os.RemoveAll(physicalPath); err != nil {
			staticDirLogger.Printf("Rmdir: RemoveAll failed for %s: %v", physicalPath, err)
			return syscall.EIO
		}
		err := db.Get().Queries.DeleteNode(ctx, node.ID)
		if err != nil {
			return fs.ToErrno(err)
		}
	}
	return fs.OK
}

func (n *staticDirectoryNode) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	// todo: implement rename logic for static directories
	// when in same directory, update name in db and rename physical file
	// when in different directory, create new node in new directory and delete old node
	panic("not implemented")
}

func (n *staticDirectoryNode) Create(ctx context.Context, name string, flags uint32, mode uint32, _ *fuse.EntryOut) (node *fs.Inode, fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	if !n.nodeConfig.Rules.AllowFileCreation {
		return nil, nil, 0, syscall.EPERM
	}
	staticDirLogger.Printf("Ingesting new file at %s: %s", n.nodeConfig.Name, name)
	fileID := ulid.Make().String()
	physicalPath := filepath.Join(config.Get().StoragePath, ".data", fileID, name)
	if err := os.MkdirAll(filepath.Dir(physicalPath), 0755); err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	f, err := os.OpenFile(physicalPath, int(flags)|os.O_CREATE, os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, fs.ToErrno(err)
	}
	err = db.Get().Queries.InsertNode(ctx, gen.InsertNodeParams{
		ID:       fileID,
		OrigName: name,
		Mode:     logic.ToStoredMode(mode, false),
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
	return childInode, f, 0, fs.OK
}

func (n *staticDirectoryNode) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Lookup: GetFilesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return nil, 0, syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName == n.nodeConfig.Name {
			physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)
			f, err := os.OpenFile(physicalPath, int(flags), 0)
			if err != nil {
				return nil, 0, fs.ToErrno(err)
			}
			return &passthroughFile{File: f}, 0, fs.OK
		}
	}
	return nil, 0, syscall.ENOENT
}

func (n *staticDirectoryNode) Unlink(ctx context.Context, name string) syscall.Errno {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Printf("Lookup: GetFilesForDir failed for %s: %v", n.nodeConfig.Name, err)
		return syscall.EIO
	}
	for _, node := range nodes {
		if node.OrigName == name {
			physicalPath := filepath.Join(config.Get().StoragePath, ".data", node.ID, node.OrigName)
			if err := os.Remove(physicalPath); err != nil {
				return fs.ToErrno(err)
			}
			err = db.Get().Queries.DeleteNode(ctx, node.ID)
			if err != nil {
				return fs.ToErrno(err)
			}
		}
	}
	return syscall.ENOENT
}

func (n *staticDirectoryNode) Setattr(context.Context, fs.FileHandle, *fuse.SetAttrIn, *fuse.AttrOut) syscall.Errno {
	return syscall.EPERM
}

func (n *staticDirectoryNode) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0755
	return fs.OK
}

var _ fs.NodeReaddirer = (*staticDirectoryNode)(nil)
var _ fs.NodeLookuper = (*staticDirectoryNode)(nil)
var _ fs.NodeMkdirer = (*staticDirectoryNode)(nil)
var _ fs.NodeRmdirer = (*staticDirectoryNode)(nil)
var _ fs.NodeRenamer = (*staticDirectoryNode)(nil)
var _ fs.NodeCreater = (*staticDirectoryNode)(nil)
var _ fs.NodeOpener = (*staticDirectoryNode)(nil)
var _ fs.NodeUnlinker = (*staticDirectoryNode)(nil)
var _ fs.NodeSetattrer = (*staticDirectoryNode)(nil)
var _ fs.NodeGetattrer = (*staticDirectoryNode)(nil)
