package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/semverutil"
)

// hasPathTraversalSegment reports whether repoPath contains a "." or ".."
// path segment, which would let pluginCheckoutSubpath escape the plugin's
// dedicated checkout folder (.gh-aw-plugins/plugin-N) and resolve to an
// unpinned path elsewhere in the workspace.
func hasPathTraversalSegment(repoPath string) bool {
	for segment := range strings.SplitSeq(repoPath, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func (c *Compiler) validatePlugins(workflowData *WorkflowData) error {
	if workflowData == nil {
		return nil
	}

	for i, plugin := range workflowData.Plugins {
		parsed := parseSkillRefSpec(plugin)
		if !parsed.isRemote || parsed.ref == "" {
			return fmt.Errorf("plugins[%d]: invalid plugin reference %q; expected owner/repository[/path]@ref, for example github/awesome-copilot/plugins/example@v1", i, plugin)
		}
		if hasPathTraversalSegment(parsed.repoPath) {
			return fmt.Errorf("plugins[%d]: repository path %q must not contain '.' or '..' segments; expected a plain repository-relative path", i, parsed.repoPath)
		}
		if !parsed.isFullSHA && len(parsed.ref) == 40 && gitutil.IsValidFullSHACaseInsensitive(parsed.ref) {
			return fmt.Errorf("plugins[%d]: ref %q looks like a commit SHA but must be lowercase hexadecimal; expected all-lowercase hex characters, for example a1b2c3", i, parsed.ref)
		}
		if looksLikeAmbiguousSHA(parsed.ref) {
			return fmt.Errorf("plugins[%d]: ref %q looks like a truncated or malformed commit SHA; use the full 40-character lowercase SHA or a branch/tag name", i, parsed.ref)
		}
		if !parsed.isFullSHA && (!skillRefCharsRegexp.MatchString(parsed.ref) || strings.Contains(parsed.ref, "..")) {
			return fmt.Errorf("plugins[%d]: ref %q contains unsupported characters; expected letters, digits, '.', '_', '-', and '/', starting with a letter or digit and not containing '..', for example v1.2.3", i, parsed.ref)
		}
	}

	mergedPlugins, err := mergeValidatedPlugins(workflowData.Plugins)
	if err != nil {
		return err
	}
	workflowData.Plugins = mergedPlugins
	return nil
}

func mergeValidatedPlugins(plugins []string) ([]string, error) {
	merged := make([]string, 0, len(plugins))
	indexByPath := make(map[string]int, len(plugins))

	for _, plugin := range plugins {
		parsed := parseSkillRefSpec(plugin)
		index, exists := indexByPath[parsed.repoPath]
		if !exists {
			indexByPath[parsed.repoPath] = len(merged)
			merged = append(merged, plugin)
			continue
		}

		existing := parseSkillRefSpec(merged[index])
		if existing.ref == parsed.ref {
			continue
		}
		if !semverutil.IsValid(existing.ref) || !semverutil.IsValid(parsed.ref) {
			return nil, fmt.Errorf("plugin %q is declared with conflicting refs %q and %q; use the same ref for every declaration", parsed.repoPath, existing.ref, parsed.ref)
		}
		if !semverutil.IsCompatible(existing.ref, parsed.ref) {
			return nil, fmt.Errorf("plugin %q is declared with incompatible semantic versions %q and %q", parsed.repoPath, existing.ref, parsed.ref)
		}
		if semverutil.Compare(parsed.ref, existing.ref) > 0 {
			merged[index] = plugin
		}
	}

	return merged, nil
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
