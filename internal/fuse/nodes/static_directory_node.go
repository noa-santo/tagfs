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
)

var staticDirLogger = log.New(os.Stdout, "STATIC_DIR: ", log.LstdFlags)

type staticDirectoryNode struct {
	fs.Inode
	nodeConfig config.DirectoryConfig
}

func (n *staticDirectoryNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	nodes, err := db.Get().GetNodesForDir(ctx, n.nodeConfig.Tags)
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

func (n *staticDirectoryNode) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	out.Mode = syscall.S_IFDIR | 0755
	return fs.OK
}

var _ fs.NodeReaddirer = (*staticDirectoryNode)(nil)
var _ fs.NodeLookuper = (*staticDirectoryNode)(nil)
var _ fs.NodeGetattrer = (*staticDirectoryNode)(nil)
