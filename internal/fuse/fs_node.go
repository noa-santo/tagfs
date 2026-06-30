package fuse

import "github.com/hanwen/go-fuse/v2/fs"

type fsNode struct {
	fs.Inode
}
