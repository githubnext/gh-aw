package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func (c *Compiler) validatePlugins(workflowData *WorkflowData) error {
	if workflowData == nil {
		return nil
	}

	for i, plugin := range workflowData.Plugins {
		parsed := parseSkillRefSpec(plugin)
		if !parsed.isRemote || parsed.ref == "" {
			return fmt.Errorf("plugins[%d]: invalid plugin reference %q; expected owner/repository[/path]@ref, for example github/awesome-copilot/plugins/example@v1", i, plugin)
		}
		if looksLikeAmbiguousSHA(parsed.ref) {
			return fmt.Errorf("plugins[%d]: ref %q looks like a truncated or malformed commit SHA; use the full 40-character lowercase SHA or a branch/tag name", i, parsed.ref)
		}
		if !parsed.isFullSHA && (!skillRefCharsRegexp.MatchString(parsed.ref) || strings.Contains(parsed.ref, "..")) {
			return fmt.Errorf("plugins[%d]: ref %q contains unsupported characters; refs may only contain letters, digits, '.', '_', '-', and '/', must start with a letter or digit, and must not contain '..'", i, parsed.ref)
		}
	}
	return nil
}

func (c *Compiler) validatePluginSupport(workflowData *WorkflowData) error {
	if workflowData == nil || len(workflowData.Plugins) == 0 {
		return nil
	}

	engine, err := c.getAgenticEngine(ResolveEngineID(workflowData))
	if err != nil {
		return err
	}
	if !engine.GetCapabilities().Plugins {
		return fmt.Errorf("plugins are not supported by engine %q; remove the plugins field or use an engine with Agent Plugins support. See: %s", engine.GetID(), constants.DocsEnginesURL)
	}
	return nil
}
