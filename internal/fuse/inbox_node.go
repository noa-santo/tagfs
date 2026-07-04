package fuse

import (
	"log"
	"os"
	"path/filepath"

	"github.com/noa-santo/tagfs/internal/config"
)

var inboxLogger = log.New(os.Stdout, "INBOX NODE: ", 0)

type InboxEntry struct {
	Name       string `json:"name"`
	IsDir      bool   `json:"is_dir"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

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

func getInboxEntries() ([]InboxEntry, error) {
	path := filepath.Join(config.Get().StoragePath, ".inbox")
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		inboxLogger.Printf("Error reading inbox: %v", err)
		return []InboxEntry{}, err
	}
	entries := make([]InboxEntry, len(dirEntries))
	for i, entry := range dirEntries {
		entryInfo, _ := entry.Info()
		entries[i] = InboxEntry{
			entry.Name(),
			entry.IsDir(),
			entryInfo.ModTime().Format("02.01.2006 15:04:05"),
			entryInfo.Size(),
		}
	}
	return entries, nil
}
