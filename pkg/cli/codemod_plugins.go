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
//     `dependencies:` and `github-token` is preserved as-is in the object form.
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
// lines into a `dependencies:` block, handling both array format and object format.
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
// is in object format.  The `repos:` items become `packages:` children under
// `dependencies:`; the `github-token:` key is preserved as-is.
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
// It produces a `dependencies:` object block with `packages:` (renamed from `repos:`)
// and preserves the `github-token:` key so APM can authenticate with the same token.
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

		// Detect repos: sub-key → rename to packages:.
		if strings.HasPrefix(trimmed, "repos:") {
			inRepos = true
			reposIndent = getIndentation(line)
			// Rename repos: → packages: preserving indentation.
			after := strings.TrimPrefix(trimmed, "repos:")
			result = append(result, getIndentation(line)+"packages:"+after)
			continue
		}

		// Preserve github-token: line (APM supports it for token authentication).
		if strings.HasPrefix(trimmed, "github-token:") {
			pluginsCodemodLog.Print("Preserving 'github-token' in dependencies object format")
			inRepos = false
			result = append(result, line)
			continue
		}

		// If we're inside the repos: block, keep items with the same indentation.
		if inRepos && isNestedUnder(line, reposIndent) {
			result = append(result, line)
			continue
		}

		// Any other top-level sub-key of the old plugins: block – preserve it.
		inRepos = false
		result = append(result, line)
	}

	return result, true
}
