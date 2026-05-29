package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var skillDownloadLog = logger.New("cli:skill_download")

// downloadSkillFileFromGitHub downloads the agentic-workflows SKILL.md file from GitHub.
func downloadSkillFileFromGitHub(verbose bool) (string, error) {
	skillDownloadLog.Print("Downloading agentic-workflows SKILL.md from GitHub")

	// Determine the ref to use (tag for releases, main for dev builds)
	ref := "main"
	if workflow.IsRelease() {
		ref = GetVersion()
		skillDownloadLog.Printf("Using release tag: %s", ref)
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using release version: "+ref))
		}
	} else {
		skillDownloadLog.Print("Using main branch for dev build")
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Using main branch (dev build)"))
		}
	}

	// Construct the raw GitHub URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/github/gh-aw/%s/.github/skills/agentic-workflows/SKILL.md", ref)
	skillDownloadLog.Printf("Downloading from URL: %s", rawURL)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: constants.DefaultHTTPClientTimeout,
	}

	// Download the file
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to download skill file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fall back to gh CLI for authenticated access (e.g., private repos in codespaces)
		if resp.StatusCode == http.StatusNotFound && isGHCLIAvailable() {
			skillDownloadLog.Print("Unauthenticated download returned 404, trying gh CLI for authenticated access")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Retrying download with gh CLI authentication..."))
			}
			if content, ghErr := downloadSkillFileViaGHCLI(ref); ghErr == nil {
				patchedContent := patchSkillFileURLs(content, ref)
				skillDownloadLog.Printf("Successfully downloaded skill file via gh CLI (%d bytes)", len(patchedContent))
				return patchedContent, nil
			} else {
				skillDownloadLog.Printf("gh CLI fallback failed: %v", ghErr)
			}
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("failed to download skill file: HTTP %d", resp.StatusCode)
	}

	// Read the content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read skill file content: %w", err)
	}

	contentStr := string(content)

	// Patch URLs to match the current version/ref
	patchedContent := patchSkillFileURLs(contentStr, ref)
	if patchedContent != contentStr && verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Patched URLs to use ref: "+ref))
	}

	skillDownloadLog.Printf("Successfully downloaded skill file (%d bytes)", len(patchedContent))
	return patchedContent, nil
}

// patchSkillFileURLs patches URLs in the skill file to use the correct ref.
func patchSkillFileURLs(content, ref string) string {
	// Pattern 1: Convert local paths to GitHub URLs
	// `.github/aw/file.md` -> `https://github.com/github/gh-aw/blob/{ref}/.github/aw/file.md`
	content = strings.ReplaceAll(content, "`.github/aw/", fmt.Sprintf("`https://github.com/github/gh-aw/blob/%s/.github/aw/", ref))

	// Pattern 2: Update existing GitHub URLs to use the correct ref
	// https://github.com/github/gh-aw/blob/main/ -> https://github.com/github/gh-aw/blob/{ref}/
	if ref != "main" {
		content = strings.ReplaceAll(content, "/blob/main/", fmt.Sprintf("/blob/%s/", ref))
	}

	return content
}

// downloadSkillFileViaGHCLI downloads the skill file using the gh CLI with authentication.
// This is used as a fallback when the unauthenticated raw.githubusercontent.com download fails
// (e.g., for private repositories accessed from codespaces).
func downloadSkillFileViaGHCLI(ref string) (string, error) {
	output, err := workflow.RunGH("Downloading skill file...", "api",
		"/repos/github/gh-aw/contents/.github/skills/agentic-workflows/SKILL.md?ref="+url.QueryEscape(ref),
		"--header", "Accept: application/vnd.github.raw")
	if err != nil {
		return "", fmt.Errorf("gh api download failed: %w", err)
	}
	return string(output), nil
}

func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}
