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
