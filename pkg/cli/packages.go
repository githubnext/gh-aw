package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/setutil"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
)

var packagesLog = logger.New("cli:packages")

// Pre-compiled regexes for package processing (performance optimization)
var (
	includePattern = regexp.MustCompile(`^@include(\?)?\s+(.+)$`)
)

// collectLocalIncludeDependencies collects dependencies for package-based workflows
func collectLocalIncludeDependencies(content, packagePath string, verbose bool) ([]IncludeDependency, error) {
	packagesLog.Printf("Collecting include dependencies: packagePath=%s, content_size=%d", packagePath, len(content))
	var dependencies []IncludeDependency
	seen := make(map[string]struct{})

	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Collecting package dependencies from: "+packagePath))
	}

	err := collectLocalIncludeDependenciesRecursive(content, packagePath, &dependencies, seen, verbose)
	packagesLog.Printf("Collected %d include dependencies from %s", len(dependencies), packagePath)
	return dependencies, err
}

// collectLocalIncludeDependenciesRecursive recursively processes @include directives in package content
func collectLocalIncludeDependenciesRecursive(content, baseDir string, dependencies *[]IncludeDependency, seen map[string]struct{}, verbose bool) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		collectLocalIncludeDependenciesRecursiveLine(scanner.Text(), baseDir, dependencies, seen, verbose)
	}

	return scanner.Err()
}

func collectLocalIncludeDependenciesRecursiveLine(line string, baseDir string, dependencies *[]IncludeDependency, seen map[string]struct{}, verbose bool) {
	matches := includePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	isOptional := matches[1] == "?"
	includePath := strings.TrimSpace(matches[2])
	filePath := collectLocalIncludeDependenciesRecursiveFilePath(includePath)
	fullSourcePath := filepath.Join(baseDir, filePath)
	if setutil.Contains(seen, fullSourcePath) {
		return
	}
	seen[fullSourcePath] = struct{}{}
	*dependencies = append(*dependencies, IncludeDependency{SourcePath: fullSourcePath, TargetPath: filePath, IsOptional: isOptional})
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Found include dependency: %s -> %s", fullSourcePath, filePath)))
	}
	collectLocalIncludeDependenciesRecursiveDescend(fullSourcePath, dependencies, seen, verbose)
}

func collectLocalIncludeDependenciesRecursiveFilePath(includePath string) string {
	if strings.Contains(includePath, "#") {
		parts := strings.SplitN(includePath, "#", 2)
		return parts[0]
	}
	return includePath
}

func collectLocalIncludeDependenciesRecursiveDescend(fullSourcePath string, dependencies *[]IncludeDependency, seen map[string]struct{}, verbose bool) {
	includedContent, err := os.ReadFile(fullSourcePath)
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not read include file %s: %v", fullSourcePath, err)))
		}
		return
	}
	markdownContent, err := parser.ExtractMarkdownContent(string(includedContent))
	if err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not extract markdown from %s: %v", fullSourcePath, err)))
		}
		return
	}
	includedDir := filepath.Dir(fullSourcePath)
	if err := collectLocalIncludeDependenciesRecursive(markdownContent, includedDir, dependencies, seen, verbose); err != nil && verbose {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Error processing includes in %s: %v", fullSourcePath, err)))
	}
}

// copyIncludeDependenciesFromPackageWithForce copies include dependencies from package filesystem with force option
func copyIncludeDependenciesFromPackageWithForce(dependencies []IncludeDependency, githubWorkflowsDir string, verbose bool, force bool, tracker *FileTracker) error {
	packagesLog.Printf("Copying %d include dependencies to %s (force=%t)", len(dependencies), githubWorkflowsDir, force)
	for _, dep := range dependencies {
		if err := copyIncludeDependenciesFromPackageWithForceOne(dep, githubWorkflowsDir, verbose, force, tracker); err != nil {
			return err
		}
	}

	return nil
}

