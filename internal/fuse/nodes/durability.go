package nodes

import (
	"os"

	"golang.org/x/sys/unix"
)

// syncPath makes a completed filesystem operation survive a crash.  Directory
// fsync is required on Linux for the directory entry (not just file contents)
// to be durable.
func syncPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func syncParent(path string) error {
	return syncPath(path)
}

func syncFile(f *os.File) error {
	if err := f.Sync(); err != nil {
		return err
	}
	// Fsync is intentionally kept explicit for callers that need the syscall
	// errno while still allowing ordinary *os.File implementations in tests.
	return unix.Fsync(int(f.Fd()))
}
