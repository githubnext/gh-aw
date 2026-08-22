package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseGradersFromFrontmatter_Absent verifies nil return when graders absent.
func TestParseGradersFromFrontmatter_Absent(t *testing.T) {
	var c Compiler
	cfg, err := c.parseGradersFromFrontmatter(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when graders absent")
	}
}

// TestParseGradersFromFrontmatter_Nil verifies nil return when graders is nil.
func TestParseGradersFromFrontmatter_Nil(t *testing.T) {
	var c Compiler
	cfg, err := c.parseGradersFromFrontmatter(map[string]any{"graders": nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when graders is nil")
	}
}

// TestParseGradersFromFrontmatter_ZeroConfig verifies {} enables all built-ins.
func TestParseGradersFromFrontmatter_ZeroConfig(t *testing.T) {
	var c Compiler
	cfg, err := c.parseGradersFromFrontmatter(map[string]any{"graders": map[string]any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if !cfg.HasGraders() {
		t.Fatal("expected HasGraders to be true")
	}
	ids := cfg.EnabledGraderIDs()
	if len(ids) != len(BuiltinGraderIDs) {
		t.Fatalf("expected %d enabled graders, got %d", len(BuiltinGraderIDs), len(ids))
	}
}

// TestParseGradersFromFrontmatter_DisableOne verifies selective disable.
func TestParseGradersFromFrontmatter_DisableOne(t *testing.T) {
	var c Compiler
	cfg, err := c.parseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"loops": map[string]any{"enabled": false},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	ids := cfg.EnabledGraderIDs()
	for _, id := range ids {
		if id == "loops" {
			t.Fatal("loops should be disabled")
		}
	}
	// Should have all built-ins minus loops
	if len(ids) != len(BuiltinGraderIDs)-1 {
		t.Fatalf("expected %d enabled graders, got %d", len(BuiltinGraderIDs)-1, len(ids))
	}
}

// TestParseGradersFromFrontmatter_CustomGrader verifies custom grader with script.
func TestParseGradersFromFrontmatter_CustomGrader(t *testing.T) {
	var c Compiler
	cfg, err := c.parseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"my-metric": map[string]any{
				"script": "trace.toolCalls.length",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	g, ok := cfg.Graders["my-metric"]
	if !ok {
		t.Fatal("expected my-metric grader")
	}
	if g.Script != "trace.toolCalls.length" {
		t.Fatalf("unexpected script: %s", g.Script)
	}
}

// TestParseGradersFromFrontmatter_InvalidType verifies error for wrong type.
func TestParseGradersFromFrontmatter_InvalidType(t *testing.T) {
	var c Compiler
	_, err := c.parseGradersFromFrontmatter(map[string]any{"graders": "invalid"})
	if err == nil {
		t.Fatal("expected error for string graders value")
	}
}

// TestParseGradersFromFrontmatter_ForbiddenScript verifies forbidden patterns in scripts.
func TestParseGradersFromFrontmatter_ForbiddenScript(t *testing.T) {
	var c Compiler
	forbidden := []string{
		"require('fs')",
		"import('os')",
		"fetch('http://evil.com')",
		"eval('bad')",
		"process.exit(1)",
	}
	for _, script := range forbidden {
		_, err := c.parseGradersFromFrontmatter(map[string]any{
			"graders": map[string]any{
				"bad-grader": map[string]any{"script": script},
			},
		})
		if err == nil {
			t.Fatalf("expected error for forbidden script: %s", script)
		}
		if !strings.Contains(err.Error(), "forbidden pattern") {
			t.Fatalf("expected forbidden pattern error, got: %v", err)
		}
	}
}

// TestParseGradersFromFrontmatter_BuiltinScriptRejected verifies built-in cannot have script.
func TestParseGradersFromFrontmatter_BuiltinScriptRejected(t *testing.T) {
	var c Compiler
	_, err := c.parseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"retries": map[string]any{"script": "1 + 1"},
		},
	})
	if err == nil {
		t.Fatal("expected error for built-in with script")
	}
}

// TestParseGradersFromFrontmatter_CustomWithoutScript verifies custom requires script.
func TestParseGradersFromFrontmatter_CustomWithoutScript(t *testing.T) {
	var c Compiler
	_, err := c.parseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"no-script": map[string]any{},
		},
	})
	if err == nil {
		t.Fatal("expected error for custom grader without script")
	}
}