func copyIncludeDependenciesFromPackageWithForceOne(dep IncludeDependency, githubWorkflowsDir string, verbose bool, force bool, tracker *FileTracker) error {
	targetPath := filepath.Join(githubWorkflowsDir, dep.TargetPath)
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}
	sourceContent, ok, err := copyIncludeDependenciesFromPackageWithForceReadSource(dep, verbose)
	if err != nil || !ok {
		return err
	}
	fileExists, shouldWrite := copyIncludeDependenciesFromPackageWithForceShouldWrite(dep, targetPath, sourceContent, verbose, force)
	if !shouldWrite {
		return nil
	}
	copyIncludeDependenciesFromPackageWithForceTrack(tracker, targetPath, fileExists)
	if err := os.WriteFile(targetPath, sourceContent, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write include file %s: %w", targetPath, err)
	}
	if verbose {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Copied include file: %s -> %s", dep.SourcePath, targetPath)))
	}
	return nil
}

func copyIncludeDependenciesFromPackageWithForceReadSource(dep IncludeDependency, verbose bool) ([]byte, bool, error) {
	sourceContent, err := os.ReadFile(dep.SourcePath)
	if err == nil {
		return sourceContent, true, nil
	}
	if dep.IsOptional {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Optional include file not found: %s (you can create this file to configure the workflow)", dep.TargetPath)))
		}
		return nil, false, nil
	}
	fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to read include file %s: %v", dep.SourcePath, err)))
	return nil, false, nil
}

func copyIncludeDependenciesFromPackageWithForceShouldWrite(dep IncludeDependency, targetPath string, sourceContent []byte, verbose bool, force bool) (bool, bool) {
	existingContent, err := os.ReadFile(targetPath)
	if err != nil {
		return false, true
	}
	if bytes.Equal(existingContent, sourceContent) {
		if verbose {
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Include file %s already exists with same content, skipping", dep.TargetPath)))
		}
		return true, false
	}
	if !force {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Include file %s already exists with different content, skipping (use --force to overwrite)", dep.TargetPath)))
		return true, false
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Overwriting existing include file: "+dep.TargetPath))
	return true, true
}

func copyIncludeDependenciesFromPackageWithForceTrack(tracker *FileTracker, targetPath string, fileExists bool) {
	if tracker == nil {
		return
	}
	if fileExists {
		tracker.TrackModified(targetPath)
	} else {
		tracker.TrackCreated(targetPath)
	}
}

// IncludeDependency represents a file dependency from @include directives
type IncludeDependency struct {
	SourcePath string // Path in the source (local)
	TargetPath string // Relative path where it should be copied in .github/workflows
	IsOptional bool   // Whether this is an optional include (@include?)
}

// ExtractWorkflowDescription extracts the description field from workflow content string
func ExtractWorkflowDescription(content string) string {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return ""
	}

	if desc, ok := result.Frontmatter["description"]; ok {
		if descStr, ok := desc.(string); ok {
			return descStr
		}
	}

	return ""
}

// ExtractWorkflowEngine extracts the engine field from workflow content string.
// Supports both string format (engine: copilot) and nested format (engine: { id: copilot }).
func ExtractWorkflowEngine(content string) string {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return ""
	}

	if engine, ok := result.Frontmatter["engine"]; ok {
		// Handle string format: engine: copilot
		if engineStr, ok := engine.(string); ok {
			packagesLog.Printf("Extracted engine (string format): %s", engineStr)
			return engineStr
		}
		// Handle nested format: engine: { id: copilot }
		if engineMap, ok := engine.(map[string]any); ok {
			if id, ok := engineMap["id"]; ok {
				if idStr, ok := id.(string); ok {
					packagesLog.Printf("Extracted engine (nested format): %s", idStr)
					return idStr
				}
			}
		}
	}

	return ""
}

// ExtractWorkflowPrivateSetting extracts the private field from workflow content string.
// Returns the boolean value and whether the field was explicitly present.
func ExtractWorkflowPrivateSetting(content string) (bool, bool) {
	result, err := parser.ExtractFrontmatterFromContent(content)
	if err != nil {
		return false, false
	}

	if private, ok := result.Frontmatter["private"]; ok {
		if privateBool, ok := private.(bool); ok {
			return privateBool, true
		}
	}

	return false, false
}

// ExtractWorkflowPrivate extracts the private field from workflow content string.
// Returns true if the workflow has private: true in its frontmatter.
func ExtractWorkflowPrivate(content string) bool {
	privateBool, ok := ExtractWorkflowPrivateSetting(content)
	if ok {
		return privateBool
	}
	return false
}

// ExtractWorkflowDescriptionFromFile extracts the description field from a workflow file
func ExtractWorkflowDescriptionFromFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	return ExtractWorkflowDescription(string(content))
}
