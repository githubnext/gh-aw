//go:build !integration

package workflow

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/github/gh-aw/pkg/constants"
)

// TestBuildAWFConfigJSON_BoundedQueries verifies that bounded-queries frontmatter
// is translated to the correct boundedQueries AWF config JSON section.
func TestBuildAWFConfigJSON_BoundedQueries(t *testing.T) {
	makeBaseConfig := func(bq *BoundedQueriesConfig) AWFCommandConfig {
		return AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com,api.github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:             "awf",
						BoundedQueries: bq,
					},
				},
			},
		}
	}

	t.Run("omits boundedQueries when not configured", func(t *testing.T) {
		config := makeBaseConfig(nil)
		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"boundedQueries"`, "boundedQueries section must be absent when not configured")
	})

	t.Run("emits boundedQueries with enabled:true and private repos", func(t *testing.T) {
		bq := &BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/public-docs", Sensitivity: "public"},
				{Repo: "my-org/internal-service", Sensitivity: "internal"},
				{Repo: "my-org/confidential-service", Sensitivity: "confidential"},
				{Repo: "my-org/sealed-service", Sensitivity: "sealed"},
			},
		}
		// Use a version that supports bounded queries.
		config := makeBaseConfig(bq)
		config.WorkflowData.SandboxConfig.Agent.Version = string(constants.AWFBoundedQueriesMinVersion)

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		var parsed map[string]any
		require.NoError(t, json.Unmarshal([]byte(jsonStr), &parsed))

		bqSection, ok := parsed["boundedQueries"].(map[string]any)
		require.True(t, ok, "boundedQueries section must be present")
		assert.Equal(t, true, bqSection["enabled"])

		repos, ok := bqSection["privateRepos"].([]any)
		require.True(t, ok, "privateRepos must be an array")
		require.Len(t, repos, 4)

		first, ok := repos[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "my-org/public-docs", first["repo"])
		assert.Equal(t, "public", first["sensitivity"])
	})

	t.Run("emits optional fields when set", func(t *testing.T) {
		bq := &BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/internal-service", Sensitivity: "internal"},
			},
			Runtime:        "docker",
			Timeout:        30,
			MemoryLimit:    "512m",
			Interpreter:    "python3",
			MaxInvocations: 32,
		}
		config := makeBaseConfig(bq)
		config.WorkflowData.SandboxConfig.Agent.Version = string(constants.AWFBoundedQueriesMinVersion)

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.Contains(t, jsonStr, `"runtime":"docker"`)
		assert.Contains(t, jsonStr, `"timeout":30`)
		assert.Contains(t, jsonStr, `"memoryLimit":"512m"`)
		assert.Contains(t, jsonStr, `"interpreter":"python3"`)
		assert.Contains(t, jsonStr, `"maxInvocations":32`)
	})

	t.Run("omits optional fields when unset (AWF stays source of truth for defaults)", func(t *testing.T) {
		bq := &BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/internal-service", Sensitivity: "internal"},
			},
			// No optional fields.
		}
		config := makeBaseConfig(bq)
		config.WorkflowData.SandboxConfig.Agent.Version = string(constants.AWFBoundedQueriesMinVersion)

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)

		assert.NotContains(t, jsonStr, `"runtime"`, "runtime must be omitted when unset")
		assert.NotContains(t, jsonStr, `"timeout"`, "timeout must be omitted when unset")
		assert.NotContains(t, jsonStr, `"memoryLimit"`, "memoryLimit must be omitted when unset")
		assert.NotContains(t, jsonStr, `"interpreter"`, "interpreter must be omitted when unset")
		assert.NotContains(t, jsonStr, `"maxInvocations"`, "maxInvocations must be omitted when unset")
	})

	t.Run("skips boundedQueries section for unsupported AWF versions", func(t *testing.T) {
		bq := &BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/internal-service", Sensitivity: "internal"},
			},
		}
		config := makeBaseConfig(bq)
		// Pin to a version that predates bounded queries support.
		config.WorkflowData.SandboxConfig.Agent.Version = "v0.27.42"

		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"boundedQueries"`, "boundedQueries must be skipped for unsupported AWF versions")
	})

	t.Run("nil sandbox config does not emit boundedQueries", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			AllowedDomains: "github.com",
			WorkflowData: &WorkflowData{
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
		}
		jsonStr, err := BuildAWFConfigJSON(config)
		require.NoError(t, err)
		assert.NotContains(t, jsonStr, `"boundedQueries"`)
	})
}

