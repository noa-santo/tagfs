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
	childInode, fh, sysFlags, errno := n.inboxNode.Create(ctx, name, flags, mode, out)
	if errno != 0 {
		return nil, nil, 0, errno
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
	return childInode, fh, sysFlags, 0
}

var _ = (fs.NodeCreater)((*rootNode)(nil))
