package workflow

// loadRepoConfig loads and caches repository-level configuration from aw.json.
func (c *Compiler) loadRepoConfig() (*RepoConfig, error) {
	if c.repoConfigLoaded {
		repoConfigLog.Print("loadRepoConfig: returning cached repo config")
		return c.repoConfig, c.repoConfigErr
	}

	repoConfigLog.Printf("loadRepoConfig: loading repo config from git root: %s", c.gitRoot)
	c.repoConfig, c.repoConfigErr = LoadRepoConfig(c.gitRoot)
	c.repoConfigLoaded = true
	if c.repoConfigErr != nil {
		repoConfigLog.Printf("loadRepoConfig: failed to load repo config: %v", c.repoConfigErr)
	} else {
		repoConfigLog.Print("loadRepoConfig: repo config loaded successfully")
	}
	return c.repoConfig, c.repoConfigErr
}
