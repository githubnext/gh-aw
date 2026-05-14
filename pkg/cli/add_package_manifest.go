package cli

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/semverutil"
)

var downloadPackageFileFromGitHubForHost = parser.DownloadFileFromGitHubForHost
var listPackageWorkflowFiles = parser.ListWorkflowFiles

var packageSourceDirectories = []string{"workflows", ".github/workflows"}

const repositoryPackageManifestFileName = "aw.yml"
const repositoryPackageManifestVersion = "1"

type resolvedRepositoryPackage struct {
	ManifestPath       string
	Name               string
	Description        string
	DocsPath           string
	InstallationSource []string
	Warnings           []string
}

func resolveRepositoryPackage(repoSpec *RepoSpec, host string) (*resolvedRepositoryPackage, error) {
	parts := strings.SplitN(repoSpec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository slug: %s", repoSpec.RepoSlug)
	}

	owner := parts[0]
	repo := parts[1]
	ref := repoSpec.Version
	if ref == "" {
		ref = "main"
	}
	packagePath := strings.Trim(repoSpec.PackagePath, "/")

	manifestPath, manifestContent, err := loadRepositoryPackageManifestFile(owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	manifest, warnings, err := parseRepositoryPackageManifest(manifestPath, manifestContent)
	if err != nil {
		return nil, err
	}

	installationSources := normalizePackageInstallablePaths(manifest.Files, packagePath)
	if len(installationSources) == 0 {
		installationSources, err = scanRepositoryPackageInstallablePaths(owner, repo, packagePath, ref, host)
		if err != nil {
			return nil, err
		}
	}
	if len(installationSources) == 0 {
		return nil, fmt.Errorf("repository %q does not declare any installable workflow markdown files", repoSpec.RepoSlug)
	}

	docsPath, err := resolveRepositoryPackageDocsPath(owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	return &resolvedRepositoryPackage{
		ManifestPath:       manifestPath,
		Name:               manifest.Name,
		Description:        manifest.Description,
		DocsPath:           docsPath,
		InstallationSource: installationSources,
		Warnings:           warnings,
	}, nil
}

func loadRepositoryPackageManifestFile(owner, repo, packagePath, ref, host string) (string, []byte, error) {
	manifestPath := joinRepositoryPackagePath(packagePath, repositoryPackageManifestFileName)
	content, err := downloadPackageFileFromGitHubForHost(owner, repo, manifestPath, ref, host)
	if err != nil {
		if !isRepositoryFileNotFound(err) {
			return "", nil, fmt.Errorf("failed to read manifest %q from %s/%s@%s: %w", manifestPath, owner, repo, ref, err)
		}
		if packagePath != "" {
			return "", nil, fmt.Errorf("repository %q is not a valid Agentic Workflow package: no aw.yml manifest found in %q; add %s or use an explicit workflow path", owner+"/"+repo+"/"+packagePath, packagePath, manifestPath)
		}
		return "", nil, fmt.Errorf("repository %q is not a valid Agentic Workflow package: no aw.yml manifest found at the repository root; add aw.yml or use an explicit workflow path", owner+"/"+repo)
	}

	return manifestPath, content, nil
}

type repositoryPackageManifest struct {
	ManifestVersion string
	MinVersion      string
	Name            string
	Description     string
	Files           []string
}

func parseRepositoryPackageManifest(manifestPath string, content []byte) (*repositoryPackageManifest, []string, error) {
	var raw any
	if err := yaml.Unmarshal(content, &raw); err != nil {
		return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: %s", manifestPath, parser.FormatYAMLError(err, 1, string(content)))
	}

	root, ok := raw.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: top-level document must be a mapping", manifestPath)
	}

	name, ok := stringValue(root["name"])
	if !ok || strings.TrimSpace(name) == "" {
		return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: name must be a non-empty string", manifestPath)
	}

	if err := parser.ValidateRepositoryPackageManifestWithSchemaAndLocation(root, manifestPath); err != nil {
		return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: %w", manifestPath, err)
	}

	manifest := &repositoryPackageManifest{
		Name: strings.TrimSpace(name),
	}
	var warnings []string

	if manifestVersion, ok := stringValue(root["manifest-version"]); ok {
		manifest.ManifestVersion = strings.TrimSpace(manifestVersion)
	} else {
		manifest.ManifestVersion = repositoryPackageManifestVersion
	}

	if minVersion, ok := stringValue(root["min-version"]); ok {
		manifest.MinVersion = strings.TrimSpace(minVersion)
		if !isSupportedManifestMinVersion(manifest.MinVersion) {
			return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: min-version must use vMAJOR.minor.patch, got %q", manifestPath, minVersion)
		}
		currentVersion := GetVersion()
		if !semverutil.IsValid(currentVersion) {
			return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: min-version validation requires a semantic-versioned compiler, but the current compiler version %q is not a valid semantic version (this indicates a build issue)", manifestPath, currentVersion)
		}
		if semverutil.Compare(currentVersion, manifest.MinVersion) < 0 {
			return nil, nil, fmt.Errorf("invalid Agentic Workflow manifest %q: min-version %q requires gh-aw %s or newer (current: %s)", manifestPath, manifest.MinVersion, manifest.MinVersion, currentVersion)
		}
	}

	if description, ok := stringValue(root["description"]); ok {
		manifest.Description = description
		if len(description) > 255 {
			warnings = append(warnings, fmt.Sprintf("Manifest %s description exceeds the 255-character marketplace display limit", manifestPath))
		}
	}

	if filesValue, ok := root["files"]; ok {
		files, fileWarnings := extractManifestFiles(filesValue, manifestPath)
		manifest.Files = files
		warnings = append(warnings, fileWarnings...)
	}

	return manifest, warnings, nil
}

func extractManifestFiles(value any, manifestPath string) ([]string, []string) {
	var rawFiles []string
	switch files := value.(type) {
	case []any:
		for _, item := range files {
			if file, ok := stringValue(item); ok {
				rawFiles = append(rawFiles, file)
			}
		}
	case []string:
		rawFiles = append(rawFiles, files...)
	default:
		return nil, []string{fmt.Sprintf("Ignoring files entry in %s because it is not a list of strings", manifestPath)}
	}

	var warnings []string
	normalized := make([]string, 0, len(rawFiles))
	seen := make(map[string]struct{})
	for _, file := range rawFiles {
		if !isSupportedPackageInstallablePath(file) {
			warnings = append(warnings, fmt.Sprintf("Ignoring files entry %q in %s: workflow files must be markdown (.md) files under workflows/ or .github/workflows/", file, manifestPath))
			continue
		}
		if _, exists := seen[file]; exists {
			continue
		}
		seen[file] = struct{}{}
		normalized = append(normalized, file)
	}

	return normalized, warnings
}

func scanRepositoryPackageInstallablePaths(owner, repo, packagePath, ref, host string) ([]string, error) {
	var collected []string
	seen := make(map[string]struct{})

	for _, sourceDir := range packageSourceDirectories {
		sourcePath := joinRepositoryPackagePath(packagePath, sourceDir)
		files, err := listPackageWorkflowFiles(owner, repo, ref, sourcePath)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan %q in %s/%s@%s: %w", sourcePath, owner, repo, ref, err)
		}

		for _, file := range files {
			if !isSupportedPackageInstallablePath(file) {
				continue
			}
			if _, exists := seen[file]; exists {
				continue
			}
			seen[file] = struct{}{}
			collected = append(collected, file)
		}
	}

	return collected, nil
}

