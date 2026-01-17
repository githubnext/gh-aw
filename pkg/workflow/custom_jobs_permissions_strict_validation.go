package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var strictCustomJobsPermissionsLog = logger.New("workflow:strict_custom_jobs_permissions")

// validateStrictCustomJobPermissions refuses write permissions in user-defined custom jobs
// when strict mode is enabled.
//
// Rationale: custom jobs are an escape hatch that could otherwise provide a "raw write" path
// outside of safe-outputs. In strict mode, all writes should be mediated by safe-outputs.
func (c *Compiler) validateStrictCustomJobPermissions(workflowData *WorkflowData) error {
	if !c.strictMode {
		return nil
	}
	if workflowData == nil || len(workflowData.Jobs) == 0 {
		return nil
	}

	for jobName, jobConfig := range workflowData.Jobs {
		configMap, ok := jobConfig.(map[string]any)
		if !ok {
			continue
		}

		permissionsVal, hasPermissions := configMap["permissions"]
		if !hasPermissions {
			continue
		}

		// permissions can be a string (e.g. "read-all"/"write-all") or a map.
		switch v := permissionsVal.(type) {
		case string:
			perm := strings.ToLower(strings.TrimSpace(v))
			if strings.Contains(perm, "write") {
				strictCustomJobsPermissionsLog.Printf("Strict mode: custom job '%s' uses permissions=%q", jobName, v)
				return fmt.Errorf("strict mode: custom job '%s' requests write permissions (%q). Custom jobs must be read-only; use safe-outputs to perform write operations safely", jobName, v)
			}
		case map[string]any:
			for scope, levelVal := range v {
				levelStr, ok := levelVal.(string)
				if !ok {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(levelStr), "write") {
					strictCustomJobsPermissionsLog.Printf("Strict mode: custom job '%s' requests %s: write", jobName, scope)
					return fmt.Errorf("strict mode: custom job '%s' requests '%s: write'. Custom jobs must be read-only; use safe-outputs to perform write operations safely", jobName, scope)
				}
			}
		default:
			// Unknown permissions type; ignore.
		}
	}

	return nil
}
