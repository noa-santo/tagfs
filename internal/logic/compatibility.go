package logic

import (
	"os"
	"slices"

	"github.com/gabriel-vasile/mimetype"
	"github.com/noa-santo/tagfs/internal/config"
)

// NodeMatchesRules is shared by the dynamic directory and inbox views. This
// prevents a node with stale/manual tags from disappearing: if it no longer
// satisfies a forced rule, it is treated as inbox content instead.
func NodeMatchesRules(path, name string, isDir bool, rules config.Rules) bool {
	if rules.ForceNamePattern && !MatchesNamePattern(name, rules.NamePatterns) {
		return false
	}
	if isDir || len(rules.MimeTypes) == 0 || !rules.ForceMimeTypes {
		return true
	}
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	mime, err := mimetype.DetectFile(path)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(rules.MimeTypes, mime.Is)
}
