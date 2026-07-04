package logic

import "github.com/noa-santo/tagfs/internal/config"

type EffectiveDir struct {
	Path string
	Tags map[string]struct{}
}

func getEffectiveDirs() []EffectiveDir {
	c := config.Get()
	var result []EffectiveDir
	for _, dir := range c.Directories {
		flattenDirs(dir, "", make(map[string]struct{}), &result)
	}
	return result
}

func flattenDirs(dir config.DirectoryConfig, parentPath string, parentTags map[string]struct{}, result *[]EffectiveDir) {
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

	*result = append(*result, EffectiveDir{
		Path: currentPath,
		Tags: currentTags,
	})

	for _, subDir := range dir.Subdirectories {
		flattenDirs(subDir, currentPath, currentTags, result)
	}
}
