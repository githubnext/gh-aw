//go:build !integration

package cli

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPNetworkCodemod_AddAllowedToExactNetwork(t *testing.T) {
	t.Parallel()
	for _, network := range []string{
		"network:\n  mode: defaults # preserve mode\n",
		"'network': {mode: defaults} # preserve mode\n",
		"",
	} {
		t.Run(strings.TrimSpace(network), func(t *testing.T) {
			t.Parallel()
			content := "---\n" + network + `mcp-servers:
  "server.one":
    command: "node"
    network: {allowed: ["example.com", api.example.com]}
    nested: {network: {allowed: [untouched.example]}}
tools: {bash: true}
---

# Body
network: {allowed: [body.example]}
`
			before, err := aw.Parse(content)
			require.NoError(t, err)
			result, applied, err := getMCPNetworkMigrationCodemod().Apply(content, before.Frontmatter)
			require.NoError(t, err)
			require.True(t, applied)
			after, err := aw.Parse(result)
			require.NoError(t, err)
			top := after.Frontmatter["network"].(map[string]any)
			assert.Equal(t, []any{"example.com", "api.example.com"}, top["allowed"])
			if network != "" {
				assert.Equal(t, "defaults", top["mode"])
				assert.Contains(t, result, "# preserve mode")
			}
			server := after.Frontmatter["mcp-servers"].(map[string]any)["server.one"].(map[string]any)
			assert.NotContains(t, server, "network")
			assert.Equal(t, "node", server["command"])
			assert.Equal(t, map[string]any{"network": map[string]any{"allowed": []any{"untouched.example"}}}, server["nested"])
			assert.Contains(t, result, `command: "node"`)
			assert.Equal(t, before.Markdown, after.Markdown)
			again, changed, err := getMCPNetworkMigrationCodemod().Apply(result, after.Frontmatter)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, result, again)
		})
	}
}

