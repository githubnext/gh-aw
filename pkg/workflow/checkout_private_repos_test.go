//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCheckoutAllowPrivateReposParsing(t *testing.T) {
	for _, value := range []any{true, false, nil, "true", 1, "${{ vars.ALLOW_PRIVATE_REPOS }}"} {
		cfg, err := checkoutConfigFromMap(map[string]any{
			"repo": "${{ inputs.repos }}", "path": "targets", "multi": true, "allowed-repos": "org/*", "allow-private-repos": value,
		})
		if allowed, ok := value.(bool); ok {
			require.NoError(t, err)
			require.NotNil(t, cfg.AllowPrivateRepos)
			assert.Equal(t, allowed, *cfg.AllowPrivateRepos)
		} else {
			require.ErrorContains(t, err, "checkout.allow-private-repos must be a literal boolean")
		}
	}
	for _, allowed := range []bool{false, true} {
		_, err := checkoutConfigFromMap(map[string]any{"multi": true, "repository": "org/a", "allow-private-repos": allowed})
		require.ErrorContains(t, err, "checkout.allow-private-repos requires checkout.allowed-repos")
		_, err = NewCompiler().CompileToYAML(&WorkflowData{
			CheckoutConfigs: []*CheckoutConfig{{Multi: true, Repository: "org/a", AllowPrivateRepos: &allowed}},
		}, filepath.Join(t.TempDir(), "worker.md"))
		require.ErrorContains(t, err, "checkout.allow-private-repos requires checkout.allowed-repos")
	}
}

func TestRuntimeCheckoutAllowPrivateReposMerging(t *testing.T) {
	allow, deny := true, false
	for _, tt := range []struct {
		name     string
		values   []*bool
		expected bool
	}{
		{"explicit opt-in", []*bool{&allow, &allow}, true},
		{"omitted", []*bool{nil, nil}, false},
		{"later opt-in", []*bool{nil, &allow}, false},
		{"earlier opt-in", []*bool{&allow, nil}, false},
		{"explicit opt-out", []*bool{&allow, &deny}, false},
		{"cannot reverse opt-out", []*bool{&deny, &allow}, false},
		{"third cannot restore", []*bool{&allow, &deny, &allow}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var configs []*CheckoutConfig
			for _, value := range tt.values {
				configs = append(configs, &CheckoutConfig{
					Repository: "${{ inputs.repos }}", Path: "targets", Multi: true, AllowedRepos: []string{"org/*"}, AllowPrivateRepos: value,
				})
			}
			manager := NewCheckoutManager(configs)
			require.Len(t, manager.ordered, 1)
			assert.Equal(t, tt.expected, manager.ordered[0].allowPrivateRepos)
		})
	}
}

func TestCompileRuntimeCheckoutImportedPrivatePolicy(t *testing.T) {
	for _, tt := range []struct {
		name     string
		main     string
		imported string
		expected string
		wantErr  bool
	}{
		{"both opt in", "true", "true", "true", false},
		{"import cannot opt in", "", "true", "false", false},
		{"import omission denies", "true", "", "false", false},
		{"main opts out", "false", "true", "false", false},
		{"import opts out", "true", "false", "false", false},
		{"import rejects expressions", "true", "${{ vars.ALLOW_PRIVATE_REPOS }}", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			checkout := "checkout:\n  multi: true\n  repo: ${{ vars.REPOS }}\n  path: targets\n  allowed-repos: org/*\n"
			mainCheckout, importedCheckout := checkout, checkout
			if tt.main != "" {
				mainCheckout += "  allow-private-repos: " + tt.main + "\n"
			}
			if tt.imported != "" {
				importedCheckout += "  allow-private-repos: " + tt.imported + "\n"
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.md"), []byte("---\n"+importedCheckout+"---\nShared checkout.\n"), 0o644))
			content := "---\non: workflow_dispatch\nengine: copilot\nimports:\n  - ./shared.md\n" + mainCheckout + "---\nProcess selected repositories.\n"
			file := filepath.Join(dir, "worker.md")
			require.NoError(t, os.WriteFile(file, []byte(content), 0o644))
			compiler := NewCompiler()
			err := compiler.CompileWorkflow(file)
			if tt.wantErr {
				require.ErrorContains(t, err, "allow-private-repos")
				require.NoFileExists(t, filepath.Join(dir, "worker.lock.yml"))
				return
			}
			require.NoError(t, err)
			lock, err := os.ReadFile(filepath.Join(dir, "worker.lock.yml"))
			require.NoError(t, err)
			expected := `GH_AW_DYNAMIC_ALLOW_PRIVATE_REPOS: "` + tt.expected + `"`
			assert.Equal(t, 3, strings.Count(string(lock), expected))
			data, err := compiler.ParseWorkflowString(content, file)
			require.NoError(t, err)
			compiled, err := compiler.CompileToYAML(data, file)
			require.NoError(t, err)
			assert.Equal(t, 3, strings.Count(compiled, expected))
		})
	}
}
