package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/noa-santo/tagfs/internal/config"
	"github.com/noa-santo/tagfs/internal/db/gen"
	"golang.org/x/sys/unix"
)

const (
	metadataFile    = ".tagfs-meta.json"
	metadataXattr   = "user.tagfs.metadata"
	metadataVersion = 1
)

// NodeMetadata is deliberately self-contained. It is the recovery record for
// a node, not merely a cache: the store can be rebuilt from these records if
// tagfs.db is deleted or corrupted.
type NodeMetadata struct {
	Version  int      `json:"version"`
	ID       string   `json:"id"`
	OrigName string   `json:"name"`
	Mode     int64    `json:"mode"`
	Tags     []string `json:"tags,omitempty"`
}

func metadataPath(id string) string {
	return filepath.Join(config.Get().StoragePath, ".data", id, metadataFile)
}

// WriteNodeMetadata atomically publishes one recovery record. The xattr is a
// useful second copy, but failure is non-fatal because some filesystems mount
// without user xattrs; the sidecar is the authoritative portable copy.
func WriteNodeMetadata(m NodeMetadata) error {
	if m.Version == 0 {
		m.Version = metadataVersion
	}
	if err := os.MkdirAll(filepath.Dir(metadataPath(m.ID)), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(metadataPath(m.ID)), ".tagfs-meta-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, metadataPath(m.ID)); err != nil {
		return err
	}
	if dir, e := os.Open(filepath.Dir(metadataPath(m.ID))); e == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	payload := filepath.Join(filepath.Dir(metadataPath(m.ID)), m.OrigName)
	if err := unix.Lsetxattr(payload, metadataXattr, data, 0); err != nil {
		// xattrs are best effort; sidecar remains available.
		if err != syscall.ENOTSUP && err != syscall.EOPNOTSUPP && err != syscall.EPERM {
			return fmt.Errorf("writing metadata xattr: %w", err)
		}
	}
	return nil
}

func ReadNodeMetadata(path string) (NodeMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NodeMetadata{}, err
	}
	var m NodeMetadata
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if m.Version != metadataVersion || m.ID == "" || m.OrigName == "" {
		return m, fmt.Errorf("invalid tagfs metadata")
	}
	return m, nil
}

// RecoverMetadata is idempotent and intentionally only adds records. It never
// deletes database rows, so a partial recovery cannot destroy information.
func (d *DB) RecoverMetadata() error {
	root := filepath.Join(config.Get().StoragePath, ".data")
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	ctx := d.Ctx
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		m, err := ReadNodeMetadata(filepath.Join(root, entry.Name(), metadataFile))
		if err != nil || m.ID != entry.Name() {
			continue
		}
		if _, err := d.Queries.GetNode(ctx, m.ID); err == nil {
			continue
		}
		if err := d.Queries.InsertNode(ctx, gen.InsertNodeParams{ID: m.ID, OrigName: m.OrigName, Mode: m.Mode}); err != nil {
			return err
		}
		if err := d.UpdateTags(m.ID, m.Tags); err != nil {
			return err
		}
	}
	return nil
}
