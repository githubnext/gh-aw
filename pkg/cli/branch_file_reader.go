package cli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var branchFileReaderLog = logger.New("cli:branch_file_reader")

func readRemoteRepoBranchFile(repoOverride, branchName, filePath, hostname string) ([]byte, error) {
	return readRemoteRepoBranchFileContext(context.Background(), repoOverride, branchName, filePath, hostname)
}

func readRemoteRepoBranchFileContext(ctx context.Context, repoOverride, branchName, filePath, hostname string) ([]byte, error) {
	branchFileReaderLog.Printf("Fetching remote file: repo=%s, branch=%s, path=%s, hostname=%s", repoOverride, branchName, filePath, hostname)
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
			branchFileReaderLog.Printf("Remote file not found: path=%s, branch=%s", filePath, branchName)
			return nil, os.ErrNotExist
		}
		branchFileReaderLog.Printf("Failed to fetch %s from %s: %v", filePath, branchName, err)
		return nil, fmt.Errorf("failed to fetch %s from %s: %w", filePath, branchName, err)
	}

	// GitHub API returns base64-encoded content with embedded newlines.
	b64 := strings.Join(strings.Fields(strings.TrimSpace(string(out))), "")
	decoded, decodeErr := base64.StdEncoding.DecodeString(b64)
	if decodeErr != nil {
		branchFileReaderLog.Printf("Failed to decode %s from %s: %v", filePath, branchName, decodeErr)
		return nil, fmt.Errorf("failed to decode %s from %s: %w", filePath, branchName, decodeErr)
	}
	branchFileReaderLog.Printf("Fetched remote file %s from %s: %d bytes", filePath, branchName, len(decoded))
	return decoded, nil
}

func isRemoteFileNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func isRemoteFileNotFoundOutput(output string) bool {
	s := strings.ToLower(output)
	return strings.Contains(s, "404") || strings.Contains(s, "not found")
}
