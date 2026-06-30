package fuse

import (
	"context"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/noa-santo/tagfs/internal/db"
)

type fsNode struct {
	fs.Inode
}

func (n *fsNode) initPassthrough(ctx context.Context) {
	for _, dirName := range db.PassthroughDirs {
		config, _ := db.LoadConfig()
		path := filepath.Join(config.StoragePath, dirName)
		childNode := &PassthroughNode{Path: path}
		childINode := n.NewPersistentInode(ctx, childNode, fs.StableAttr{Mode: syscall.S_IFDIR})
		n.AddChild(dirName, childINode, true)
	}
}
