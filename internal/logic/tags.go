package logic

import "github.com/noa-santo/tagfs/internal/config"

func GetAllTags() []string {
	c := config.Get()
	uniqueTags := make(map[string]struct{})

	for _, dir := range c.Directories {
		collectTags(dir, uniqueTags)
	}

	tags := make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		tags = append(tags, tag)
	}

	return tags
}

func collectTags(dir config.DirectoryConfig, uniqueTags map[string]struct{}) {
	for _, tag := range dir.Tags {
		if tag != "" {
			uniqueTags[tag] = struct{}{}
		}
	}

	for _, subDir := range dir.Subdirectories {
		collectTags(subDir, uniqueTags)
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

func GetImplicitTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
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
