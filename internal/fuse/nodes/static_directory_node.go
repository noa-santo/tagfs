package nodes

import (
	"context"
	"log"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db"
)

var staticDirLogger = log.New(os.Stdout, "STATIC_DIR", log.LstdFlags)

type staticDirectoryNode struct {
	passthroughNode
	nodeConfig config.DirectoryConfig
}

func (n *staticDirectoryNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	files, err := db.Get().GetFilesForDir(ctx, n.nodeConfig.Tags)
	if err != nil {
		staticDirLogger.Fatalf("GetFilesForDir: %v", err)
	}

	var result []fuse.DirEntry
	for _, file := range files {
		result = append(result, fuse.DirEntry{
			Name: file.OrigName,
			Ino:  0,
			Mode: uint32(file.Mode),
		})
	}

	return fs.NewListDirStream(result), fs.OK
}
