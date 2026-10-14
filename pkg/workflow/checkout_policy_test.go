//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCheckoutAllowedRepoPolicyProvenance(t *testing.T) {
	for _, value := range []string{
		"${{ vars.ALLOWED_REPOS }}", "${{vars.allowed_repos}}", " ${{\n vars.ALLOWED_REPOS\n}} \n",
		"${{ vars['ALLOWED_REPOS'] }}", "${{ vars[ 'ALLOWED_REPOS' ] }}", "${{ VARS.ALLOWED_REPOS }}",
	} {
		t.Run("trusted/"+value, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "worker.md")
			content := "---\non: workflow_dispatch\nengine: copilot\n---\nProcess selected repositories.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			data, err := NewCompiler().ParseWorkflowString(content, file)
			require.NoError(t, err)
			for _, policy := range []any{value, []any{"org/*", value}} {
				cfg, err := checkoutConfigFromMap(map[string]any{
					"repo": "${{ inputs.repos }}", "path": "targets", "multi": true, "allowed-repos": policy,
				})
				require.NoError(t, err)
				assert.Contains(t, cfg.AllowedRepos, value)
				data.CheckoutConfigs = []*CheckoutConfig{cfg}
				_, err = NewCompiler().CompileToYAML(data, file)
				require.NoError(t, err)
			}
		})
	}
	for _, value := range []string{
		"${{ inputs.allowed }}", "${{ needs.policy.outputs.allowed }}", "${{ matrix.allowed }}",
		"${{ github.event.inputs.allowed }}", "${{ env.ALLOWED_REPOS }}", "${{ secrets.ALLOWED_REPOS }}",
		"${{ github.repository }}", "${{ vars[inputs.key] }}", "${{ vars[vars.KEY] }}",
		"${{ vars.ALLOWED_REPOS || inputs.allowed }}", "${{ vars.ALLOWED_REPOS || vars.OTHER_REPOS }}",
		"${{ format('{0}', vars.ALLOWED_REPOS) }}", "${{ fromJSON(vars.ALLOWED_REPOS) }}",
		"org/${{ vars.ALLOWED_REPOS }}", "${{ vars.ALLOWED_REPOS }}\norg/*",
		"${{ vars.ALLOWED_REPOS }}${{ inputs.allowed }}", "${{ vars.ALLOWED_REPOS }}\n*",
		"${{ vars }}", "${{ vars.ALLOWED_REPOS.other }}", "${{ vars.* }}", "${{ vars.ALLOWED_REPOS",
	} {
		t.Run("untrusted/"+value, func(t *testing.T) {
			for _, policy := range []any{value, []any{"org/*", value}} {
				_, err := checkoutConfigFromMap(map[string]any{
					"repository": "${{ inputs.repos }}", "path": "targets", "multi": true, "allowed-repos": policy,
				})
				require.ErrorContains(t, err, "direct repository or organization variable reference")
			}
			_, err := NewCompiler().CompileToYAML(&WorkflowData{CheckoutConfigs: []*CheckoutConfig{{
				Repository: "${{ inputs.repos }}", Path: "targets", PathExplicit: true,
				Multi: true, AllowedRepos: []string{"org/*", value},
			}}}, filepath.Join(t.TempDir(), "worker.md"))
			require.ErrorContains(t, err, "direct repository or organization variable reference")
		})
	}
}

func TestCompileRuntimeCheckoutPolicyProvenance(t *testing.T) {
	for _, policy := range []string{
		"${{ vars.ALLOWED_REPOS }}", "${{ vars['ALLOWED_REPOS'] }}", "${{ inputs.allowed }}",
	} {
		t.Run(policy, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "worker.md")
			checkout := "checkout:\n  multi: true\n  repo: ${{ inputs.repos }}\n  path: targets\n  allowed-repos: " + strconv.Quote(policy) + "\n"
			content := "---\non: workflow_dispatch\nengine: copilot\n" + checkout + "---\nProcess selected repositories.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			err := NewCompiler().CompileWorkflow(file)
			_, parseErr := NewCompiler().ParseWorkflowString(content, file)
			trusted := policy != "${{ inputs.allowed }}"
			if trusted {
				require.NoError(t, err)
				require.NoError(t, parseErr)
			} else {
				require.ErrorContains(t, err, "direct repository or organization variable reference")
				require.ErrorContains(t, parseErr, "direct repository or organization variable reference")
				require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
			}

			require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\n"+checkout+"---\nShared checkout.\n"), 0o644))
			content = "---\non: workflow_dispatch\nengine: copilot\nimports:\n  - ./shared.md\n---\nProcess selected repositories.\n"
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			err = NewCompiler().CompileWorkflow(file)
			_, parseErr = NewCompiler().ParseWorkflowString(content, file)
			if trusted {
				require.NoError(t, err)
				require.NoError(t, parseErr)
			} else {
				require.ErrorContains(t, err, "direct repository or organization variable reference")
				require.ErrorContains(t, parseErr, "direct repository or organization variable reference")
				require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
			}
		})
	}
}
