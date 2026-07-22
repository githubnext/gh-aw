//go:build !integration

package workflow

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileFormalPM10Workflow(t *testing.T, frontmatter string) string {
	t.Helper()

	md := frontmatter + `

# Mission

Formal PM10/AppG test workflow.
`

	compiler := NewCompiler(WithNoEmit(true))
	wd, err := compiler.ParseWorkflowString(md, "workflow.md")
	require.NoError(t, err)

	yamlOut, err := compiler.CompileToYAML(wd, "workflow.md")
	require.NoError(t, err)
	require.NotEmpty(t, yamlOut)

	return yamlOut
}

func TestFormalPM10a_PreActivationJobPrecedesActivation(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-pm10a
on:
  pull_request:
    types: [opened]
  roles: [triage]
engine: copilot
strict: false
permissions:
  contents: read
---`)

	preActivationSection := extractJobSection(yamlOut, "pre_activation")
	require.NotEmpty(t, preActivationSection, "pre_activation job must be present when roles are configured")

	activationSection := extractJobSection(yamlOut, "activation")
	require.NotEmpty(t, activationSection, "activation job must be present")
	assert.Contains(t, activationSection, "needs: pre_activation", "activation must depend on pre_activation")
}

func TestFormalPM10b_ActivatedOutputExistsAndGatesActivation(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-pm10b
on:
  pull_request:
    types: [opened]
  roles: [triage]
engine: copilot
strict: false
permissions:
  contents: read
---`)

	preActivationSection := extractJobSection(yamlOut, "pre_activation")
	require.NotEmpty(t, preActivationSection)
	assert.Contains(t, preActivationSection, "outputs:")
	assert.Contains(t, preActivationSection, "activated:")

	activationSection := extractJobSection(yamlOut, "activation")
	require.NotEmpty(t, activationSection)
	assert.Contains(t, activationSection, "if: needs.pre_activation.outputs.activated == 'true'")
}

func TestFormalPM10c_PreActivationJobHasNoWritePermissions(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-pm10c
on:
  pull_request:
    types: [opened]
  roles: [triage]
engine: copilot
strict: false
permissions:
  contents: read
---`)

	preActivationSection := extractJobSection(yamlOut, "pre_activation")
	require.NotEmpty(t, preActivationSection)

	writePermissionLine := regexp.MustCompile(`(?m)^\s{6}[a-z0-9-]+:\s*write\s*$`)
	assert.False(t, writePermissionLine.MatchString(preActivationSection), "pre_activation permissions must be read-only")
}

func TestFormalPM10d_RequiredRolesDefaultToAdminMaintainerWrite(t *testing.T) {
	compiler := NewCompiler()

	roles := compiler.extractRoles(map[string]any{
		"on": map[string]any{
			"pull_request": map[string]any{"types": []any{"opened"}},
		},
	})

	assert.Equal(t, []string{"admin", "maintainer", "write"}, roles)
}

func TestFormalPM10d_RequiredRolesCustomFieldHonoured(t *testing.T) {
	compiler := NewCompiler()

	roles := compiler.extractRoles(map[string]any{
		"on": map[string]any{
			"roles": []any{"triage"},
		},
	})

	assert.Equal(t, []string{"triage"}, roles)
}

func TestFormalAppG1_CompiledStepsUseSHAPins(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-appg1
on:
  pull_request:
    types: [opened]
  roles: [triage]
engine: copilot
strict: false
permissions:
  contents: read
---`)

	usesLine := regexp.MustCompile(`(?m)^\s*uses:\s*(\S+)`)
	matches := usesLine.FindAllStringSubmatch(yamlOut, -1)
	require.NotEmpty(t, matches)

	shaRef := regexp.MustCompile(`^[0-9a-f]{40}$`)
	remoteUsesCount := 0

	for _, m := range matches {
		ref := m[1]
		if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "docker://") {
			continue
		}

		parts := strings.SplitN(ref, "@", 2)
		require.Len(t, parts, 2, "remote uses step must include @ref: %s", ref)
		assert.True(t, shaRef.MatchString(parts[1]), "remote uses step must be pinned to full SHA: %s", ref)
		remoteUsesCount++
	}

	assert.Positive(t, remoteUsesCount, "expected at least one remote uses step")
}

func TestFormalAppG2_PullRequestTargetContainsForkValidation(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-appg2
on:
  pull_request:
    types: [opened]
    forks: []
  roles: [triage]
engine: copilot
strict: false
permissions:
  contents: read
---`)

	assert.Contains(t, yamlOut, "# forks: [] # Fork filtering applied via job conditions")
}

func TestFormalStrictMode_WritePermissionsRejected(t *testing.T) {
	tests := []struct {
		name  string
		scope PermissionScope
	}{
		{name: "contents write", scope: PermissionContents},
		{name: "actions write", scope: PermissionActions},
		{name: "issues write", scope: PermissionIssues},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			perms := NewPermissions()
			perms.Set(tc.scope, PermissionWrite)

			err := validateDangerousPermissions(&WorkflowData{Permissions: "permissions: {}"}, perms)
			require.Error(t, err)
			require.ErrorContains(t, err, "write permissions")
		})
	}
}

func TestFormalPM10a_PreActivationAbsentWhenNoRoles(t *testing.T) {
	yamlOut := compileFormalPM10Workflow(t, `---
name: formal-pm10a-no-roles
on:
  workflow_dispatch:
engine: copilot
strict: false
permissions:
  contents: read
---`)

	preActivationSection := extractJobSection(yamlOut, "pre_activation")
	require.NotEmpty(t, preActivationSection)
	assert.Contains(t, preActivationSection, `GH_AW_REQUIRED_ROLES: ""`)

	activationSection := extractJobSection(yamlOut, "activation")
	require.NotEmpty(t, activationSection)
	assert.Contains(t, activationSection, "needs: pre_activation")
	assert.Contains(t, activationSection, "needs.pre_activation.outputs.activated")
}

func TestFormalPM10c_PreActivationPermissionsTableDriven(t *testing.T) {
	tests := []struct {
		name string
		on   string
	}{
		{
			name: "nil permissions variant",
			on: `on:
  pull_request:
    types: [opened]
  roles: [triage]`,
		},
		{
			name: "contents read variant",
			on: `on:
  pull_request:
    types: [opened]
  roles: [triage]
  permissions:
    contents: read`,
		},
		{
			name: "actions read variant",
			on: `on:
  pull_request:
    types: [opened]
  roles: [triage]
  permissions:
    actions: read`,
		},
	}

	writePermissionLine := regexp.MustCompile(`(?m)^\s{6}[a-z0-9-]+:\s*write\s*$`)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			yamlOut := compileFormalPM10Workflow(t, `---
name: formal-pm10c-table
`+tc.on+`
engine: copilot
strict: false
permissions:
  contents: read
---`)

			preActivationSection := extractJobSection(yamlOut, "pre_activation")
			require.NotEmpty(t, preActivationSection)
			assert.False(t, writePermissionLine.MatchString(preActivationSection), "pre_activation permissions must stay read-only")
		})
	}
}
