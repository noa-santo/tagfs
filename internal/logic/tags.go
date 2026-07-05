package logic

import (
	"fmt"

	"github.com/noa-santo/tagfs/internal/config"
)

func GetAllTags() []string {
	c := config.Get()
	uniqueTags := make(map[string]struct{})

	for _, dir := range c.Directories {
		collectTags(dir, uniqueTags, 0)
	}

	tags := make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		tags = append(tags, tag)
	}

	return tags
}

func collectTags(dir config.DirectoryConfig, uniqueTags map[string]struct{}, level int) {
	dir.Tags = append(dir.Tags, fmt.Sprintf("level:%d", level))
	for _, tag := range dir.Tags {
		if tag != "" {
			uniqueTags[tag] = struct{}{}
		}
	}

	for _, subDir := range dir.Subdirectories {
		collectTags(subDir, uniqueTags, level+1)
	}
}

func IsTagCompatible(newTag string, existingTags []string) bool {
	requiredTags := make(map[string]struct{})
	for _, t := range existingTags {
		requiredTags[t] = struct{}{}
	}
	requiredTags[newTag] = struct{}{}
	effectiveDirs := getEffectiveDirs()
	for _, dir := range effectiveDirs {
		isValidForDir := true
		for reqTag := range requiredTags {
			if _, exists := dir.Tags[reqTag]; !exists {
				isValidForDir = false
				break
			}
		}

		if isValidForDir {
			return true
		}
	}
	return false
}

func GetValidDestinations(tags []string) ([]EffectiveDir, map[string]struct{}) {
	if len(tags) == 0 {
		return nil, nil
	}

	requiredTags := make(map[string]struct{})
	for _, t := range tags {
		requiredTags[t] = struct{}{}
	}

	effectiveDirs := getEffectiveDirs()
	var validDestinations []EffectiveDir

	for _, dir := range effectiveDirs {
		isValidForDir := true
		for reqTag := range requiredTags {
			if _, exists := dir.Tags[reqTag]; !exists {
				isValidForDir = false
				break
			}
		}

		if isValidForDir {
			validDestinations = append(validDestinations, dir)
		}
	}

	return validDestinations, requiredTags
}

func GetTargetDestination(tags []string) (EffectiveDir, bool) {
	validDestinations, _ := GetValidDestinations(tags)
	if len(validDestinations) == 1 {
		return validDestinations[0], true
	}
	return EffectiveDir{}, false
}

func GetImplicitTags(tags []string) []string {
	validDestinations, requiredTags := GetValidDestinations(tags)

	if len(validDestinations) == 1 {
		targetDir := validDestinations[0]

		var implicitTags []string
		for tag := range targetDir.Tags {
			if _, isOriginal := requiredTags[tag]; !isOriginal {
				implicitTags = append(implicitTags, tag)
			}
		}

		return implicitTags
	}

	return nil
}
