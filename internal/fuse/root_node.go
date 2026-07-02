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
	childINode := n.NewPersistentInode(ctx, n.inboxNode, fs.StableAttr{})
	n.AddChild(db.InboxDir, childINode, true)
}

func (n *rootNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	rootLogger.Println("Create", name, flags, mode, out)
	return n.inboxNode.Create(ctx, name, flags, mode, out)
}

var _ = (fs.NodeCreater)((*rootNode)(nil))
