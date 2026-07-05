package logic

import (
	"fmt"

	"github.com/noa-santo/tagfs/internal/config"
)

type EffectiveDir struct {
	Path string
	Tags map[string]struct{}
}

func getEffectiveDirs() []EffectiveDir {
	c := config.Get()
	var result []EffectiveDir
	for _, dir := range c.Directories {
		flattenDirs(dir, "", make(map[string]struct{}), &result, 0)
	}
	return result
}

func flattenDirs(dir config.DirectoryConfig, parentPath string, parentTags map[string]struct{}, result *[]EffectiveDir, level int) {
	currentPath := dir.Name
	if parentPath != "" {
		currentPath = parentPath + "/" + dir.Name
	}

	currentTags := make(map[string]struct{})
	for tag := range parentTags {
		if tag == fmt.Sprintf("level:%d", level-1) {
			continue
		}
		currentTags[tag] = struct{}{}
	}

	dir.Tags = append(dir.Tags, fmt.Sprintf("level:%d", level))
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
		flattenDirs(subDir, currentPath, currentTags, result, level+1)
	}
}
