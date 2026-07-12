package logic

import (
	"regexp"

	"github.com/noa-santo/tagfs/internal/config"
	. "github.com/noa-santo/tagfs/internal/shared"
)

type TagSuggestion struct {
	// CommonTags are the tags guaranteed to apply, shared by all top valid destinations.
	CommonTags []string
	// Options represent mutually exclusive paths. Each sub-slice contains the
	// additional tags needed to route the file to a specific valid destination.
	Options [][]string
}

type EffectiveRuleDir struct {
	Path  string
	Tags  map[string]struct{}
	Rules config.Rules
}

func MatchesNamePattern(name string, patterns []string) bool {
	if len(patterns) > 0 {
		for _, pattern := range patterns {
			matched, err := regexp.MatchString(pattern, name)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

func SuggestTags(entry InboxEntry) *TagSuggestion {
	dirs := getEffectiveRuleDirs()

	type match struct {
		dir   EffectiveRuleDir
		score int
	}

	var validMatches []match
	maxScore := -1

	for _, d := range dirs {
		if entry.IsDir {
			if !d.Rules.AllowSubdirCreation {
				continue
			}
		} else {
			if !d.Rules.AllowFileCreation {
				continue
			}
		}

		mimeMatch := false
		if !entry.IsDir && len(d.Rules.MimeTypes) > 0 {
			inList := false
			for _, m := range d.Rules.MimeTypes {
				if m == entry.MimeType {
					inList = true
					break
				}
			}

			if d.Rules.ForceMimeTypes && !inList {
				continue
			}
			if inList {
				mimeMatch = true
			}
		}

		nameMatch := MatchesNamePattern(entry.Name, d.Rules.NamePatterns)
		if d.Rules.ForceNamePattern && !nameMatch {
			continue
		}

		score := 0
		if mimeMatch {
			score++
		}
		if nameMatch {
			score++
		}

		validMatches = append(validMatches, match{dir: d, score: score})
		if score > maxScore {
			maxScore = score
		}
	}

	if len(validMatches) == 0 {
		return nil
	}

	var topScorers []EffectiveRuleDir
	for _, m := range validMatches {
		if m.score == maxScore {
			topScorers = append(topScorers, m.dir)
		}
	}

	return computeSuggestionGroups(topScorers)
}

func computeSuggestionGroups(topScorers []EffectiveRuleDir) *TagSuggestion {
	commonMap := make(map[string]struct{})
	for i, ts := range topScorers {
		if i == 0 {
			for tag := range ts.Tags {
				commonMap[tag] = struct{}{}
			}
		} else {
			for tag := range commonMap {
				if _, exists := ts.Tags[tag]; !exists {
					delete(commonMap, tag)
				}
			}
		}
	}

	var commonTags []string
	for tag := range commonMap {
		commonTags = append(commonTags, tag)
	}

	var options [][]string
	for _, ts := range topScorers {
		var opt []string
		for tag := range ts.Tags {
			if _, isCommon := commonMap[tag]; !isCommon {
				opt = append(opt, tag)
			}
		}
		options = append(options, opt)
	}

	return &TagSuggestion{
		CommonTags: commonTags,
		Options:    options,
	}
}

func getEffectiveRuleDirs() []EffectiveRuleDir {
	var result []EffectiveRuleDir
	for _, dir := range config.Get().Directories {
		flattenRuleDirs(dir, "", make(map[string]struct{}), &result)
	}
	return result
}

func flattenRuleDirs(dir config.DirectoryConfig, parentPath string, parentTags map[string]struct{}, result *[]EffectiveRuleDir) {
	currentPath := dir.Name
	if parentPath != "" {
		currentPath = parentPath + "/" + dir.Name
	}

	currentTags := make(map[string]struct{})
	for tag := range parentTags {
		currentTags[tag] = struct{}{}
	}
	for _, tag := range dir.Tags {
		if tag != "" {
			currentTags[tag] = struct{}{}
		}
	}

	*result = append(*result, EffectiveRuleDir{
		Path:  currentPath,
		Tags:  currentTags,
		Rules: dir.Rules,
	})

	for _, subDir := range dir.Subdirectories {
		flattenRuleDirs(subDir, currentPath, currentTags, result)
	}
}
