package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/noa-santo/tagfs/internal/config"
)

func TestNodeMetadataRoundTripAndAtomicRecord(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "config.json")
	configJSON := map[string]string{
		"storage_path": filepath.Join(root, "storage"),
		"mount_path": filepath.Join(root, "mount"),
		"inbox_dir": "Inbox",
	}
	data, _ := json.Marshal(configJSON)
	if err := os.WriteFile(cfg, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.InitConfig(cfg); err != nil {
		t.Fatal(err)
	}

	id := "01TESTMETADATA00000000000000"
	payloadDir := filepath.Join(config.Get().StoragePath, ".data", id)
	if err := os.MkdirAll(payloadDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(payloadDir, "file.txt"), []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	want := NodeMetadata{ID: id, OrigName: "file.txt", Mode: 0100644, Tags: []string{"documents"}}
	if err := WriteNodeMetadata(want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadNodeMetadata(metadataPath(id))
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.OrigName != want.OrigName || got.Mode != want.Mode || len(got.Tags) != 1 || got.Tags[0] != "documents" {
		t.Fatalf("metadata mismatch: got %#v want %#v", got, want)
	}
}
