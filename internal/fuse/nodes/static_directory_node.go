package nodes

import (
	"github.com/noa-santo/tagfs/internal/config"
)

type staticDirectoryNode struct {
	passthroughNode
	nodeConfig config.DirectoryConfig
}
