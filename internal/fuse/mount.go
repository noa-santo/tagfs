package fuse

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// StartDaemon initializes the FUSE mount
func StartDaemon(storagePath, mountPoint string) {
	logger := log.New(os.Stderr, "FUSE", log.LstdFlags|log.Lmicroseconds)

	opts := &fs.Options{
		Logger: logger,
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	}
	root := &fsNode{}
	server, err := fs.Mount(mountPoint, root, opts)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	logger.Printf("Mounted %s at %s", storagePath, mountPoint)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Signal received, unmounting...")
		err := server.Unmount()
		if err != nil {
			logger.Fatalf("Fatal error while unmounting: %s", err)
		}
	}()

	server.Wait()
	logger.Println("Daemon shut down.")
}