// TestParseGradersFromFrontmatter_InvalidID verifies ID validation.
func TestParseGradersFromFrontmatter_InvalidID(t *testing.T) {
	var c Compiler
	_, err := c.parseGradersFromFrontmatter(map[string]any{
		"graders": map[string]any{
			"UPPER_CASE": map[string]any{"script": "1"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

// TestParseGradersFromFrontmatter_AllDisabledError verifies error when all disabled.
func TestParseGradersFromFrontmatter_AllDisabledError(t *testing.T) {
	var c Compiler
	graders := map[string]any{}
	for _, id := range BuiltinGraderIDs {
		graders[id] = map[string]any{"enabled": false}
	}
	_, err := c.parseGradersFromFrontmatter(map[string]any{"graders": graders})
	if err == nil {
		t.Fatal("expected error when all graders disabled")
	}
}

// TestGradersConfig_EnabledGraderIDs_Order verifies stable ordering.
func TestGradersConfig_EnabledGraderIDs_Order(t *testing.T) {
	cfg := &GradersConfig{
		Graders: map[string]*GraderDefinition{
			"zebra-metric":      {ID: "zebra-metric", Script: "1"},
			"alpha-metric":      {ID: "alpha-metric", Script: "1"},
			"retries":           {ID: "retries"},
			"tool-success-rate": {ID: "tool-success-rate"},
		},
	}
	ids := cfg.EnabledGraderIDs()
	// Built-ins first in canonical order, then custom alphabetically
	if ids[0] != "tool-success-rate" {
		t.Fatalf("expected tool-success-rate first, got %s", ids[0])
	}
	if ids[1] != "retries" {
		t.Fatalf("expected retries second, got %s", ids[1])
	}
	if ids[2] != "alpha-metric" {
		t.Fatalf("expected alpha-metric third, got %s", ids[2])
	}
	if ids[3] != "zebra-metric" {
		t.Fatalf("expected zebra-metric fourth, got %s", ids[3])
	}
}

// TestBuildGraderManifest verifies manifest serialization.
func TestBuildGraderManifest(t *testing.T) {
	enabled := true
	disabled := false
	cfg := &GradersConfig{
		Graders: map[string]*GraderDefinition{
			"tool-success-rate": {ID: "tool-success-rate", Enabled: &enabled},
			"retries":           {ID: "retries", Enabled: &disabled},
			"my-custom":         {ID: "my-custom", Script: "trace.toolCalls.length"},
		},
	}
	entries := buildGraderManifest(cfg)
	if entries == nil {
		t.Fatal("expected non-nil manifest")
	}
	if entries.Version != 1 {
		t.Fatalf("expected version 1, got %d", entries.Version)
	}
	if len(entries.Graders) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries.Graders))
	}

	// Verify JSON serialization round-trips
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("json marshal error: %v", err)
	}
	var decoded graderManifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}
	if decoded.Version != 1 {
		t.Fatalf("expected decoded version 1, got %d", decoded.Version)
	}
	if len(decoded.Graders) != 3 {
		t.Fatalf("expected 3 decoded entries, got %d", len(decoded.Graders))
	}
}

// TestGenerateGradersStep_Absent verifies no step when graders nil.
func TestGenerateGradersStep_Absent(t *testing.T) {
	c := &Compiler{}
	var yaml strings.Builder
	data := &WorkflowData{Graders: nil}
	c.generateGradersStep(&yaml, data)
	if yaml.Len() != 0 {
		t.Fatal("expected no output when graders nil")
	}
}

