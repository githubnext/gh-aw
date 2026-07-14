package cli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/workflow"
)

func readRemoteRepoBranchFile(repoOverride, branchName, filePath, hostname string) ([]byte, error) {
	return readRemoteRepoBranchFileContext(context.Background(), repoOverride, branchName, filePath, hostname)
}

func readRemoteRepoBranchFileContext(ctx context.Context, repoOverride, branchName, filePath, hostname string) ([]byte, error) {
	args := []string{"api",
		"--method", "GET",
		"repos/{owner}/{repo}/contents/" + filePath,
		"--field", "ref=" + branchName,
		"--jq", ".content",
		"--repo", repoOverride,
	}
	if hostname != "" && hostname != "github.com" {
		args = append(args, "--hostname", hostname)
	}
	cmd := workflow.ExecGHContext(ctx, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if isRemoteFileNotFoundOutput(string(out)) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("failed to fetch %s from %s: %w", filePath, branchName, err)
	}

	// GitHub API returns base64-encoded content with embedded newlines.
	b64 := strings.Join(strings.Fields(strings.TrimSpace(string(out))), "")
	decoded, decodeErr := base64.StdEncoding.DecodeString(b64)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode %s from %s: %w", filePath, branchName, decodeErr)
	}
	return decoded, nil
}

func isRemoteFileNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func isRemoteFileNotFoundOutput(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "404") || strings.Contains(s, "not found")
}
