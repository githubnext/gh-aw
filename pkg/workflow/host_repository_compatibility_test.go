//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCallDispatchDefaultRefCompatibility(t *testing.T) {
	for _, event := range []string{"pull_request", "issue_comment"} {
		for _, inlined := range []bool{true, false} {
			for _, explicitRef := range []string{"", "refs/heads/configured"} {
				t.Run(event+"/"+map[bool]string{true: "inlined", false: "host"}[inlined]+"/"+explicitRef, func(t *testing.T) {
					data := &WorkflowData{
						On:             "on:\n  workflow_call:\n  " + event + ":\n",
						InlinedImports: inlined,
						SafeOutputs: &SafeOutputsConfig{
							DispatchWorkflow: &DispatchWorkflowConfig{Workflows: []string{"worker"}, TargetRef: explicitRef},
						},
					}
					var steps []string
					NewCompiler().addHandlerManagerConfigEnvVar(&steps, data)
					config := extractHandlerConfig(t, strings.Join(steps, ""))["dispatch_workflow"]
					assert.Equal(t, activationTargetRepoExpr, config["target-repo"])
					switch {
					case explicitRef != "":
						assert.Equal(t, explicitRef, config["target-ref"])
					case inlined:
						assert.NotContains(t, config, "target-ref", "the handler must resolve the PR head, not github.workflow_ref")
					default:
						assert.Equal(t, "${{ steps.resolve-host-repo.outputs.target_ref }}", config["target-ref"])
					}
					assert.Equal(t, explicitRef, data.SafeOutputs.DispatchWorkflow.TargetRef, "runtime config generation must not mutate source configuration")
				})
			}
		}
	}
}

func TestWorkflowCallGHESIdentityPermissions(t *testing.T) {
	for _, optIn := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing callers", true: "GHES opt-in"}[optIn], func(t *testing.T) {
			compiler := NewCompiler()
			data := &WorkflowData{
				Name:           "host-identity",
				WorkflowID:     "host-identity",
				On:             "on:\n  workflow_call:\n",
				RawFrontmatter: map[string]any{"on": map[string]any{"workflow_call": nil}},
				AI:             "copilot",
				Permissions:    "permissions:\n  contents: read\n",
				SkipBots:       []string{"bot[bot]"},
				SafeOutputs:    &SafeOutputsConfig{AddComments: &AddCommentsConfig{}},
			}
			if optIn {
				data.Permissions += "  id-token: write\n"
			}
			require.NoError(t, compiler.buildJobs(data, ".github/workflows/host-identity.md"))
			resolverJobs := map[string]bool{}
			for name, job := range compiler.jobManager.GetAllJobs() {
				if !strings.Contains(strings.Join(job.Steps, ""), "id: resolve-host-repo\n") {
					continue
				}
				resolverJobs[name] = true
				perms := NewPermissionsParser(job.Permissions).ToPermissions()
				level, exists := perms.Get(PermissionIdToken)
				if optIn {
					assert.True(t, exists, "%s must be able to request its own GHES identity", name)
					assert.Equal(t, PermissionWrite, level, name)
				} else {
					assert.False(t, exists, "%s must not elevate existing reusable-workflow callers", name)
				}
			}
			for _, name := range []string{"pre_activation", "activation", "agent", "safe_outputs", "conclusion"} {
				assert.True(t, resolverJobs[name], "%s must be covered by the generated-job permission contract", name)
			}
		})
	}
}

func TestWorkflowCallGHESIdentityRespectsExplicitJobDenial(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{On: "workflow_call", Permissions: "permissions:\n  id-token: write\n"}
	job := &Job{Name: "safe_outputs", RunsOn: "runs-on: ubuntu-latest", Permissions: "permissions:\n  contents: read\n  id-token: none\n", Steps: []string{compiler.generateResolveHostRepoStep(data)}}
	require.NoError(t, compiler.jobManager.AddJob(job))
	compiler.ensureHostRepositoryOIDCJobPermissions(data)
	level, exists := NewPermissionsParser(job.Permissions).ToPermissions().Get(PermissionIdToken)
	assert.True(t, exists)
	assert.Equal(t, PermissionNone, level)
}

func TestWorkflowCallGHESIdentityRespectsSafeOutputDenial(t *testing.T) {
	compiler := NewCompiler()
	none := "none"
	data := &WorkflowData{
		On:          "workflow_call",
		Permissions: "permissions:\n  id-token: write\n",
		SafeOutputs: &SafeOutputsConfig{IDToken: &none},
	}
	job := &Job{Name: "safe_outputs", RunsOn: "runs-on: ubuntu-latest", Permissions: "permissions:\n  contents: read\n", Steps: []string{compiler.generateResolveHostRepoStep(data)}}
	require.NoError(t, compiler.jobManager.AddJob(job))
	compiler.ensureHostRepositoryOIDCJobPermissions(data)
	_, exists := NewPermissionsParser(job.Permissions).ToPermissions().Get(PermissionIdToken)
	assert.False(t, exists)
}
