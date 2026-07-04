package fuse

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
	"github.com/noa-santo/tagfs/internal/config"
	. "github.com/noa-santo/tagfs/internal/shared"
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
		mimeType, _ := mimetype.DetectFile(filepath.Join(path, entry.Name()))
		entries[i] = InboxEntry{
			Name:       entry.Name(),
			IsDir:      entry.IsDir(),
			ModifiedAt: entryInfo.ModTime().Format("02.01.2006 15:04:05"),
			Size:       entryInfo.Size(),
			MimeType:   mimeType.String(),
		}
	}
	return entries, nil
}

func getInboxEntry(filename string) (InboxEntry, error) {
	path := filepath.Join(config.Get().StoragePath, ".inbox", filename)
	entryInfo, err := os.Stat(path)
	if err != nil {
		inboxLogger.Printf("Error reading inbox: %v", err)
		return InboxEntry{}, err
	}
	mimeType, err := mimetype.DetectFile(path)
	if err != nil {
		inboxLogger.Printf("Error detecting mime type: %v", err)
		return InboxEntry{}, err
	}
	return InboxEntry{
		Name:       filename,
		IsDir:      entryInfo.IsDir(),
		ModifiedAt: entryInfo.ModTime().Format("02.01.2006 15:04:05"),
		Size:       entryInfo.Size(),
		MimeType:   mimeType.String(),
	}, nil
}