// TestExtractBoundedQueriesConfig validates the extraction helper in isolation.
func TestExtractBoundedQueriesConfig(t *testing.T) {
	t.Run("returns nil for nil WorkflowData", func(t *testing.T) {
		assert.Nil(t, extractBoundedQueriesConfig(nil))
	})

	t.Run("returns nil for missing sandbox config", func(t *testing.T) {
		assert.Nil(t, extractBoundedQueriesConfig(&WorkflowData{}))
	})

	t.Run("returns nil for missing agent config", func(t *testing.T) {
		assert.Nil(t, extractBoundedQueriesConfig(&WorkflowData{
			SandboxConfig: &SandboxConfig{},
		}))
	})

	t.Run("returns nil when bounded-queries is absent", func(t *testing.T) {
		assert.Nil(t, extractBoundedQueriesConfig(&WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{ID: "awf"},
			},
		}))
	})

	t.Run("maps all fields correctly", func(t *testing.T) {
		data := &WorkflowData{
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
					BoundedQueries: &BoundedQueriesConfig{
						PrivateRepos: []*BoundedQueryPrivateRepo{
							{Repo: "my-org/internal-service", Sensitivity: "internal"},
							{Repo: "my-org/confidential-service", Sensitivity: "confidential"},
						},
						Runtime:        "docker",
						Timeout:        30,
						MemoryLimit:    "512m",
						Interpreter:    "python3",
						MaxInvocations: 32,
					},
				},
			},
		}

		got := extractBoundedQueriesConfig(data)
		require.NotNil(t, got)
		assert.True(t, got.Enabled)
		assert.Equal(t, "docker", got.Runtime)
		assert.Equal(t, 30, got.Timeout)
		assert.Equal(t, "512m", got.MemoryLimit)
		assert.Equal(t, "python3", got.Interpreter)
		assert.Equal(t, 32, got.MaxInvocations)
		require.Len(t, got.PrivateRepos, 2)
		assert.Equal(t, "my-org/internal-service", got.PrivateRepos[0].Repo)
		assert.Equal(t, "internal", got.PrivateRepos[0].Sensitivity)
		assert.Equal(t, "my-org/confidential-service", got.PrivateRepos[1].Repo)
		assert.Equal(t, "confidential", got.PrivateRepos[1].Sensitivity)
	})
}

// TestValidateBoundedQueriesConfig validates all validation rules for bounded queries.
func TestValidateBoundedQueriesConfig(t *testing.T) {
	validAWFAgent := func(bq *BoundedQueriesConfig) *AgentSandboxConfig {
		return &AgentSandboxConfig{ID: "awf", BoundedQueries: bq}
	}

	t.Run("valid minimal config passes", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
			},
		})
		assert.NoError(t, validateBoundedQueriesConfig(agent))
	})

	t.Run("valid config with all optional fields passes", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "confidential"},
			},
			Runtime:        "docker",
			Timeout:        30,
			MemoryLimit:    "512m",
			Interpreter:    "python3",
			MaxInvocations: 32,
		})
		assert.NoError(t, validateBoundedQueriesConfig(agent))
	})

	t.Run("all four sensitivity values are accepted", func(t *testing.T) {
		for _, sensitivity := range []string{"public", "internal", "confidential", "sealed"} {
			t.Run(sensitivity, func(t *testing.T) {
				agent := validAWFAgent(&BoundedQueriesConfig{
					PrivateRepos: []*BoundedQueryPrivateRepo{
						{Repo: "my-org/my-repo", Sensitivity: sensitivity},
					},
				})
				assert.NoError(t, validateBoundedQueriesConfig(agent))
			})
		}
	})

	t.Run("nil agent or nil bounded-queries returns nil", func(t *testing.T) {
		assert.NoError(t, validateBoundedQueriesConfig(nil))
		assert.NoError(t, validateBoundedQueriesConfig(&AgentSandboxConfig{ID: "awf"}))
	})

	t.Run("rejects non-AWF sandbox", func(t *testing.T) {
		agent := &AgentSandboxConfig{
			// No ID / type set — will be empty string.
			BoundedQueries: &BoundedQueriesConfig{
				PrivateRepos: []*BoundedQueryPrivateRepo{
					{Repo: "my-org/my-repo", Sensitivity: "internal"},
				},
			},
		}
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bounded-queries requires the AWF sandbox")
	})

	t.Run("rejects empty private-repos", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one private-repos entry")
	})

	t.Run("rejects nil private-repos", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one private-repos entry")
	})

	t.Run("rejects invalid sensitivity value", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "top-secret"},
			},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sensitivity must be one of")
	})

	t.Run("rejects duplicate repo slugs", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
				{Repo: "my-org/my-repo", Sensitivity: "confidential"},
			},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate repository slug")
	})

	t.Run("rejects GitHub Actions expressions in repo slug", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "${{ inputs.repo }}", Sensitivity: "internal"},
			},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not contain GitHub Actions expressions")
	})

	t.Run("rejects malformed repo slug (missing slash)", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "myrepo", Sensitivity: "internal"},
			},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "'owner/repo' format")
	})

	t.Run("rejects empty repo slug", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "", Sensitivity: "internal"},
			},
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("rejects unsupported runtime", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
			},
			Runtime: "podman",
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported bounded-queries runtime")
	})

	t.Run("rejects negative timeout", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
			},
			Timeout: -1,
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be a positive integer")
	})

	t.Run("rejects invalid memory-limit format", func(t *testing.T) {
		for _, invalid := range []string{"512", "512mb", "5.5g", "abc"} {
			t.Run(invalid, func(t *testing.T) {
				agent := validAWFAgent(&BoundedQueriesConfig{
					PrivateRepos: []*BoundedQueryPrivateRepo{
						{Repo: "my-org/my-repo", Sensitivity: "internal"},
					},
					MemoryLimit: invalid,
				})
				err := validateBoundedQueriesConfig(agent)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "memory-limit")
			})
		}
	})

	t.Run("accepts valid memory-limit formats", func(t *testing.T) {
		for _, valid := range []string{"512m", "2g", "1024k", "512M", "2G"} {
			t.Run(valid, func(t *testing.T) {
				agent := validAWFAgent(&BoundedQueriesConfig{
					PrivateRepos: []*BoundedQueryPrivateRepo{
						{Repo: "my-org/my-repo", Sensitivity: "internal"},
					},
					MemoryLimit: valid,
				})
				assert.NoError(t, validateBoundedQueriesConfig(agent))
			})
		}
	})

	t.Run("rejects unsupported interpreter", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
			},
			Interpreter: "ruby",
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported bounded-queries interpreter")
	})

	t.Run("rejects negative max-invocations", func(t *testing.T) {
		agent := validAWFAgent(&BoundedQueriesConfig{
			PrivateRepos: []*BoundedQueryPrivateRepo{
				{Repo: "my-org/my-repo", Sensitivity: "internal"},
			},
			MaxInvocations: -1,
		})
		err := validateBoundedQueriesConfig(agent)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max-invocations must be a positive integer")
	})
}

