package fuse

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/db"
)

var rootLogger = log.New(os.Stdout, "ROOT NODE: ", log.LstdFlags|log.Lmicroseconds)

type rootNode struct {
	fs.Inode
	inboxNode *inboxNode
}

func (n *rootNode) init(ctx context.Context) {
	n.initPassthrough(ctx)
	n.initInbox(ctx)
}

func (n *rootNode) initPassthrough(ctx context.Context) {
	for _, dirName := range db.PassthroughDirs {
		config, _ := db.LoadConfig()
		path := filepath.Join(config.StoragePath, dirName)
		childNode := &passthroughNode{Path: path}
		childINode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR})
		n.AddChild(dirName, childINode, true)
	}
}

func (n *rootNode) initInbox(ctx context.Context) {
	n.inboxNode = newInboxNode()
	childINode := n.NewPersistentInode(ctx, n.inboxNode, fs.StableAttr{Mode: syscall.S_IFDIR})
	n.AddChild(db.InboxDir, childINode, true)
}

func (n *rootNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	rootLogger.Printf("Forwarding %s to the inbox", name)
	childInode, inboxFh, sysFlags, errno := n.inboxNode.Create(ctx, name, flags, mode, out)
	if errno != 0 {
		return nil, nil, 0, errno
	}

	wrappedFh := &rootFileHandle{
		FileHandle: inboxFh,
		rootNode:   n,
		name:       name,
	}

	return childInode, wrappedFh, sysFlags, 0
}

func (n *rootNode) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	rootLogger.Printf("Forwarding %s to the inbox", name)
	childInode, errno := n.inboxNode.Mkdir(ctx, name, mode, out)
	if errno != 0 {
		return nil, errno
	}

	out.EntryValid = 0
	out.AttrValid = 0
	go func() {
		n.RmChild(name)
		errno = n.NotifyEntry(name)
		if errno != 0 {
			rootLogger.Printf("Error notifying entry %q: %v", name, errno)
		}
	}()

	return childInode, 0
}

type rootFileHandle struct {
	fs.FileHandle
	rootNode *rootNode
	name     string
}

func (fh *rootFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	if writer, ok := fh.FileHandle.(fs.FileWriter); ok {
		return writer.Write(ctx, data, off)
	}
	return 0, syscall.ENOSYS
}

func (fh *rootFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if reader, ok := fh.FileHandle.(fs.FileReader); ok {
		return reader.Read(ctx, dest, off)
	}
	return nil, syscall.ENOSYS
}

func (fh *rootFileHandle) Flush(ctx context.Context) syscall.Errno {
	if flusher, ok := fh.FileHandle.(fs.FileFlusher); ok {
		return flusher.Flush(ctx)
	}
	return 0
}

func (fh *rootFileHandle) Allocate(ctx context.Context, off uint64, size uint64, mode uint32) syscall.Errno {
	if allocator, ok := fh.FileHandle.(fs.FileAllocater); ok {
		return allocator.Allocate(ctx, off, size, mode)
	}
	return syscall.ENOSYS
}

func (fh *rootFileHandle) Release(ctx context.Context) syscall.Errno {
	rootLogger.Printf("Releasing %s", fh.name)
	var err syscall.Errno
	if releaser, ok := fh.FileHandle.(fs.FileReleaser); ok {
		err = releaser.Release(ctx)
	}

	go func() {
		fh.rootNode.RmChild(fh.name)
		errno := fh.rootNode.NotifyEntry(fh.name)
		if errno != 0 {
			rootLogger.Printf("Error notifying entry %q after release: %v", fh.name, errno)
		}
	}()

	return err
}

var _ = (fs.NodeMkdirer)((*passthroughNode)(nil))
var _ = (fs.NodeCreater)((*rootNode)(nil))

var _ fs.FileWriter = (*rootFileHandle)(nil)
var _ fs.FileReader = (*rootFileHandle)(nil)
var _ fs.FileFlusher = (*rootFileHandle)(nil)
var _ fs.FileAllocater = (*rootFileHandle)(nil)
var _ fs.FileReleaser = (*rootFileHandle)(nil)
