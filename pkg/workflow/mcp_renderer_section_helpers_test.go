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

func TestWriteJSONStringMapSectionRaw(t *testing.T) {
	var output strings.Builder

	writeJSONStringMapSectionRaw(&output, "  ", "env", map[string]string{
		"B": "2",
		"A": "1",
	}, true)

	expected := "  \"env\": {\n" +
		"    \"A\": \"1\",\n" +
		"    \"B\": \"2\"\n" +
		"  },\n"

	if output.String() != expected {
		t.Fatalf("expected raw JSON map section:\n%s\ngot:\n%s", expected, output.String())
	}
}

// TestWriteJSONStringMapSectionRawDoesNotDoubleEscapeShellPlaceholders verifies that
// pre-escaped shell placeholders (\${VAR}) are written with a single backslash, not
// double-escaped by json.Marshal. This is the core regression guard for the bug where
// custom MCP server env secrets were double-escaped in the generated lock file.
func TestWriteJSONStringMapSectionRawDoesNotDoubleEscapeShellPlaceholders(t *testing.T) {
	var output strings.Builder

	// \${MY_API_TOKEN} is the pre-escaped shell placeholder that
	// ReplaceTemplateExpressionsWithEnvVars emits for Copilot JSON heredoc configs.
	writeJSONStringMapSectionRaw(&output, "  ", "env", map[string]string{
		"MY_API_TOKEN": "\\${MY_API_TOKEN}", // one backslash in actual string
	}, false)

	result := output.String()

	// The output must contain a single backslash before ${, not double.
	if !strings.Contains(result, `"\${MY_API_TOKEN}"`) {
		t.Fatalf("expected single-backslash placeholder in output, got:\n%s", result)
	}
	if strings.Contains(result, `"\\${MY_API_TOKEN}"`) {
		t.Fatalf("double-escaped placeholder found in output (regression): got:\n%s", result)
	}
}

// TestWriteJSONStringMapSectionRawProducesValidJSONForPlainValues verifies that
// writeJSONStringMapSectionRaw produces structurally valid JSON even when values contain
// JSON-special characters such as double quotes, backslashes, and control characters.
// This is the regression guard for the bug where writing values verbatim (without
// json.Marshal) produced invalid JSON for non-placeholder values.
func TestWriteJSONStringMapSectionRawProducesValidJSONForPlainValues(t *testing.T) {
	var output strings.Builder

	writeJSONStringMapSectionRaw(&output, "  ", "env", map[string]string{
		// A plain header/env value that contains a double-quote and backslash —
		// both valid in a header value but must be JSON-escaped in the output.
		"KEY": `value with "quotes" and trailing\backslash`,
	}, false)

	var parsed map[string]map[string]string
	if err := json.Unmarshal([]byte("{\n"+output.String()+"}"), &parsed); err != nil {
		t.Fatalf("writeJSONStringMapSectionRaw output is not valid JSON: %v\noutput:\n%s", err, output.String())
	}
	want := `value with "quotes" and trailing\backslash`
	if got := parsed["env"]["KEY"]; got != want {
		t.Fatalf("round-trip value mismatch: want %q, got %q", want, got)
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
	envVars := buildGitHubMCPEnvVars("$TOKEN", "$GITHUB_SERVER_URL", false, false, "", "")

	if _, exists := envVars["GITHUB_TOOLSETS"]; exists {
		t.Fatalf("expected empty toolsets to be omitted, got: %#v", envVars)
	}
}
