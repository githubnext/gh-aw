package workflow

import (
	"errors"
	"fmt"
)

func (c *Compiler) requireWebhookTriggers(roles []string) error {
	config, err := c.loadRepoConfig()
	if err != nil {
		return fmt.Errorf("cannot enable hosted CAO triggers: %w", err)
	}
	if config == nil || !config.WebhookGateway {
		return fmt.Errorf("webhook triggers are experimental synthetic webhook events that must be triggered by a hosted CAO server; enable them by setting {\"webhook-gateway\":true} in %s and defining nested triggers under on.gateway-dispatch in the workflow", RepoConfigFileName)
	}
	return validateWebhookTriggerRoles(roles)
}

func validateWebhookTriggerRoles(roles []string) error {
	const message = "webhook triggers require on.roles with write access or higher; use [admin, maintainer, write] or a stricter subset"
	if len(roles) == 0 {
		return errors.New(message)
	}
	for _, role := range roles {
		switch role {
		case "admin", "maintainer", "maintain", "write":
		default:
			return fmt.Errorf("%s; role %q is not allowed", message, role)
		}
	}
	return nil
}
