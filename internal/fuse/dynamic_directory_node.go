package fuse

import "github.com/noa-santo/tagfs/internal/config"

type dynamicDirectoryNode struct {
	passthroughNode
	nodeConfig config.DirectoryConfig
}
