//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func findRepositoryRootForSharedWrappersTest(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("failed to locate repository root")
		}
		wd = parent
	}
}

func loadSharedGitHubMCPWrappersFrontmatter(t *testing.T) map[string]any {
	t.Helper()

	root := findRepositoryRootForSharedWrappersTest(t)
	content, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "shared", "github-mcp-pagination-wrappers.md"))
	if err != nil {
		t.Fatalf("failed to read shared wrapper file: %v", err)
	}

	frontmatterYAML, err := extractMarkdownFrontmatterYAML(content)
	if err != nil {
		t.Fatalf("failed to extract frontmatter: %v", err)
	}

	var frontmatter map[string]any
	if err := yaml.Unmarshal(frontmatterYAML, &frontmatter); err != nil {
		t.Fatalf("failed to parse frontmatter: %v", err)
	}
	return frontmatter
}

func TestSharedGitHubMCPWrappersIncludeFileExcerptTool(t *testing.T) {
	frontmatter := loadSharedGitHubMCPWrappersFrontmatter(t)
	config := NewCompiler().extractMCPScriptsConfig(frontmatter)
	if config == nil {
		t.Fatal("expected mcp-scripts config")
	}

	tool := config.Tools["get_file_contents_excerpt"]
	if tool == nil {
		t.Fatal("expected get_file_contents_excerpt tool")
	}

	for _, input := range []string{"owner", "repo", "path", "ref", "byteOffset", "maxBytes", "startLine", "endLine"} {
		if _, ok := tool.Inputs[input]; !ok {
			t.Fatalf("expected input %q", input)
		}
	}
	for _, input := range []string{"owner", "repo", "path"} {
		if !tool.Inputs[input].Required {
			t.Fatalf("expected input %q to be required", input)
		}
	}

	for _, snippet := range []string{
		`-H "Range: bytes=${BYTE_OFFSET}-${BYTE_END}"`,
		`"maxBytes must be between 1 and 200000"`,
		`"truncated_by_max_bytes"`,
		`"content_bytes"`,
	} {
		if !strings.Contains(tool.Run, snippet) {
			t.Fatalf("expected wrapper script to contain %q", snippet)
		}
	}
}

func TestSharedGitHubMCPFileExcerptToolBoundsOutput(t *testing.T) {
	frontmatter := loadSharedGitHubMCPWrappersFrontmatter(t)
	config := NewCompiler().extractMCPScriptsConfig(frontmatter)
	tool := config.Tools["get_file_contents_excerpt"]
	if tool == nil {
		t.Fatal("expected get_file_contents_excerpt tool")
	}

	tempDir := t.TempDir()
	ghLog := filepath.Join(tempDir, "gh.log")
	ghPath := filepath.Join(tempDir, "gh")
	if err := os.WriteFile(ghPath, []byte(`#!/bin/bash
printf '%s\n' "$*" > "$GH_LOG"
printf 'alpha\nbeta\ngamma\ndelta\n'
`), 0755); err != nil {
		t.Fatalf("failed to write mock gh: %v", err)
	}

	cmd := exec.Command("bash", "-c", tool.Run)
	cmd.Env = append(os.Environ(),
		"PATH="+tempDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GH_LOG="+ghLog,
		"INPUT_OWNER=github",
		"INPUT_REPO=gh-aw",
		"INPUT_PATH=README.md",
		"INPUT_REF=main",
		"INPUT_MAXBYTES=20",
		"INPUT_STARTLINE=2",
		"INPUT_ENDLINE=3",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wrapper failed: %v\n%s", err, output)
	}

	var result map[string]any
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse wrapper JSON %q: %v", output, err)
	}

	if result["content"] != "beta\ngamma\n" {
		t.Fatalf("expected bounded line excerpt, got %#v", result["content"])
	}
	if result["path"] != "README.md" {
		t.Fatalf("expected path in result, got %#v", result["path"])
	}
	if result["max_bytes"] != float64(20) {
		t.Fatalf("expected max_bytes 20, got %#v", result["max_bytes"])
	}

	logBytes, err := os.ReadFile(ghLog)
	if err != nil {
		t.Fatalf("failed to read gh log: %v", err)
	}
	log := string(logBytes)
	if !strings.Contains(log, "Range: bytes=0-20") {
		t.Fatalf("expected gh api range header, got %q", log)
	}
	if !strings.Contains(log, "repos/github/gh-aw/contents/README.md") {
		t.Fatalf("expected gh api contents endpoint, got %q", log)
	}
}
