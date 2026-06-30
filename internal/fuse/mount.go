package fuse

import (
	"log"
	"os"

	"github.com/hanwen/go-fuse/v2/fs"
)

// StartDaemon initializes the FUSE mount
func StartDaemon(storagePath, mountPoint string) {
	root, err := fs.NewLoopbackRoot(storagePath)
	if err != nil {
		log.Fatal(err)
	}

	logger := log.New(os.Stderr, "FUSE", log.LstdFlags|log.Lmicroseconds)

	opts := &fs.Options{
		Logger: logger,
	}
	_, err = fs.Mount(mountPoint, root, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	logger.Printf("Mounted %s at %s", storagePath, mountPoint)
}
