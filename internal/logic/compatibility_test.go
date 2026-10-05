package logic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noa-santo/tagfs/internal/config"
)

func TestNodeMatchesRules(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	rules := config.Rules{MimeTypes: []string{"text/plain"}, ForceMimeTypes: true}
	if !NodeMatchesRules(path, "note.txt", false, rules) {
		t.Fatal("plain text file should satisfy forced mime rule")
	}
	bad := config.Rules{MimeTypes: []string{"image/png"}, ForceMimeTypes: true}
	if NodeMatchesRules(path, "note.txt", false, bad) {
		t.Fatal("plain text file must not satisfy image-only rule")
	}
	name := config.Rules{NamePatterns: []string{`^Screenshot `}, ForceNamePattern: true}
	if NodeMatchesRules(path, "note.txt", false, name) {
		t.Fatal("file with incompatible forced name must be rejected")
	}
}
