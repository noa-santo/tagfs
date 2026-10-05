package nodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// TestMountedPassthroughPOSIX performs the same operations through a real
// mounted FUSE filesystem. It is opt-in because ordinary CI containers and
// developer sandboxes usually do not expose /dev/fuse.
func TestMountedPassthroughPOSIX(t *testing.T) {
	if os.Getenv("TAGFS_FUSE_TEST") != "1" {
		t.Skip("set TAGFS_FUSE_TEST=1 to run a real FUSE mount test")
	}
	if _, err := os.Stat("/dev/fuse"); err != nil {
		t.Skipf("FUSE device unavailable: %v", err)
	}

	backing := t.TempDir()
	mountpoint := filepath.Join(t.TempDir(), "mnt")
	if err := os.Mkdir(mountpoint, 0755); err != nil {
		t.Fatal(err)
	}

	server, err := fs.Mount(mountpoint, &passthroughNode{Path: backing}, &fs.Options{
		MountOptions: fuse.MountOptions{Debug: false},
	})
	if err != nil {
		t.Fatalf("mount: %v", err)
	}
	defer func() {
		if err := server.Unmount(); err != nil {
			t.Errorf("unmount: %v", err)
		}
	}()

	file := filepath.Join(mountpoint, "file.txt")
	if err := os.WriteFile(file, []byte("hello"), 0640); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(file)
	if err != nil || string(got) != "hello" {
		t.Fatalf("read through FUSE: %q, %v", got, err)
	}
	if err := os.Chmod(file, 0600); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := os.Rename(file, filepath.Join(mountpoint, "renamed.txt")); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := os.Mkdir(filepath.Join(mountpoint, "dir"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("renamed.txt", filepath.Join(mountpoint, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(mountpoint, "link")); err != nil || target != "renamed.txt" {
		t.Fatalf("readlink: %q, %v", target, err)
	}
	if err := os.Remove(filepath.Join(mountpoint, "link")); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if err := os.Remove(filepath.Join(mountpoint, "renamed.txt")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Remove(filepath.Join(mountpoint, "dir")); err != nil {
		t.Fatalf("rmdir: %v", err)
	}
}
