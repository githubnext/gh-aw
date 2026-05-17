package workflow

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed data/gh_cli_permissions.json
var ghCLIPermissionsJSON []byte

// ghCLISubcommandGroup maps a gh subcommand group (e.g. "pr", "issue") to its permissions.
type ghCLISubcommandGroup struct {
	Description       string   `json:"description"`
	ReadSubcommands   []string `json:"read_subcommands"`
	WriteSubcommands  []string `json:"write_subcommands"`
	ReadPermissions   []string `json:"read_permissions"`
	WritePermissions  []string `json:"write_permissions"`
	AppReadPermissions  []string `json:"app_read_permissions"`
	AppWritePermissions []string `json:"app_write_permissions"`
}

// ghCLIAPIPathPattern maps a REST API path pattern to the required permissions.
type ghCLIAPIPathPattern struct {
	Pattern        string   `json:"pattern"`
	Description    string   `json:"description"`
	Permissions    []string `json:"permissions"`
	AppPermissions []string `json:"app_permissions"`
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
	// subcommandRE is dynamically compiled from the subcommand_groups keys so that adding
	// a new group to the JSON automatically extends the pattern without a code change.
	// Capture groups: (1) subcommand group, (2) action word.
	subcommandRE *regexp.Regexp
	// readCommands maps "group action" (e.g. "pr diff") to read permission scopes.
	readCommands map[string][]PermissionScope
	// writeCommands maps "group action" (e.g. "pr create") to write permission scopes.
	writeCommands map[string][]PermissionScope
	// groupReadPermissions maps a subcommand group name (e.g. "pr") to read permission scopes
	// used as fallback when the specific action is not recognised.
	groupReadPermissions map[string][]PermissionScope
	// appReadCommands maps "group action" to GitHub App read permission scopes.
	appReadCommands map[string][]PermissionScope
	// appWriteCommands maps "group action" to GitHub App write permission scopes.
	appWriteCommands map[string][]PermissionScope
	// groupAppReadPermissions maps a subcommand group name to GitHub App read permission scopes
	// used as fallback when the specific action is not recognised.
	groupAppReadPermissions map[string][]PermissionScope
	// apiPathPatterns holds compiled regexps paired with required permission scopes.
	apiPathPatterns []compiledAPIPathPattern
}

type compiledAPIPathPattern struct {
	re             *regexp.Regexp
	permissions    []PermissionScope
	appPermissions []PermissionScope
}

var ghCLIPermissions compiledGHCLIPermissions

