package workflow

// extractNetworkPermissions extracts network permissions from frontmatter
func (c *Compiler) extractNetworkPermissions(frontmatter map[string]any) *NetworkPermissions {
	if network, exists := frontmatter["network"]; exists {
		// Handle string format: "defaults"
		if networkStr, ok := network.(string); ok {
			if networkStr == "defaults" {
				return &NetworkPermissions{
					Mode: "defaults",
				}
			}
			// Unknown string format, return nil
			return nil
		}

		// Handle object format: { allowed: [...] } or {}
		if networkObj, ok := network.(map[string]any); ok {
			permissions := &NetworkPermissions{}

			// Extract allowed domains if present
			if allowed, hasAllowed := networkObj["allowed"]; hasAllowed {
				if allowedSlice, ok := allowed.([]any); ok {
					for _, domain := range allowedSlice {
						if domainStr, ok := domain.(string); ok {
							permissions.Allowed = append(permissions.Allowed, domainStr)
						}
					}
				}
			}
			// Empty object {} means no network access (empty allowed list)
			return permissions
		}
	}
	return nil
}
