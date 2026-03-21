package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var pluginsCodemodLog = logger.New("cli:codemod_plugins")

// getPluginsToDependenciesCodemod creates a codemod that migrates the top-level
// `plugins:` field to `dependencies:`.  The `plugins:` field has been removed in
// favour of `dependencies:` backed by Microsoft/apm, which provides cross-agent
// support for skills, prompts, instructions, and plugins (including the Claude
// plugin.json format).
//
// Migration rules:
//   - Array format  →  the same list is moved to `dependencies:`
//   - Object format (repos + github-token)  →  the repos list is moved to
//     `dependencies:`; the `github-token` key is dropped because APM uses
//     `github-app:` for cross-org private package access instead.
func getPluginsToDependenciesCodemod() Codemod {
	return Codemod{
		ID:           "plugins-to-dependencies",
		Name:         "Migrate plugins to dependencies",
		Description:  "Renames the top-level 'plugins' field to 'dependencies'. The plugins feature has been removed in favour of 'dependencies' (Microsoft/apm), which provides cross-agent support.",
		IntroducedIn: "1.0.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			_, hasPlugins := frontmatter["plugins"]
			if !hasPlugins {
				return content, false, nil
			}

			// Skip if dependencies already exist – avoid clobbering user config.
			_, hasDeps := frontmatter["dependencies"]
			if hasDeps {
				pluginsCodemodLog.Print("Both 'plugins' and 'dependencies' exist – skipping migration to avoid overwriting existing dependencies")
				return content, false, nil
			}

			return applyFrontmatterLineTransform(content, func(lines []string) ([]string, bool) {
				return migratePluginsToDependencies(lines)
			})
		},
	}
}

// migratePluginsToDependencies rewrites the `plugins:` block in the frontmatter
// lines into a `dependencies:` block, dropping the `github-token` sub-key when
// the object format is used.
func migratePluginsToDependencies(lines []string) ([]string, bool) {
	// Locate the plugins: key and determine the extent of its block.
	pluginsIdx := -1
	for i, line := range lines {
		if isTopLevelKey(line) && strings.HasPrefix(strings.TrimSpace(line), "plugins:") {
			pluginsIdx = i
			break
		}
	}
	if pluginsIdx == -1 {
		return lines, false
	}

	pluginsIndent := getIndentation(lines[pluginsIdx])

	// Find the end of the plugins block (exclusive).
	blockEnd := pluginsIdx + 1
	for blockEnd < len(lines) {
		line := lines[blockEnd]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			blockEnd++
			continue
		}
		if isNestedUnder(line, pluginsIndent) {
			blockEnd++
			continue
		}
		break
	}

	// Collect the original block lines and rewrite them.
	block := lines[pluginsIdx:blockEnd]
	rewritten, changed := rewritePluginsBlock(block)
	if !changed {
		return lines, false
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:pluginsIdx]...)
	result = append(result, rewritten...)
	result = append(result, lines[blockEnd:]...)
	pluginsCodemodLog.Print("Migrated 'plugins' to 'dependencies'")
	return result, true
}

// rewritePluginsBlock transforms the lines of a plugins block into a
// dependencies block, handling both array format and object format.
//
// Object format detection: if the block contains a `repos:` sub-key the input
// is in object format.  Lines with `github-token:` are dropped; the `repos:`
// line and its children become top-level `dependencies:` children.
func rewritePluginsBlock(block []string) ([]string, bool) {
	if len(block) == 0 {
		return block, false
	}

	// Rename the key on the first line.
	firstLine := block[0]
	trimmedFirst := strings.TrimSpace(firstLine)
	indent := getIndentation(firstLine)

	// Determine whether this is object format by scanning for a `repos:` sub-key.
	isObjectFormat := false
	for _, line := range block[1:] {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "repos:") {
			isObjectFormat = true
			break
		}
	}

	if isObjectFormat {
		return rewriteObjectFormatPlugins(block, indent)
	}

	// Array format – just rename the key and keep the body.
	after := strings.TrimPrefix(trimmedFirst, "plugins:")
	newFirst := indent + "dependencies:" + after

	result := make([]string, len(block))
	result[0] = newFirst
	copy(result[1:], block[1:])
	return result, true
}

// rewriteObjectFormatPlugins handles the object format:
//
//	plugins:
//	  repos:
//	    - org/repo
//	  github-token: ${{ secrets.TOKEN }}
//
// It flattens the repos children directly under `dependencies:` and drops the
// `github-token` line (APM uses `github-app:` for private repo access instead).
func rewriteObjectFormatPlugins(block []string, indent string) ([]string, bool) {
	var result []string
	result = append(result, indent+"dependencies:")

	inRepos := false
	reposIndent := ""

	for _, line := range block[1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Detect repos: sub-key.
		if strings.HasPrefix(trimmed, "repos:") {
			inRepos = true
			reposIndent = getIndentation(line)
			// If repos has an inline value (unlikely), keep it.
			after := strings.TrimPrefix(trimmed, "repos:")
			after = strings.TrimSpace(after)
			if after != "" {
				// Inline list – rare but handle it.
				result = append(result, indent+"  "+after)
			}
			continue
		}

		// Drop github-token line.
		if strings.HasPrefix(trimmed, "github-token:") {
			pluginsCodemodLog.Print("Dropping 'github-token' from plugins object format (use 'github-app:' for private package access)")
			inRepos = false
			continue
		}

		// If we're inside the repos: block, re-indent items one level under dependencies:.
		if inRepos && isNestedUnder(line, reposIndent) {
			// Re-indent to be a direct child of dependencies:.
			result = append(result, indent+"  "+trimmed)
			continue
		}

		// Any other top-level sub-key of the old plugins: block – drop it.
		inRepos = false
	}

	return result, true
}
