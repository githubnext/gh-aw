package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ChangesetEntry represents a parsed changeset file
type ChangesetEntry struct {
	Package     string
	BumpType    string // "major", "minor", or "patch"
	Description string
	FilePath    string
}

// VersionInfo represents semantic version components
type VersionInfo struct {
	Major int
	Minor int
	Patch int
}

// NewChangesetCommand creates the changeset command and its subcommands
func NewChangesetCommand() *cobra.Command {
	changesetCmd := &cobra.Command{
		Use:   "changeset",
		Short: "Manage changesets for version releases",
		Long: `Manage changesets for version releases.
		
Changesets are markdown files in .changeset/ directory that describe changes
and specify the type of version bump (major, minor, or patch).`,
	}

	// Add subcommands
	changesetCmd.AddCommand(newChangesetVersionCommand())
	changesetCmd.AddCommand(newChangesetReleaseCommand())

	return changesetCmd
}

// newChangesetVersionCommand creates the version subcommand
func newChangesetVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Determine version bump from changesets and update CHANGELOG.md",
		Long: `Analyze changesets in .changeset/ directory, determine the appropriate
version bump (patch, minor, or major), and update CHANGELOG.md with changes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChangesetVersion()
		},
	}
}

// newChangesetReleaseCommand creates the release subcommand
func newChangesetReleaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release [patch|minor]",
		Short: "Create a release commit based on changesets",
		Long: `Create a release commit that includes version bump and changelog updates.
		
If no version type is specified, it will be determined from changesets.
Valid types: patch, minor (major releases not automated for safety).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var releaseType string
			if len(args) > 0 {
				releaseType = args[0]
				if releaseType != "patch" && releaseType != "minor" {
					return fmt.Errorf("invalid release type '%s'. Must be 'patch' or 'minor'", releaseType)
				}
			}
			return runChangesetRelease(releaseType)
		},
	}
	return cmd
}

// parseChangesetFile parses a single changeset markdown file
func parseChangesetFile(path string) (*ChangesetEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read changeset file: %w", err)
	}

	// Extract frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil, fmt.Errorf("invalid changeset format: missing frontmatter")
	}

	// Find end of frontmatter
	var frontmatterEnd int
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			frontmatterEnd = i
			break
		}
	}
	if frontmatterEnd == 0 {
		return nil, fmt.Errorf("invalid changeset format: unclosed frontmatter")
	}

	// Parse frontmatter YAML
	frontmatterYAML := strings.Join(lines[1:frontmatterEnd], "\n")
	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract package and bump type
	packageName, ok := frontmatter["gh-aw"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'gh-aw' field in frontmatter")
	}

	// Get description (everything after frontmatter)
	description := strings.TrimSpace(strings.Join(lines[frontmatterEnd+1:], "\n"))

	return &ChangesetEntry{
		Package:     "gh-aw",
		BumpType:    packageName,
		Description: description,
		FilePath:    path,
	}, nil
}

// readChangesets reads all changeset files from .changeset/ directory
func readChangesets() ([]*ChangesetEntry, error) {
	changesetDir := ".changeset"
	if _, err := os.Stat(changesetDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("changeset directory not found: .changeset/")
	}

	entries, err := os.ReadDir(changesetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read changeset directory: %w", err)
	}

	var changesets []*ChangesetEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(changesetDir, entry.Name())
		changeset, err := parseChangesetFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, console.FormatWarningMessage("Skipping %s: %v\n"), entry.Name(), err)
			continue
		}
		changesets = append(changesets, changeset)
	}

	return changesets, nil
}

// determineVersionBump determines the highest version bump from changesets
func determineVersionBump(changesets []*ChangesetEntry) string {
	if len(changesets) == 0 {
		return ""
	}

	// Priority: major > minor > patch
	hasMajor := false
	hasMinor := false
	hasPatch := false

	for _, cs := range changesets {
		switch cs.BumpType {
		case "major":
			hasMajor = true
		case "minor":
			hasMinor = true
		case "patch":
			hasPatch = true
		}
	}

	if hasMajor {
		return "major"
	}
	if hasMinor {
		return "minor"
	}
	if hasPatch {
		return "patch"
	}

	return ""
}

// getCurrentVersion gets the current version from git tags
func getCurrentVersion() (*VersionInfo, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		// No tags exist, start from v0.0.0
		return &VersionInfo{Major: 0, Minor: 0, Patch: 0}, nil
	}

	// Parse version string (e.g., "v1.2.3")
	versionStr := strings.TrimSpace(string(output))
	versionStr = strings.TrimPrefix(versionStr, "v")

	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s", versionStr)
	}

	var major, minor, patch int
	if _, err := fmt.Sscanf(versionStr, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	return &VersionInfo{Major: major, Minor: minor, Patch: patch}, nil
}

// bumpVersion increments version based on bump type
func bumpVersion(current *VersionInfo, bumpType string) *VersionInfo {
	next := &VersionInfo{
		Major: current.Major,
		Minor: current.Minor,
		Patch: current.Patch,
	}

	switch bumpType {
	case "major":
		next.Major++
		next.Minor = 0
		next.Patch = 0
	case "minor":
		next.Minor++
		next.Patch = 0
	case "patch":
		next.Patch++
	}

	return next
}