func resolveRepositoryPackageDocsPath(owner, repo, packagePath, ref, host string) (string, error) {
	readmePath := joinRepositoryPackagePath(packagePath, "README.md")
	if _, err := downloadPackageFileFromGitHubForHost(owner, repo, readmePath, ref, host); err == nil {
		return readmePath, nil
	} else if isRepositoryFileNotFound(err) {
		return "", fmt.Errorf("repository %q is not a valid Agentic Workflow package: missing required README.md at %q", owner+"/"+repo, readmePath)
	} else {
		return "", fmt.Errorf("failed to read package README %q from %s/%s@%s: %w", readmePath, owner, repo, ref, err)
	}
}

func normalizePackageInstallablePaths(paths []string, packagePath string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{})
	for _, path := range paths {
		if !isSupportedPackageInstallablePath(path) {
			continue
		}
		path = joinRepositoryPackagePath(packagePath, path)
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized
}

func isSupportedPackageInstallablePath(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".md") &&
		(strings.HasPrefix(path, "workflows/") || strings.HasPrefix(path, ".github/workflows/"))
}

func parseRepositoryPackageSpec(spec string) (*RepoSpec, bool, error) {
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") || isLocalWorkflowPath(spec) {
		return nil, false, nil
	}

	parts := strings.SplitN(spec, "@", 2)
	specWithoutVersion := parts[0]
	if strings.HasSuffix(strings.ToLower(specWithoutVersion), ".md") {
		return nil, false, nil
	}

	slashParts := strings.Split(specWithoutVersion, "/")
	if len(slashParts) < 2 || slashParts[0] == "" || slashParts[1] == "" {
		return nil, false, nil
	}
	if !parser.IsValidGitHubIdentifier(slashParts[0]) || !parser.IsValidGitHubIdentifier(slashParts[1]) {
		return nil, false, nil
	}

	packagePath := strings.Trim(strings.Join(slashParts[2:], "/"), "/")
	if packagePath != "" {
		cleanedPath := path.Clean(packagePath)
		if cleanedPath == "." {
			packagePath = ""
		} else if cleanedPath == ".." || strings.HasPrefix(cleanedPath, "../") {
			return nil, true, fmt.Errorf("invalid repository package path %q", packagePath)
		} else {
			packagePath = cleanedPath
		}
	}

	repoSpec := &RepoSpec{
		RepoSlug:    slashParts[0] + "/" + slashParts[1],
		PackagePath: packagePath,
	}
	if len(parts) == 2 {
		repoSpec.Version = parts[1]
	}

	return repoSpec, true, nil
}

func joinRepositoryPackagePath(packagePath, relativePath string) string {
	if packagePath == "" {
		return filepath.ToSlash(relativePath)
	}
	return filepath.ToSlash(filepath.Join(packagePath, relativePath))
}

func stringValue(value any) (string, bool) {
	s, ok := value.(string)
	return s, ok
}

func isRepositoryFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "404") || strings.Contains(errText, "not found")
}

func isRepositoryPackageManifestNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "no aw.yml manifest found")
}

func isSupportedManifestMinVersion(version string) bool {
	return semverutil.IsActionVersionTag(version) && strings.Count(strings.TrimPrefix(version, "v"), ".") == 2
}
