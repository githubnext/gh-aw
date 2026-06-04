//go:build !integration

package workflow

import "testing"

func TestCollectEngineVersionsForMetadata(t *testing.T) {
	data := &WorkflowData{
		AI: "copilot",
		EngineConfig: &EngineConfig{
			ID:         "copilot",
			Version:    "1.2.3-custom",
			CopilotSDK: true,
		},
	}

	versions := collectEngineVersionsForMetadata(data)
	if versions["copilot"] != "1.2.3-custom" {
		t.Fatalf("Expected copilot override version, got: %q", versions["copilot"])
	}
	if versions["copilot-sdk"] == "" {
		t.Fatal("Expected copilot-sdk version when copilot-sdk is enabled")
	}
	if versions["claude"] == "" {
		t.Fatal("Expected default claude version in metadata map")
	}
}

func TestResolveAgentImageRunnerIdentifier(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		frontmatter := map[string]any{"runs-on": "ubuntu-latest"}
		if got := resolveAgentImageRunnerIdentifier(frontmatter); got != "ubuntu-latest" {
			t.Fatalf("Expected string runner identifier, got: %q", got)
		}
	})

	t.Run("array value", func(t *testing.T) {
		frontmatter := map[string]any{"runs-on": []any{"self-hosted", "linux"}}
		if got := resolveAgentImageRunnerIdentifier(frontmatter); got != `["self-hosted","linux"]` {
			t.Fatalf("Expected serialized array runner identifier, got: %q", got)
		}
	})
}
