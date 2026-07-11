package nodes

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/gabriel-vasile/mimetype"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db"
	"github.com/noa-santo/tagfs/internal/db/gen"
	. "github.com/noa-santo/tagfs/internal/shared"
)

var inboxLogger = log.New(os.Stdout, "INBOX NODE: ", log.LstdFlags)

type inboxNode struct {
	fs.Inode
}

// TODO: optimize inbox node
func categorizedFileIDs(ctx context.Context, dirConfigs []config.DirectoryConfig) (map[string]bool, error) {
	matched := make(map[string]bool)
	var walk func([]config.DirectoryConfig) error
	walk = func(configs []config.DirectoryConfig) error {
		for _, dirConf := range configs {
			files, err := db.Get().GetNodesForDir(ctx, dirConf.Tags)
			if err != nil {
				return err
			}
			for _, f := range files {
				matched[f.ID] = true
			}
			if len(dirConf.Subdirectories) > 0 {
				if err := walk(dirConf.Subdirectories); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(dirConfigs); err != nil {
		return nil, err
	}
	return matched, nil
}

func uncategorizedFiles(ctx context.Context) ([]gen.Node, error) {
	all, err := db.Get().Queries.GetAllNodes(ctx)
	if err != nil {
		return nil, err
	}
	categorized, err := categorizedFileIDs(ctx, config.Get().Directories)
	if err != nil {
		return nil, err
	}
	result := make([]gen.Node, 0, len(all))
	for _, f := range all {
		if !categorized[f.ID] {
			result = append(result, f)
		}
	}
	return result, nil
}

func (n *inboxNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	files, err := uncategorizedFiles(ctx)
	if err != nil {
		inboxLogger.Printf("Readdir: failed to compute uncategorized files: %v", err)
		return nil, syscall.EIO
	}
	result := make([]fuse.DirEntry, 0, len(files))
	for _, f := range files {
		result = append(result, fuse.DirEntry{
			Name: f.OrigName,
			Ino:  0,
			Mode: uint32(f.Mode),
		})
	}
	return fs.NewListDirStream(result), fs.OK
}

func (n *inboxNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	files, err := uncategorizedFiles(ctx)
	if err != nil {
		inboxLogger.Printf("Lookup: failed to compute uncategorized files: %v", err)
		return nil, syscall.EIO
	}
	for _, f := range files {
		if f.OrigName != name {
			continue
		}
		physicalPath := filepath.Join(config.Get().StoragePath, ".data", f.ID, f.OrigName)
		var st syscall.Stat_t
		if err := syscall.Stat(physicalPath, &st); err != nil {
			inboxLogger.Printf("Lookup: stat failed for %s (id=%s): %v", physicalPath, f.ID, err)
			return nil, fs.ToErrno(err)
		}
		out.Attr.FromStat(&st)
		childNode := &passthroughNode{Path: physicalPath}
		childInode := n.NewInode(ctx, childNode, fs.StableAttr{
			Mode: uint32(f.Mode) & syscall.S_IFMT,
		})
		return childInode, fs.OK
	}
	return nil, syscall.ENOENT
}

func (n *inboxNode) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0755
	return fs.OK
}

var _ fs.NodeReaddirer = (*inboxNode)(nil)
var _ fs.NodeLookuper = (*inboxNode)(nil)
var _ fs.NodeGetattrer = (*inboxNode)(nil)

func buildInboxEntry(f gen.Node) (InboxEntry, error) {
	physicalPath := filepath.Join(config.Get().StoragePath, ".data", f.ID, f.OrigName)
	entryInfo, err := os.Stat(physicalPath)
	if err != nil {
		inboxLogger.Printf("Error reading inbox: %v", err)
		return InboxEntry{}, err
	}

	var mimeType string
	if !entryInfo.IsDir() {
		mimeTypeRaw, err := mimetype.DetectFile(physicalPath)
		if err != nil {
			inboxLogger.Printf("Error detecting mime type: %v", err)
			return InboxEntry{}, err
		}
		mimeType = mimeTypeRaw.String()
	} else {
		mimeType = "inode/directory"
	}

	return InboxEntry{
		Name:       f.OrigName,
		IsDir:      entryInfo.IsDir(),
		ModifiedAt: entryInfo.ModTime().Format("02.01.2006 15:04:05"),
		Size:       entryInfo.Size(),
		MimeType:   mimeType,
		UID:        f.ID,
	}, nil
}

func GetInboxEntries() ([]InboxEntry, error) {
	files, err := uncategorizedFiles(context.Background())
	if err != nil {
		inboxLogger.Printf("Error reading inbox: %v", err)
		return []InboxEntry{}, err
	}
	entries := make([]InboxEntry, 0, len(files))
	for _, f := range files {
		entry, err := buildInboxEntry(f)
		if err != nil {
			return []InboxEntry{}, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func GetInboxEntry(uid string) (InboxEntry, error) {
	file, err := db.Get().Queries.GetNode(db.Get().Ctx, uid)
	if err != nil {
		return InboxEntry{}, err
	}
	return buildInboxEntry(file)
}
