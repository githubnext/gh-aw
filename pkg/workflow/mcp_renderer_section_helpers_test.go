//go:build !integration

package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteJSONStringMapSection(t *testing.T) {
	var output strings.Builder

	writeJSONStringMapSection(&output, "  ", "env", map[string]string{
		"B": "2",
		"A": "1",
	}, true)

	expected := "  \"env\": {\n" +
		"    \"A\": \"1\",\n" +
		"    \"B\": \"2\"\n" +
		"  },\n"

	if output.String() != expected {
		t.Fatalf("expected JSON map section:\n%s\ngot:\n%s", expected, output.String())
	}
}

func TestWriteJSONStringMapSectionEscapesKeysAndValues(t *testing.T) {
	var output strings.Builder

	writeJSONStringMapSection(&output, "  ", "env", map[string]string{
		"A\"key\t\r": "line1\nline2\\end\t\r",
	}, false)

	var parsed map[string]map[string]string
	if err := json.Unmarshal([]byte("{\n"+output.String()+"}"), &parsed); err != nil {
		t.Fatalf("expected valid JSON section, got error: %v\noutput:\n%s", err, output.String())
	}
	if parsed["env"]["A\"key\t\r"] != "line1\nline2\\end\t\r" {
		t.Fatalf("unexpected parsed value: %#v", parsed["env"])
	}
}

func TestWriteTOMLInlineStringMapSection(t *testing.T) {
	var output strings.Builder

	writeTOMLInlineStringMapSection(&output, "  ", "env", map[string]string{
		"B": "2",
		"A": "1",
	})

	expected := "  env = { \"A\" = \"1\", \"B\" = \"2\" }\n"
	if output.String() != expected {
		t.Fatalf("expected TOML map section %q, got %q", expected, output.String())
	}
}

func TestRenderGitHubMCPGuardPoliciesFromStep(t *testing.T) {
	var output strings.Builder

	renderGitHubMCPGuardPolicies(&output, nil, true, "  ")

	result := output.String()
	expected := []string{
		`"guard-policies": {`,
		`"min-integrity": "$GITHUB_MCP_GUARD_MIN_INTEGRITY"`,
		`"repos": "$GITHUB_MCP_GUARD_REPOS"`,
	}

	for _, want := range expected {
		if !strings.Contains(result, want) {
			t.Fatalf("expected guard policy output to contain %q, got:\n%s", want, result)
		}
	}
}

func TestBuildGitHubMCPEnvVarsOmitsEmptyToolsets(t *testing.T) {
	envVars := buildGitHubMCPEnvVars("$TOKEN", "$GITHUB_SERVER_URL", false, false, "")

	if _, exists := envVars["GITHUB_TOOLSETS"]; exists {
		t.Fatalf("expected empty toolsets to be omitted, got: %#v", envVars)
	}
}
