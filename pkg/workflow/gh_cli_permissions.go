package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed data/gh_cli_permissions.json
var ghCLIPermissionsJSON []byte

// ghCLISubcommandGroup maps a gh subcommand group (e.g. "pr", "issue") to its permissions.
type ghCLISubcommandGroup struct {
	Description      string   `json:"description"`
	ReadSubcommands  []string `json:"read_subcommands"`
	WriteSubcommands []string `json:"write_subcommands"`
	ReadPermissions  []string `json:"read_permissions"`
	WritePermissions []string `json:"write_permissions"`
}

// ghCLIAPIPathPattern maps a REST API path pattern to the required permissions.
type ghCLIAPIPathPattern struct {
	Pattern     string   `json:"pattern"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// ghCLIPermissionsData is the top-level structure of gh_cli_permissions.json.
type ghCLIPermissionsData struct {
	Version          string                          `json:"version"`
	Description      string                          `json:"description"`
	SubcommandGroups map[string]ghCLISubcommandGroup `json:"subcommand_groups"`
	APIPathPatterns  []ghCLIAPIPathPattern            `json:"api_path_patterns"`
}

// compiledGHCLIPermissions holds pre-compiled lookup data built from the JSON.
type compiledGHCLIPermissions struct {
	// readCommands maps "group action" (e.g. "pr diff") to read permission scopes.
	readCommands map[string][]PermissionScope
	// writeCommands maps "group action" (e.g. "pr create") to write permission scopes.
	writeCommands map[string][]PermissionScope
	// groupReadPermissions maps a subcommand group name (e.g. "pr") to read permission scopes
	// used as fallback when the specific action is not recognised.
	groupReadPermissions map[string][]PermissionScope
	// apiPathPatterns holds compiled regexps paired with required permission scopes.
	apiPathPatterns []compiledAPIPathPattern
}

type compiledAPIPathPattern struct {
	re          *regexp.Regexp
	permissions []PermissionScope
}

var ghCLIPermissions compiledGHCLIPermissions

func init() {
	var data ghCLIPermissionsData
	if err := json.Unmarshal(ghCLIPermissionsJSON, &data); err != nil {
		panic(fmt.Sprintf("failed to load gh CLI permissions from JSON: %v", err))
	}

	cp := compiledGHCLIPermissions{
		readCommands:         make(map[string][]PermissionScope),
		writeCommands:        make(map[string][]PermissionScope),
		groupReadPermissions: make(map[string][]PermissionScope),
	}

	for group, sg := range data.SubcommandGroups {
		readPerms := make([]PermissionScope, len(sg.ReadPermissions))
		for i, p := range sg.ReadPermissions {
			readPerms[i] = PermissionScope(p)
		}
		writePerms := make([]PermissionScope, len(sg.WritePermissions))
		for i, p := range sg.WritePermissions {
			writePerms[i] = PermissionScope(p)
		}

		// Store group-level fallback (used when specific action is unknown).
		cp.groupReadPermissions[group] = readPerms

		for _, action := range sg.ReadSubcommands {
			key := group + " " + action
			cp.readCommands[key] = readPerms
		}
		for _, action := range sg.WriteSubcommands {
			key := group + " " + action
			cp.writeCommands[key] = writePerms
		}
	}

	for _, ap := range data.APIPathPatterns {
		re, err := regexp.Compile(ap.Pattern)
		if err != nil {
			panic(fmt.Sprintf("invalid gh API path pattern %q in gh_cli_permissions.json: %v", ap.Pattern, err))
		}
		perms := make([]PermissionScope, len(ap.Permissions))
		for i, p := range ap.Permissions {
			perms[i] = PermissionScope(p)
		}
		cp.apiPathPatterns = append(cp.apiPathPatterns, compiledAPIPathPattern{re: re, permissions: perms})
	}

	ghCLIPermissions = cp
}

// ghSubcommandRE matches invocations of the gh CLI followed by a known subcommand group
// and an action word.  It is designed to handle both:
//   - simple single-line calls  (gh pr diff "$PR" --name-only)
//   - multi-line shell scripts where the `gh` binary may be preceded by whitespace.
//
// Capture groups: (1) subcommand group, (2) action word.
// The trailing `\b` ensures the action word is complete (avoids matching partial words).
var ghSubcommandRE = regexp.MustCompile(`(?m)(?:^|[\s|;])gh\s+(pr|issue|release|workflow|run|cache|repo|label)\s+([\w][\w-]*)\b`)

// ghAPIRE matches `gh api <path>` invocations.
// Capture group: (1) API path (up to the first whitespace, pipe, or quote).
var ghAPIRE = regexp.MustCompile(`(?m)(?:^|[\s|;])gh\s+api\s+([^\s|;&"'\\]+)`)

// inferPermissionsFromShellScripts scans one or more shell script strings for
// gh CLI invocations and returns the minimum set of GitHub Actions permissions
// required to run those commands.
//
// Only read-level permissions are inferred here; write-level operations are
// intentionally not auto-escalated. Use detectWriteCommandsInShellScripts to
// surface write commands as validation errors.
func inferPermissionsFromShellScripts(scripts []string) map[PermissionScope]PermissionLevel {
	perms := make(map[PermissionScope]PermissionLevel)

	for _, script := range scripts {
		// Match gh <group> <action> patterns.
		for _, m := range ghSubcommandRE.FindAllStringSubmatch(script, -1) {
			group := strings.ToLower(m[1])
			action := strings.ToLower(m[2])
			key := group + " " + action

			// Check explicit read mapping first.
			if readPerms, ok := ghCLIPermissions.readCommands[key]; ok {
				for _, scope := range readPerms {
					if _, exists := perms[scope]; !exists {
						perms[scope] = PermissionRead
					}
				}
				continue
			}
			// Write commands only need read-level permissions in the activation job context.
			// (Full write escalation is rejected by detectWriteCommandsInShellScripts instead.)
			if readPerms, ok := ghCLIPermissions.writeCommands[key]; ok {
				for _, scope := range readPerms {
					if _, exists := perms[scope]; !exists {
						perms[scope] = PermissionRead
					}
				}
				continue
			}
			// Fall back to group-level read permissions for unrecognised actions.
			if readPerms, ok := ghCLIPermissions.groupReadPermissions[group]; ok {
				for _, scope := range readPerms {
					if _, exists := perms[scope]; !exists {
						perms[scope] = PermissionRead
					}
				}
			}
		}

		// Match gh api <path> patterns.
		for _, m := range ghAPIRE.FindAllStringSubmatch(script, -1) {
			path := m[1]
			for _, ap := range ghCLIPermissions.apiPathPatterns {
				if ap.re.MatchString(path) {
					for _, scope := range ap.permissions {
						if _, exists := perms[scope]; !exists {
							perms[scope] = PermissionRead
						}
					}
				}
			}
		}
	}

	return perms
}

// detectWriteCommandsInShellScripts returns all write gh CLI commands found in the
// given scripts, formatted as "gh <group> <action>" (e.g. "gh pr create").
// The slice contains no duplicates and is sorted deterministically in discovery order.
func detectWriteCommandsInShellScripts(scripts []string) []string {
	var found []string
	seen := make(map[string]struct{})

	for _, script := range scripts {
		for _, m := range ghSubcommandRE.FindAllStringSubmatch(script, -1) {
			group := strings.ToLower(m[1])
			action := strings.ToLower(m[2])
			key := group + " " + action

			if _, isWrite := ghCLIPermissions.writeCommands[key]; isWrite {
				cmd := "gh " + key
				if _, already := seen[cmd]; !already {
					seen[cmd] = struct{}{}
					found = append(found, cmd)
				}
			}
		}
	}

	return found
}

// extractRunScriptsFromJobPreSteps returns the `run` script text from every
// pre-step in the named job configuration inside the frontmatter jobs map.
//
// It is a read-only extraction: it never mutates the jobs map.
func extractRunScriptsFromJobPreSteps(jobs map[string]any, jobName string) []string {
	if len(jobs) == 0 {
		return nil
	}

	jobConfig, ok := jobs[jobName]
	if !ok {
		return nil
	}

	configMap, ok := jobConfig.(map[string]any)
	if !ok {
		return nil
	}

	raw, ok := configMap["pre-steps"]
	if !ok {
		return nil
	}

	stepsList, ok := raw.([]any)
	if !ok {
		return nil
	}

	var scripts []string
	for _, step := range stepsList {
		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}
		if runVal, ok := stepMap["run"].(string); ok && runVal != "" {
			scripts = append(scripts, runVal)
		}
	}
	return scripts
}
