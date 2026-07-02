package fuse

import (
	"log"
	"os"
	"path/filepath"

	"github.com/noa-santo/tagfs/internal/config"
)

var inboxLogger = log.New(os.Stdout, "INBOX NODE: ", 0)

type inboxNode struct {
	passthroughNode
}

func newInboxNode() *inboxNode {
	path := filepath.Join(config.Get().StoragePath, ".inbox")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0755); err != nil {
			inboxLogger.Fatal(err)
		}
	}
	return &inboxNode{
		passthroughNode: passthroughNode{
			Path: path,
		},
	}
}
