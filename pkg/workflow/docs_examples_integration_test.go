//go:build integration

package workflow

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestDocumentationGalleryWorkflowsCompile(t *testing.T) {
	galleryDir := filepath.Join("..", "..", "docs", "src", "content", "docs", "gallery")
	entries, err := os.ReadDir(galleryDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || entry.Name() == "index.md" || entry.Name() == "multi-repo.md" || entry.Name() == "maintaining-repos.md" {
			continue
		}

		t.Run(strings.TrimSuffix(entry.Name(), ".md"), func(t *testing.T) {
			pagePath := filepath.Join(galleryDir, entry.Name())
			content, readErr := os.ReadFile(pagePath)
			if readErr != nil {
				t.Fatal(readErr)
			}

			workflowName, workflowContent, extractErr := extractPrimaryExampleWorkflow(string(content))
			if extractErr != nil {
				t.Fatal(extractErr)
			}
			compileExampleWorkflow(t, workflowName, []byte(workflowContent), pagePath)
		})
	}

	t.Run("repo-assist-upstream", func(t *testing.T) {
		const (
			sourceURL = "https://github.com/githubnext/agentics/blob/main/workflows/repo-assist.md?plain=1"
			rawURL    = "https://raw.githubusercontent.com/githubnext/agentics/main/workflows/repo-assist.md"
		)

		client := &http.Client{Timeout: 30 * time.Second}
		response, fetchErr := client.Get(rawURL)
		if fetchErr != nil {
			t.Fatalf("fetch %s (raw form of %s): %v", rawURL, sourceURL, fetchErr)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("fetch %s (raw form of %s): %s", rawURL, sourceURL, response.Status)
		}

		workflowContent, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			t.Fatalf("read %s: %v", rawURL, readErr)
		}
		compileExampleWorkflow(t, "repo-assist.md", workflowContent, sourceURL)
	})
}

func compileExampleWorkflow(t *testing.T, workflowName string, workflowContent []byte, source string) {
	t.Helper()
	tempDir := testutil.TempDir(t, "docs-example")
	workflowPath := filepath.Join(tempDir, workflowName)
	if writeErr := os.WriteFile(workflowPath, workflowContent, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	compiler := NewCompiler()
	compiler.SetWorkflowIdentifier(workflowName)
	if compileErr := compiler.CompileWorkflow(workflowPath); compileErr != nil {
		t.Fatalf("compile primary workflow from %s: %v", source, compileErr)
	}
}

func extractPrimaryExampleWorkflow(content string) (string, string, error) {
	const titlePrefix = `title=".github/workflows/`

	var workflowName string
	var workflowLines []string
	inPrimaryWorkflow := false
	primaryWorkflowCount := 0

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if !inPrimaryWorkflow {
			if !strings.HasPrefix(line, "```aw ") || !strings.Contains(line, titlePrefix) {
				continue
			}

			nameStart := strings.Index(line, titlePrefix) + len(titlePrefix)
			nameEnd := strings.Index(line[nameStart:], `"`)
			if nameEnd < 0 {
				return "", "", fmt.Errorf("primary workflow title is missing a closing quote")
			}
			workflowName = line[nameStart : nameStart+nameEnd]
			if filepath.Ext(workflowName) != ".md" || filepath.Base(workflowName) != workflowName {
				return "", "", fmt.Errorf("primary workflow title must name a .github/workflows/*.md file")
			}

			primaryWorkflowCount++
			inPrimaryWorkflow = true
			workflowLines = nil
			continue
		}

		if line == "```" {
			inPrimaryWorkflow = false
			continue
		}
		workflowLines = append(workflowLines, line)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return "", "", scanErr
	}
	if inPrimaryWorkflow {
		return "", "", fmt.Errorf("primary workflow code block is not closed")
	}
	if primaryWorkflowCount != 1 {
		return "", "", fmt.Errorf("expected exactly one primary workflow code block, found %d", primaryWorkflowCount)
	}

	return workflowName, strings.Join(workflowLines, "\n") + "\n", nil
}
