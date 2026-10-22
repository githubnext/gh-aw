package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var mcpCACertPathPattern = regexp.MustCompile(`^[A-Za-z0-9_./ -]+$`)

const (
	mcpCAWorkspacePrefix = "${{ github.workspace }}/"
	mcpCATempPrefix      = "${{ runner.temp }}/"
)

// ParseMCPCACertFile validates the optional author-facing per-server CA setting.
func ParseMCPCACertFile(config map[string]any, serverType, name string) (string, error) {
	raw, exists := config["ca-cert-file"]
	if !exists {
		return "", nil
	}
	field := "mcp-servers." + name + ".ca-cert-file"
	if serverType != "http" {
		return "", fmt.Errorf("%s is only supported for HTTP servers", field)
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%s must be a non-empty file path", field)
	}
	if err := ValidateMCPCACertFile(value); err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	url, ok := config["url"].(string)
	if !ok || !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("%s requires an HTTPS URL", field)
	}
	return value, nil
}

// ValidateMCPCACertFile checks author-controlled paths before they enter a shell heredoc.
// Files may be provisioned by setup steps, so existence and PEM validation happen at runtime.
func ValidateMCPCACertFile(value string) error {
	path := value
	for _, prefix := range []string{mcpCAWorkspacePrefix, mcpCATempPrefix} {
		if remainder, ok := strings.CutPrefix(path, prefix); ok {
			path = remainder
			break
		}
	}
	if path == "" || strings.TrimSpace(value) != value || !mcpCACertPathPattern.MatchString(path) ||
		strings.HasSuffix(path, "/") {
		return errors.New("expected a non-empty literal PEM file path, optionally prefixed with '${{ github.workspace }}/' or '${{ runner.temp }}/'; shell substitutions and other expressions are not supported")
	}
	for segment := range strings.SplitSeq(path, "/") {
		if segment == ".." {
			return errors.New("parent-directory traversal is not supported; use a repository-relative or absolute PEM file path")
		}
	}
	return nil
}
