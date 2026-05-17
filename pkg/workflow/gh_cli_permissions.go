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

// ghAPICmdRE matches `gh api` at a command boundary, capturing the rest of the line.
var ghAPICmdRE = regexp.MustCompile(`(?m)(?:^|[\s|;])gh\s+api\s+(.+)`)

// ghAPIValueFlags is the set of `gh api` flags that consume the next token as their value.
// These are skipped so that the API endpoint path can be located even when flags precede it.
var ghAPIValueFlags = map[string]bool{
	"-X": true, "--method": true,
	"-H": true, "--header": true,
	"-f": true, "--field": true,
	"-F": true, "--raw-field": true,
	"-q": true, "--jq": true,
	"-t": true, "--template": true,
	"--cache": true, "--paginate-limit": true,
}

// extractGHAPIEndpoints extracts API endpoint paths from `gh api` invocations in a shell script.
// It handles common invocation patterns such as:
//
//	gh api /repos/owner/repo/pulls
//	gh api -H 'Accept: application/vnd.github+json' /repos/owner/repo/pulls
//	gh api --method GET /repos/owner/repo/pulls
//	gh api "/repos/owner/repo/pulls"
//
// Flags and their arguments are skipped until the first non-flag token is found.
func extractGHAPIEndpoints(script string) []string {
	var endpoints []string
	for _, m := range ghAPICmdRE.FindAllStringSubmatch(script, -1) {
		// m[1] is everything after `gh api ` on the same line.
		if ep := parseGHAPIEndpoint(m[1]); ep != "" {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

// parseGHAPIEndpoint returns the API endpoint path from the argument string that follows
// `gh api`, skipping flags and their values. Surrounding quotes are stripped.
func parseGHAPIEndpoint(args string) string {
	tokens := splitShellTokens(strings.TrimSpace(args))
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if strings.HasPrefix(tok, "-") {
			// Flags with embedded value (e.g. --method=GET) consume only one token.
			if strings.Contains(tok, "=") {
				i++
				continue
			}
			i++
			// Flags that take a separate value argument: skip the next token too.
			if ghAPIValueFlags[tok] && i < len(tokens) {
				i++
			}
			continue
		}
		// First non-flag token is the endpoint; strip surrounding quotes.
		return strings.Trim(tok, `"'`)
	}
	return ""
}

// splitShellTokens splits a shell argument string by whitespace while respecting
// single and double quoted regions. Quotes are preserved in the returned tokens so
// that the caller can strip them as needed.
//
// Backslash escape sequences (e.g. \" inside a double-quoted string) are treated as
// opaque two-character sequences and passed through unchanged; the caller is
// responsible for any further unescaping. This is sufficient for the practical
// `gh api` invocation patterns handled here.
func splitShellTokens(s string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && !inSingle && i+1 < len(s):
			// Escape sequence outside single quotes: consume the backslash and the
			// next character as a single unit so that \" does not close a quoted string.
			cur.WriteByte(c)
			i++
			cur.WriteByte(s[i])
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			cur.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			cur.WriteByte(c)
		case (c == ' ' || c == '\t') && !inSingle && !inDouble:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

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
		for _, path := range extractGHAPIEndpoints(script) {
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

// extractRunScriptsFromSectionYAML parses a step-section YAML string (e.g. as stored in
// WorkflowData.PreSteps, PostSteps, PreAgentSteps, or CustomSteps) and returns the `run`
// script text from every step. sectionName must match the top-level key in the YAML
// (e.g. "pre-steps", "post-steps", "pre-agent-steps", "steps").
func extractRunScriptsFromSectionYAML(sectionYAML, sectionName string) []string {
	if sectionYAML == "" {
		return nil
	}
	var wrapper map[string][]map[string]any
	if err := yaml.Unmarshal([]byte(sectionYAML), &wrapper); err != nil {
		return nil
	}
	steps := wrapper[sectionName]
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

// extractRunScriptsFromPreStepsYAML parses a pre-steps YAML string (as stored in
// WorkflowData.PreSteps) and returns the `run` script text from every step.
func extractRunScriptsFromPreStepsYAML(preStepsYAML string) []string {
	return extractRunScriptsFromSectionYAML(preStepsYAML, "pre-steps")
}

// extractRunScriptsFromJobSection returns the `run` script text from every step in the
// named section (e.g. "pre-steps", "steps", "post-steps") of the named job configuration
// inside the frontmatter jobs map.
//
// It is a read-only extraction: it never mutates the jobs map.
func extractRunScriptsFromJobSection(jobs map[string]any, jobName, sectionName string) []string {
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

	raw, ok := configMap[sectionName]
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

// extractRunScriptsFromJobPreSteps returns the `run` script text from every
// pre-step in the named job configuration inside the frontmatter jobs map.
//
// It is a read-only extraction: it never mutates the jobs map.
func extractRunScriptsFromJobPreSteps(jobs map[string]any, jobName string) []string {
	return extractRunScriptsFromJobSection(jobs, jobName, "pre-steps")
}
