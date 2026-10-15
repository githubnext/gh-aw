//go:build !integration

package cli

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitCommandWithSAMLForbidden(t *testing.T) {
	for _, rawStatus := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(http.StatusText(rawStatus), func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("CI", "true")
			t.Setenv("GH_TOKEN", "test-token-not-for-target-org")
			require.NoError(t, exec.Command("git", "init", "--quiet").Run())
			stubSAMLForbiddenCopilotSetup(t)

			originalTransport := http.DefaultTransport
			originalList := listAgenticWorkflowsMarkdownFiles
			t.Cleanup(func() {
				http.DefaultTransport = originalTransport
				listAgenticWorkflowsMarkdownFiles = originalList
			})
			listAgenticWorkflowsMarkdownFiles = fetchAgenticWorkflowsMarkdownFiles
			var requests []string
			http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				require.Empty(t, req.Header.Get("Authorization"), "public downloads must not send the SAML token")
				requests = append(requests, req.URL.String())
				status := http.StatusForbidden
				body := `{"message":"Resource protected by organization SAML enforcement"}`
				switch req.URL.String() {
				case ghAWMarkdownFilesAPIURL:
				case ghAWMarkdownFilesRawURL:
					status = rawStatus
					if status == http.StatusOK {
						body = `["create-agentic-workflow.md","public-raw-list.md"]`
					}
				default:
					t.Fatalf("unexpected request: %s", req.URL)
				}
				return &http.Response{
					StatusCode: status,
					Status:     http.StatusText(status),
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			})

			for range 2 {
				cmd := NewInitCommand()
				cmd.SetArgs([]string{"--engine", "copilot"})
				require.NoError(t, cmd.Execute(), "init must succeed and remain idempotent with SAML/403 responses")
			}
			require.Equal(t, []string{ghAWMarkdownFilesAPIURL, ghAWMarkdownFilesRawURL, ghAWMarkdownFilesAPIURL, ghAWMarkdownFilesRawURL}, requests)
			for _, file := range []string{
				".gitattributes", ".vscode/settings.json", bootstrapAgenticSkillPath,
				bootstrapAgenticAgentPath, bootstrapMCPConfigPath, bootstrapCopilotSetupPath,
			} {
				require.FileExists(t, filepath.FromSlash(file))
			}
			skill, err := os.ReadFile(bootstrapAgenticSkillPath)
			require.NoError(t, err)
			require.Contains(t, string(skill), ".github/aw/create-agentic-workflow.md")
			require.Contains(t, string(skill), "never send an Authorization header or token")
			if rawStatus == http.StatusOK {
				require.Contains(t, string(skill), ".github/aw/public-raw-list.md")
			} else {
				require.Contains(t, string(skill), ".github/aw/designer.md", "use embedded list when both public endpoints fail")
			}
		})
	}
}

func stubSAMLForbiddenCopilotSetup(t *testing.T) {
	t.Helper()
	originalVersion := GetVersion()
	originalBranch := resolveGhAwDefaultBranchForCopilotSetup
	originalRef := resolveGhAwRefForCopilotSetup
	t.Cleanup(func() {
		SetVersionInfo(originalVersion)
		resolveGhAwDefaultBranchForCopilotSetup = originalBranch
		resolveGhAwRefForCopilotSetup = originalRef
	})
	SetVersionInfo("dev")
	forbidden := errors.New("gh: Resource protected by organization SAML enforcement (HTTP 403)")
	resolveGhAwDefaultBranchForCopilotSetup = func(context.Context, string) (string, error) {
		return "", forbidden
	}
	resolveGhAwRefForCopilotSetup = func(context.Context, string) (string, error) {
		return "", forbidden
	}
}
