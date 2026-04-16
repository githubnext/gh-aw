package workflow

// loadRepoConfig loads and caches repository-level configuration from aw.json.
func (c *Compiler) loadRepoConfig() (*RepoConfig, error) {
	if c.repoConfigLoaded {
		return c.repoConfig, c.repoConfigErr
	}

	c.repoConfig, c.repoConfigErr = LoadRepoConfig(c.gitRoot)
	c.repoConfigLoaded = true
	return c.repoConfig, c.repoConfigErr
}