func init() {
	var data ghCLIPermissionsData
	if err := json.Unmarshal(ghCLIPermissionsJSON, &data); err != nil {
		panic(fmt.Sprintf("failed to load gh CLI permissions from JSON: %v", err))
	}

	cp := compiledGHCLIPermissions{
		readCommands:            make(map[string][]PermissionScope),
		writeCommands:           make(map[string][]PermissionScope),
		groupReadPermissions:    make(map[string][]PermissionScope),
		appReadCommands:         make(map[string][]PermissionScope),
		appWriteCommands:        make(map[string][]PermissionScope),
		groupAppReadPermissions: make(map[string][]PermissionScope),
	}

	// Build the subcommand regex dynamically from the JSON group keys so that adding a new
	// group to gh_cli_permissions.json automatically extends the pattern without a code change.
	groups := make([]string, 0, len(data.SubcommandGroups))
	for group := range data.SubcommandGroups {
		groups = append(groups, regexp.QuoteMeta(group))
	}
	sort.Strings(groups) // deterministic alternation order
	subcommandPattern := `(?m)(?:^|[\s|;])gh\s+(` + strings.Join(groups, "|") + `)\s+([\w][\w-]*)\b`
	cp.subcommandRE = regexp.MustCompile(subcommandPattern)

	for group, sg := range data.SubcommandGroups {
		readPerms := make([]PermissionScope, len(sg.ReadPermissions))
		for i, p := range sg.ReadPermissions {
			readPerms[i] = PermissionScope(p)
		}
		writePerms := make([]PermissionScope, len(sg.WritePermissions))
		for i, p := range sg.WritePermissions {
			writePerms[i] = PermissionScope(p)
		}
		appReadPerms := make([]PermissionScope, len(sg.AppReadPermissions))
		for i, p := range sg.AppReadPermissions {
			appReadPerms[i] = PermissionScope(p)
		}
		appWritePerms := make([]PermissionScope, len(sg.AppWritePermissions))
		for i, p := range sg.AppWritePermissions {
			appWritePerms[i] = PermissionScope(p)
		}

		// Store group-level fallback (used when specific action is unknown).
		cp.groupReadPermissions[group] = readPerms
		cp.groupAppReadPermissions[group] = appReadPerms

		for _, action := range sg.ReadSubcommands {
			key := group + " " + action
			cp.readCommands[key] = readPerms
			cp.appReadCommands[key] = appReadPerms
		}
		for _, action := range sg.WriteSubcommands {
			key := group + " " + action
			cp.writeCommands[key] = writePerms
			cp.appWriteCommands[key] = appWritePerms
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
		appPerms := make([]PermissionScope, len(ap.AppPermissions))
		for i, p := range ap.AppPermissions {
			appPerms[i] = PermissionScope(p)
		}
		cp.apiPathPatterns = append(cp.apiPathPatterns, compiledAPIPathPattern{
			re:             re,
			permissions:    perms,
			appPermissions: appPerms,
		})
	}

	ghCLIPermissions = cp
}

// ghAPIRE matches `gh api <path>` invocations.
// Capture group: (1) API path (up to the first whitespace, pipe, or quote).
var ghAPIRE = regexp.MustCompile(`(?m)(?:^|[\s|;])gh\s+api\s+([^\s|;&"'\\]+)`)

// inferPermissionsFromShellScripts scans one or more shell script strings for
// gh CLI invocations and returns the minimum set of GitHub Actions and GitHub App
// permissions required to run those commands.
//
// The returned map includes both GitHub Actions (GITHUB_TOKEN) scopes and GitHub
// App-only scopes. Callers should use IsGitHubAppOnlyScope to distinguish them:
// App-only scopes are skipped when rendering GITHUB_TOKEN permissions but are passed
// to the GitHub App token minting step when a GitHub App is configured.
//
// Only read-level permissions are inferred here; write-level operations are
// intentionally not auto-escalated. Use detectWriteCommandsInShellScripts to
// surface write commands as validation errors.
func inferPermissionsFromShellScripts(scripts []string) map[PermissionScope]PermissionLevel {
	perms := make(map[PermissionScope]PermissionLevel)

	addScopes := func(scopes []PermissionScope) {
		for _, scope := range scopes {
			if _, exists := perms[scope]; !exists {
				perms[scope] = PermissionRead
			}
		}
	}

	for _, script := range scripts {
		// Match gh <group> <action> patterns.
		for _, m := range ghCLIPermissions.subcommandRE.FindAllStringSubmatch(script, -1) {
			group := strings.ToLower(m[1])
			action := strings.ToLower(m[2])
			key := group + " " + action

			matched := false
			// Check explicit read mapping first.
			if readPerms, ok := ghCLIPermissions.readCommands[key]; ok {
				addScopes(readPerms)
				matched = true
			}
			if appReadPerms, ok := ghCLIPermissions.appReadCommands[key]; ok {
				addScopes(appReadPerms)
				matched = true
			}
			if matched {
				continue
			}
			// Write commands only need read-level permissions in the activation job context.
			// (Full write escalation is rejected by detectWriteCommandsInShellScripts instead.)
			if readPerms, ok := ghCLIPermissions.writeCommands[key]; ok {
				addScopes(readPerms)
				matched = true
			}
			if appWritePerms, ok := ghCLIPermissions.appWriteCommands[key]; ok {
				addScopes(appWritePerms)
				matched = true
			}
			if matched {
				continue
			}
			// Fall back to group-level read permissions for unrecognised actions.
			addScopes(ghCLIPermissions.groupReadPermissions[group])
			addScopes(ghCLIPermissions.groupAppReadPermissions[group])
		}

		// Match gh api <path> patterns.
		for _, m := range ghAPIRE.FindAllStringSubmatch(script, -1) {
			path := m[1]
			for _, ap := range ghCLIPermissions.apiPathPatterns {
				if ap.re.MatchString(path) {
					addScopes(ap.permissions)
					addScopes(ap.appPermissions)
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
		for _, m := range ghCLIPermissions.subcommandRE.FindAllStringSubmatch(script, -1) {
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

// extractRunScriptsFromPreStepsYAML parses a pre-steps YAML string (as stored in
// WorkflowData.PreSteps) and returns the `run` script text from every step.
func extractRunScriptsFromPreStepsYAML(preStepsYAML string) []string {
	if preStepsYAML == "" {
		return nil
	}
	var wrapper map[string][]map[string]any
	if err := yaml.Unmarshal([]byte(preStepsYAML), &wrapper); err != nil {
		return nil
	}
	steps := wrapper["pre-steps"]
	if len(steps) == 0 {
		return nil
	}
	var scripts []string
	for _, step := range steps {
		if runVal, ok := step["run"].(string); ok && runVal != "" {
			scripts = append(scripts, runVal)
		}
	}
	return scripts
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
