package fuse

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/fuse/nodes"
)

var logger = log.New(os.Stdout, "DAEMON: ", log.LstdFlags|log.Lmicroseconds)

func StartDaemon() {
	opts := &fs.Options{
		Logger: logger,
		MountOptions: fuse.MountOptions{
			Debug: true,
		},
	}
	root := &nodes.RootNode{}
	server, err := fs.Mount(config.Get().MountPath, root, opts)
	if err != nil {
		logger.Fatalf("Mount fail: %v\n", err)
	}
	logger.Printf("Mounted %s at %s", config.Get().StoragePath, config.Get().MountPath)

	root.Init(context.Background())
	go startCommandListener()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Println("Signal received, unmounting...")
		err := server.Unmount()
		if err != nil {
			logger.Fatalf("Fatal error while unmounting: %s", err)
		}
	}()

	server.Wait()
	logger.Println("Daemon shut down.")
}