func TestExecutionCodemods_AWASTStructures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		migration Codemod
		yaml      string
		verify    func(*testing.T, map[string]any)
	}{
		{
			name:      "checkout flow steps and exact agent job scope",
			migration: getCheckoutPersistCredentialsFalseCodemod(),
			yaml: `"steps": [{uses: 'actions/checkout@v4', with: {ref: main}}]
jobs:
  agent: {pre-steps: [{uses: 'actions/checkout@v4', with: {'persist-credentials': true}}]}
  other: {steps: [{uses: 'actions/checkout@v4'}]}
`,
			verify: func(t *testing.T, fm map[string]any) {
				step := fm["steps"].([]any)[0].(map[string]any)
				assert.Equal(t, map[string]any{"ref": "main", "persist-credentials": false}, step["with"])
				jobs := fm["jobs"].(map[string]any)
				agent := jobs["agent"].(map[string]any)["pre-steps"].([]any)[0].(map[string]any)
				assert.Equal(t, true, agent["with"].(map[string]any)["persist-credentials"])
				other := jobs["other"].(map[string]any)["steps"].([]any)[0].(map[string]any)
				assert.NotContains(t, other, "with")
			},
		},
		{
			name:      "quoted run shell and inline existing env",
			migration: getStepsRunSecretsToEnvCodemod(),
			yaml: `steps:
  - "run": 'echo ${{ secrets.TOKEN }}'
    shell: "pwsh"
    env: {KEEP: value}
    with: {run: '${{ secrets.UNTOUCHED }}'}
other: {steps: [{run: '${{ secrets.UNTOUCHED }}'}]}
`,
			verify: func(t *testing.T, fm map[string]any) {
				step := fm["steps"].([]any)[0].(map[string]any)
				assert.Equal(t, "echo $env:TOKEN", step["run"])
				assert.Equal(t, map[string]any{"KEEP": "value", "TOKEN": "${{ secrets.TOKEN }}"}, step["env"])
				assert.Equal(t, map[string]any{"run": "${{ secrets.UNTOUCHED }}"}, step["with"])
				other := fm["other"].(map[string]any)["steps"].([]any)[0].(map[string]any)
				assert.Equal(t, "${{ secrets.UNTOUCHED }}", other["run"])
			},
		},
		{
			name:      "engine inline env exact scope",
			migration: getEngineEnvSecretsCodemod(),
			yaml: `"engine":
  id: copilot
  "env": {UNSAFE: '${{ secrets.TOKEN }}', SAFE: keep}
  runtime: {env: {UNSAFE: '${{ secrets.TOKEN }}'}}
`,
			verify: func(t *testing.T, fm map[string]any) {
				engine := fm["engine"].(map[string]any)
				assert.Equal(t, map[string]any{"SAFE": "keep"}, engine["env"])
				assert.Equal(t, map[string]any{"env": map[string]any{"UNSAFE": "${{ secrets.TOKEN }}"}}, engine["runtime"])
			},
		},
		{
			name:      "playwright quoted inline domains and network",
			migration: getPlaywrightDomainsToNetworkAllowedCodemod(),
			yaml: `"tools": {playwright: {'allowed_domains': [new.example], version: latest}}
network: {allowed: [existing.example], other: keep}
other: {tools: {playwright: {allowed_domains: [untouched.example]}}}
`,
			verify: func(t *testing.T, fm map[string]any) {
				playwright := fm["tools"].(map[string]any)["playwright"].(map[string]any)
				assert.Equal(t, map[string]any{"version": "latest"}, playwright)
				assert.Equal(t, map[string]any{"allowed": []any{"existing.example", "new.example"}, "other": "keep"}, fm["network"])
				assert.Equal(t, map[string]any{"tools": map[string]any{"playwright": map[string]any{"allowed_domains": []any{"untouched.example"}}}}, fm["other"])
			},
		},
		{
			name:      "permissions inline retains unrelated grant",
			migration: getDependabotPermissionsCodemod(),
			yaml: `tools: {github: {toolsets: [issues]}}
"permissions": {contents: write}
`,
			verify: func(t *testing.T, fm map[string]any) {
				assert.Equal(t, map[string]any{"contents": "write", "issues": "read"}, fm["permissions"])
			},
		},
		{
			name:      "dispatch quoted inline keys exact required scope",
			migration: getWorkflowDispatchRequiredFalseCodemod(),
			yaml: `"on":
  slash_command: test
  "workflow_dispatch":
    inputs:
      "path.with.dots": {'required': true, description: 'required: true', nested: {required: true}}
      string: {required: "true"}
other: {required: true}
`,
			verify: func(t *testing.T, fm map[string]any) {
				inputs := fm["on"].(map[string]any)["workflow_dispatch"].(map[string]any)["inputs"].(map[string]any)
				input := inputs["path.with.dots"].(map[string]any)
				assert.Equal(t, false, input["required"])
				assert.Equal(t, "required: true", input["description"])
				assert.Equal(t, map[string]any{"required": true}, input["nested"])
				assert.Equal(t, map[string]any{"required": "true"}, inputs["string"])
				assert.Equal(t, map[string]any{"required": true}, fm["other"])
			},
		},
		{
			name:      "pull request target quoted checkout preserves sibling",
			migration: getPullRequestTargetCheckoutFalseCodemod(),
			yaml: `"on": {pull_request_target: null}
"checkout": true # checkout comment
other: {checkout: true}
`,
			verify: func(t *testing.T, fm map[string]any) {
				assert.Equal(t, false, fm["checkout"])
				assert.Equal(t, map[string]any{"checkout": true}, fm["other"])
			},
		},
		{
			name:      "install scripts inline runtimes preserves versions",
			migration: getRunInstallScriptsToRuntimesNodeCodemod(),
			yaml: `"run-install-scripts": false # install policy
"runtimes": {node: {version: "22"}, python: {version: "3.13"}}
other: {run-install-scripts: true}
`,
			verify: func(t *testing.T, fm map[string]any) {
				assert.NotContains(t, fm, "run-install-scripts")
				runtimes := fm["runtimes"].(map[string]any)
				assert.Equal(t, map[string]any{"version": "22", "run-install-scripts": false}, runtimes["node"])
				assert.Equal(t, map[string]any{"version": "3.13"}, runtimes["python"])
				assert.Equal(t, map[string]any{"run-install-scripts": true}, fm["other"])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := "---\n# retained header\nname: 'Keep quotes'\n" + tt.yaml + "---\n\n# Body\n${{ secrets.UNTOUCHED }}\n"
			before, err := aw.Parse(content)
			require.NoError(t, err)
			result, applied, err := tt.migration.Apply(content, before.Frontmatter)
			require.NoError(t, err)
			require.True(t, applied)
			after, err := aw.Parse(result)
			require.NoError(t, err)
			tt.verify(t, after.Frontmatter)
			assert.Equal(t, before.Markdown, after.Markdown)
			assert.Contains(t, result, "# retained header")
			assert.Contains(t, result, "name: 'Keep quotes'")
			for _, marker := range []string{"# checkout comment", "# install policy"} {
				if strings.Contains(tt.yaml, marker) {
					assert.Contains(t, result, marker)
				}
			}
			again, changed, err := tt.migration.Apply(result, after.Frontmatter)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, result, again)
		})
	}
}