// TestGenerateGradersStep_Present verifies step is emitted.
func TestGenerateGradersStep_Present(t *testing.T) {
	c := &Compiler{}
	initActionPinCacheForTest(c)
	var yaml strings.Builder
	data := &WorkflowData{
		Graders: &GradersConfig{
			Graders: map[string]*GraderDefinition{
				"retries": {ID: "retries"},
			},
		},
	}
	c.generateGradersStep(&yaml, data)
	output := yaml.String()
	if !strings.Contains(output, "Run trace graders") {
		t.Fatal("expected step name 'Run trace graders'")
	}
	if !strings.Contains(output, "if: always()") {
		t.Fatal("expected always() condition")
	}
	if !strings.Contains(output, "trace_graders.cjs") {
		t.Fatal("expected trace_graders.cjs require")
	}
	if !strings.Contains(output, "actions/github-script") {
		t.Fatal("expected actions/github-script usage")
	}
	if !strings.Contains(output, "await main('") {
		t.Fatal("expected main invocation with encoded payloads")
	}
	if strings.Contains(output, "{\"version\"") {
		t.Fatal("expected manifest to be encoded, not embedded as raw JSON")
	}
}

// TestGenerateGradersStep_BeforeArtifactUpload verifies ordering.
func TestGenerateGradersStep_BeforeArtifactUpload(t *testing.T) {
	c := &Compiler{}
	initActionPinCacheForTest(c)
	var yaml strings.Builder

	data := &WorkflowData{
		Graders: &GradersConfig{
			Graders: map[string]*GraderDefinition{
				"retries": {ID: "retries"},
			},
		},
	}

	// Simulate the ordering: graders step then artifact upload
	c.generateGradersStep(&yaml, data)
	yaml.WriteString("      - name: Upload agent artifacts\n")

	output := yaml.String()
	graderIdx := strings.Index(output, "Run trace graders")
	uploadIdx := strings.Index(output, "Upload agent artifacts")
	if graderIdx < 0 || uploadIdx < 0 {
		t.Fatal("expected both steps to be present")
	}
	if graderIdx >= uploadIdx {
		t.Fatal("graders step must come before artifact upload")
	}
}

// TestCollectGraderArtifactPaths verifies paths include manifest and results.
func TestCollectGraderArtifactPaths(t *testing.T) {
	paths := collectGraderArtifactPaths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if !strings.Contains(paths[0], "grader_manifest.json") {
		t.Fatal("expected grader_manifest.json in paths")
	}
	if !strings.Contains(paths[1], "grader_results.json") {
		t.Fatal("expected grader_results.json in paths")
	}
}

// initActionPinCacheForTest sets up minimal action pin resolution for tests.
func initActionPinCacheForTest(c *Compiler) {
	// The Compiler uses getActionPin/getCachedActionPin which resolves from a global
	// cache. In tests, we just verify the step generation logic, the pin is tested separately.
}

// TestCollectGraderArtifactPaths_AgentGradersDir verifies paths use the agent/graders subdirectory.
func TestCollectGraderArtifactPaths_AgentGradersDir(t *testing.T) {
	paths := collectGraderArtifactPaths()
	for _, p := range paths {
		if !strings.Contains(p, "agent/graders/") {
			t.Errorf("expected path to contain agent/graders/, got %q", p)
		}
	}
}

// TestGenerateGraderRedactionStep_CustomOnly verifies that the redaction step is only emitted
// when custom (non-builtin) grader scripts are present.
func TestGenerateGraderRedactionStep_CustomOnly(t *testing.T) {
	c := &Compiler{stepOrderTracker: NewStepOrderTracker()}
	var yaml strings.Builder

	// Built-in only — no redaction step
	data := &WorkflowData{
		Graders: &GradersConfig{
			Graders: map[string]*GraderDefinition{
				"retries": {ID: "retries"},
			},
		},
	}
	c.generateGraderRedactionStep(&yaml, "", data)
	if yaml.Len() > 0 {
		t.Error("expected no redaction step for built-in-only graders")
	}

	// Custom script — should emit redaction step
	data.Graders.Graders["my-custom"] = &GraderDefinition{
		ID:     "my-custom",
		Script: "return {value: 1}",
	}
	c.generateGraderRedactionStep(&yaml, "", data)
	if !strings.Contains(yaml.String(), "Redact grader outputs") {
		t.Error("expected redaction step for custom grader script")
	}
}

// TestGradersConfig_FrontmatterConfigField verifies the FrontmatterConfig struct has a Graders field.
func TestGradersConfig_FrontmatterConfigField(t *testing.T) {
	fc := FrontmatterConfig{
		Graders: map[string]any{},
	}
	if fc.Graders == nil {
		t.Error("expected Graders field to be set")
	}
}
