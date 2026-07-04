package config

func (c Config) GetAllTags() []string {
	uniqueTags := make(map[string]struct{})

	for _, dir := range c.Directories {
		c.collectTags(dir, uniqueTags)
	}

	tags := make([]string, 0, len(uniqueTags))
	for tag := range uniqueTags {
		tags = append(tags, tag)
	}

	return tags
}

func (c Config) collectTags(dir DirectoryConfig, uniqueTags map[string]struct{}) {
	for _, tag := range dir.Tags {
		if tag != "" {
			uniqueTags[tag] = struct{}{}
		}
	}

	for _, subDir := range dir.Subdirectories {
		c.collectTags(subDir, uniqueTags)
	}
}

func (c Config) IsTagCompatible(newTag string, existingTags []string) bool {
	requiredTags := make(map[string]struct{})
	for _, t := range existingTags {
		requiredTags[t] = struct{}{}
	}
	requiredTags[newTag] = struct{}{}
	effectiveDirs := c.getEffectiveDirs()
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

func (c Config) GetImplicitTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	requiredTags := make(map[string]struct{})
	for _, t := range tags {
		requiredTags[t] = struct{}{}
	}

	effectiveDirs := c.getEffectiveDirs()
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

		fullTags := make([]string, 0, len(targetDir.Tags))
		for tag := range targetDir.Tags {
			fullTags = append(fullTags, tag)
		}

		return fullTags
	}

	return nil
}
