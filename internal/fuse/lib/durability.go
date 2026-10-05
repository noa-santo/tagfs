package lib

import (
	"log"
	"os"

	"github.com/noa-santo/tagfs/internal/db"
	"golang.org/x/sys/unix"
)

var logger = log.New(os.Stdout, "DURABILITY: ", log.LstdFlags|log.Lmicroseconds)

// SyncPath makes a completed filesystem operation survive a crash.  Directory
// fsync is required on Linux for the directory entry (not just file contents)
// to be durable.
func SyncPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logger.Printf("Error while syncing path %s: %v", path, err)
		}
	}(f)
	return f.Sync()
}

func WriteNodeMetadata(id, name string, mode int64, tags []string) error {
	return db.WriteNodeMetadata(db.NodeMetadata{ID: id, OrigName: name, Mode: mode, Tags: tags})
}

func SyncFile(f *os.File) error {
	if err := f.Sync(); err != nil {
		return err
	}
	return unix.Fsync(int(f.Fd()))
}