// TestValidateRepoSlug covers edge cases for the repo-slug validator.
func TestValidateRepoSlug(t *testing.T) {
	valid := []string{
		"my-org/my-repo",
		"github/gh-aw",
		"my_org/my_repo",
		"my-org/my.repo",
		"a/b",
	}
	for _, slug := range valid {
		t.Run("valid: "+slug, func(t *testing.T) {
			assert.NoError(t, validateRepoSlug("field", slug))
		})
	}

	invalid := []string{
		"",
		"myrepo",
		"/myrepo",
		"my-org/",
		"${{ inputs.owner }}/my-repo",
		"my-org/${{ inputs.repo }}",
	}
	for _, slug := range invalid {
		t.Run("invalid: "+slug, func(t *testing.T) {
			assert.Error(t, validateRepoSlug("field", slug))
		})
	}
}

// TestAWFBoundedQueriesJSONRoundtrip verifies the JSON serialization of AWFBoundedQueriesConfig.
func TestAWFBoundedQueriesJSONRoundtrip(t *testing.T) {
	cfg := &AWFBoundedQueriesConfig{
		Enabled: true,
		PrivateRepos: []*AWFBoundedQueryPrivateRepo{
			{Repo: "my-org/public-docs", Sensitivity: "public"},
			{Repo: "my-org/internal-service", Sensitivity: "internal"},
			{Repo: "my-org/confidential-service", Sensitivity: "confidential"},
			{Repo: "my-org/sealed-service", Sensitivity: "sealed"},
		},
		Runtime:        "docker",
		Timeout:        30,
		MemoryLimit:    "512m",
		Interpreter:    "python3",
		MaxInvocations: 32,
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"enabled":true`)
	assert.Contains(t, jsonStr, `"privateRepos"`)
	assert.Contains(t, jsonStr, `"my-org/public-docs"`)
	assert.Contains(t, jsonStr, `"sealed"`)
	assert.Contains(t, jsonStr, `"runtime":"docker"`)
	assert.Contains(t, jsonStr, `"timeout":30`)
	assert.Contains(t, jsonStr, `"memoryLimit":"512m"`)
	assert.Contains(t, jsonStr, `"interpreter":"python3"`)
	assert.Contains(t, jsonStr, `"maxInvocations":32`)

	// Round-trip through JSON.
	var got AWFBoundedQueriesConfig
	require.NoError(t, json.Unmarshal(data, &got))
	assert.True(t, got.Enabled)
	require.Len(t, got.PrivateRepos, 4)
	assert.Equal(t, "my-org/public-docs", got.PrivateRepos[0].Repo)
	assert.Equal(t, "public", got.PrivateRepos[0].Sensitivity)
	assert.Equal(t, "my-org/sealed-service", got.PrivateRepos[3].Repo)
	assert.Equal(t, "sealed", got.PrivateRepos[3].Sensitivity)
}