// formatVersion formats version as string
func formatVersion(v *VersionInfo) string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// updateChangelog updates CHANGELOG.md with new version and changes
func updateChangelog(version string, changesets []*ChangesetEntry) error {
	changelogPath := "CHANGELOG.md"

	// Read existing changelog or create header
	var existingContent string
	if content, err := os.ReadFile(changelogPath); err == nil {
		existingContent = string(content)
	} else {
		existingContent = "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n"
	}

	// Build new entry
	var newEntry strings.Builder
	newEntry.WriteString(fmt.Sprintf("## %s - %s\n\n", version, time.Now().Format("2006-01-02")))

	// Group changes by type
	var majorChanges, minorChanges, patchChanges []*ChangesetEntry
	for _, cs := range changesets {
		switch cs.BumpType {
		case "major":
			majorChanges = append(majorChanges, cs)
		case "minor":
			minorChanges = append(minorChanges, cs)
		case "patch":
			patchChanges = append(patchChanges, cs)
		}
	}

	// Write changes by category
	if len(majorChanges) > 0 {
		newEntry.WriteString("### Breaking Changes\n\n")
		for _, cs := range majorChanges {
			newEntry.WriteString(fmt.Sprintf("- %s\n", extractFirstLine(cs.Description)))
		}
		newEntry.WriteString("\n")
	}

	if len(minorChanges) > 0 {
		newEntry.WriteString("### Features\n\n")
		for _, cs := range minorChanges {
			newEntry.WriteString(fmt.Sprintf("- %s\n", extractFirstLine(cs.Description)))
		}
		newEntry.WriteString("\n")
	}

	if len(patchChanges) > 0 {
		newEntry.WriteString("### Bug Fixes\n\n")
		for _, cs := range patchChanges {
			newEntry.WriteString(fmt.Sprintf("- %s\n", extractFirstLine(cs.Description)))
		}
		newEntry.WriteString("\n")
	}

	// Insert new entry after header
	headerEnd := strings.Index(existingContent, "\n## ")
	var updatedContent string
	if headerEnd == -1 {
		// No existing entries, append to end
		updatedContent = existingContent + newEntry.String()
	} else {
		// Insert before first existing entry
		updatedContent = existingContent[:headerEnd+1] + newEntry.String() + existingContent[headerEnd+1:]
	}

	// Write updated changelog
	if err := os.WriteFile(changelogPath, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write changelog: %w", err)
	}

	return nil
}

// extractFirstLine extracts the first non-empty line from text
func extractFirstLine(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return text
}

// deleteChangesetFiles removes processed changeset files
func deleteChangesetFiles(changesets []*ChangesetEntry) error {
	for _, cs := range changesets {
		if err := os.Remove(cs.FilePath); err != nil {
			return fmt.Errorf("failed to remove changeset %s: %w", cs.FilePath, err)
		}
	}
	return nil
}

// runChangesetVersion implements the version command
func runChangesetVersion() error {
	changesets, err := readChangesets()
	if err != nil {
		return err
	}

	if len(changesets) == 0 {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("No changesets found"))
		return nil
	}

	bumpType := determineVersionBump(changesets)
	currentVersion, err := getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	nextVersion := bumpVersion(currentVersion, bumpType)
	versionString := formatVersion(nextVersion)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Current version: %s", formatVersion(currentVersion))))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Bump type: %s", bumpType)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Next version: %s", versionString)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nChanges:"))
	for _, cs := range changesets {
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", cs.BumpType, extractFirstLine(cs.Description))
	}

	// Update changelog
	if err := updateChangelog(versionString, changesets); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("\n✓ Updated CHANGELOG.md"))

	return nil
}

// runChangesetRelease implements the release command
func runChangesetRelease(releaseType string) error {
	changesets, err := readChangesets()
	if err != nil {
		return err
	}

	if len(changesets) == 0 {
		return fmt.Errorf("no changesets found to release")
	}

	// Determine bump type
	bumpType := releaseType
	if bumpType == "" {
		bumpType = determineVersionBump(changesets)
	}

	if bumpType == "major" && releaseType == "" {
		return fmt.Errorf("major releases must be explicitly specified with 'gh aw changeset release major' for safety")
	}

	currentVersion, err := getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	nextVersion := bumpVersion(currentVersion, bumpType)
	versionString := formatVersion(nextVersion)

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Creating %s release: %s", bumpType, versionString)))

	// Update changelog
	if err := updateChangelog(versionString, changesets); err != nil {
		return err
	}

	// Delete changeset files
	if err := deleteChangesetFiles(changesets); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("✓ Updated CHANGELOG.md"))
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("✓ Removed %d changeset file(s)", len(changesets))))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("\nNext steps:"))
	fmt.Fprintf(os.Stderr, "  1. Review CHANGELOG.md\n")
	fmt.Fprintf(os.Stderr, "  2. Commit changes: git add CHANGELOG.md .changeset/ && git commit -m \"Release %s\"\n", versionString)
	fmt.Fprintf(os.Stderr, "  3. Create tag: git tag -a %s -m \"Release %s\"\n", versionString, versionString)
	fmt.Fprintf(os.Stderr, "  4. Push: git push origin main %s\n", versionString)

	return nil
}
