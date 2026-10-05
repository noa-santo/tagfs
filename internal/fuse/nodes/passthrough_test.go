package nodes

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// This is intentionally an unmounted test: it exercises the FUSE file-handle
// contract without requiring CAP_SYS_ADMIN in CI. Mount-level tests can be
// added behind TAGFS_FUSE_TEST on hosts that provide a FUSE device.
func TestPassthroughFilePosixIO(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	fh := &passthroughFile{File: f}
	if n, errno := fh.Write(context.Background(), []byte("hello"), 0); errno != 0 || n != 5 {
		t.Fatalf("write: n=%d errno=%v", n, errno)
	}
	if errno := fh.Flush(context.Background()); errno != 0 {
		t.Fatalf("flush: %v", errno)
	}
	buf := make([]byte, 5)
	result, errno := fh.Read(context.Background(), buf, 0)
	data, status := result.Bytes(buf)

	if errno != 0 || status != 0 || string(data) != "hello" {
		t.Fatalf("read: %q errno=%v status=%v", data, errno, status)
	}
	if errno := fh.Release(context.Background()); errno != 0 {
		t.Fatalf("release: %v", errno)
	}
}
