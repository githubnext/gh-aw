package workflow

import "fmt"

func (c *Compiler) requireWebhookTriggers() error {
	config, err := c.loadRepoConfig()
	if err != nil {
		return fmt.Errorf("cannot enable hosted CAO triggers: %w", err)
	}
	if config == nil || !config.WebhookTriggers {
		return fmt.Errorf("on.projects-item and on.security-alert are experimental synthetic webhook events that must be triggered by a hosted CAO server; enable them by setting {\"webhook-triggers\":true} in %s", RepoConfigFileName)
	}
	return nil
}
