package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/parser"
)

var downloadPackageFileFromGitHubForHost = parser.DownloadFileFromGitHubForHost
var listPackageWorkflowFiles = parser.ListWorkflowFiles

var packageManifestAliases = []string{"aw.yml", "agents.yml", "agents.yaml"}
var packageSourceDirectories = []string{"workflows", ".github/workflows"}

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

	manifestPath, manifestContent, foundAliases, err := loadRepositoryPackageManifestFile(owner, repo, ref, host)
	if err != nil {
		return nil, err
	}

	manifest, warnings, err := parseRepositoryPackageManifest(manifestPath, manifestContent)
	if err != nil {
		return nil, err
	}

	warnings = append(warnings, repositoryPackageAliasWarnings(foundAliases, manifestPath)...)

	installationSources := normalizePackageInstallablePaths(manifest.Files)
	if len(installationSources) == 0 {
		installationSources, err = scanRepositoryPackageInstallablePaths(owner, repo, ref, host)
		if err != nil {
			return nil, err
		}
	}
	if len(installationSources) == 0 {
		return nil, fmt.Errorf("repository %q does not declare any installable workflow markdown files", repoSpec.RepoSlug)
	}

	docsPath, docsWarnings := resolveRepositoryPackageDocsPath(owner, repo, ref, host, manifest, installationSources)
	warnings = append(warnings, docsWarnings...)

	return &resolvedRepositoryPackage{
		ManifestPath:       manifestPath,
		Name:               manifest.Name,
		Description:        manifest.Description,
		DocsPath:           docsPath,
		InstallationSource: installationSources,
		Warnings:           warnings,
	}, nil
}

func loadRepositoryPackageManifestFile(owner, repo, ref, host string) (string, []byte, []string, error) {
	var selectedPath string
	var selectedContent []byte
	var foundAliases []string

	for _, manifestPath := range packageManifestAliases {
		content, err := downloadPackageFileFromGitHubForHost(owner, repo, manifestPath, ref, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return "", nil, nil, fmt.Errorf("failed to read manifest %q from %s/%s@%s: %w", manifestPath, owner, repo, ref, err)
		}

		foundAliases = append(foundAliases, manifestPath)
		if selectedPath == "" {
			selectedPath = manifestPath
			selectedContent = content
		}
	}

	if selectedPath == "" {
		return "", nil, nil, fmt.Errorf("repository %q is not a valid Agentic Workflow package: no aw.yml manifest found at the repository root; add aw.yml or use an explicit workflow path", owner+"/"+repo)
	}

	return selectedPath, selectedContent, foundAliases, nil
}

type repositoryPackageManifest struct {
	Name        string
	Description string
	Docs        string
	Files       []string
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

	manifest := &repositoryPackageManifest{Name: strings.TrimSpace(name)}
	var warnings []string

	if description, ok := stringValue(root["description"]); ok {
		manifest.Description = description
		if len(description) > 255 {
			warnings = append(warnings, fmt.Sprintf("Manifest %s description exceeds the 255-character marketplace display limit", manifestPath))
		}
	}

	if docs, ok := stringValue(root["docs"]); ok {
		if strings.HasSuffix(strings.ToLower(docs), ".md") {
			manifest.Docs = docs
		} else {
			warnings = append(warnings, fmt.Sprintf("Ignoring docs entry %q in %s because it does not point to a markdown file", docs, manifestPath))
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

func scanRepositoryPackageInstallablePaths(owner, repo, ref, host string) ([]string, error) {
	var collected []string
	seen := make(map[string]struct{})

	for _, sourceDir := range packageSourceDirectories {
		files, err := listPackageWorkflowFiles(owner, repo, ref, sourceDir)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan %q in %s/%s@%s: %w", sourceDir, owner, repo, ref, err)
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

func resolveRepositoryPackageDocsPath(owner, repo, ref, host string, manifest *repositoryPackageManifest, installationSources []string) (string, []string) {
	if manifest.Docs != "" {
		return manifest.Docs, nil
	}

	var warnings []string
	candidates := []string{
		filepath.ToSlash(filepath.Join("docs", parameterizePackageName(manifest.Name)+".md")),
	}
	for _, source := range installationSources {
		candidates = append(candidates, filepath.ToSlash(filepath.Join("docs", filepath.Base(source))))
	}

	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}

		if _, err := downloadPackageFileFromGitHubForHost(owner, repo, candidate, ref, host); err == nil {
			return candidate, warnings
		} else if !isRepositoryFileNotFound(err) {
			warnings = append(warnings, fmt.Sprintf("Unable to validate docs path %q: %v", candidate, err))
		}
	}

	if len(installationSources) > 0 {
		return installationSources[0], warnings
	}

	return "", warnings
}

func normalizePackageInstallablePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{})
	for _, path := range paths {
		if !isSupportedPackageInstallablePath(path) {
			continue
		}
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

func repositoryPackageAliasWarnings(foundAliases []string, selectedPath string) []string {
	var warnings []string
	if selectedPath != "aw.yml" {
		warnings = append(warnings, fmt.Sprintf("Using legacy manifest %q; rename it to aw.yml", selectedPath))
	}
	if len(foundAliases) > 1 {
		var extras []string
		for _, alias := range foundAliases {
			if alias != selectedPath {
				extras = append(extras, alias)
			}
		}
		if len(extras) > 0 {
			warnings = append(warnings, fmt.Sprintf("Multiple repository manifests found (%s); using %s and ignoring the legacy aliases", strings.Join(foundAliases, ", "), selectedPath))
		}
	}
	return warnings
}

// parameterizePackageName converts a human-readable package name into a lowercase
// hyphenated slug suitable for docs path probing (for example, "Repo Assist" → "repo-assist").
func parameterizePackageName(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return ""
	}

	var b strings.Builder
	lastHyphen := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			b.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.Trim(b.String(), "-")
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
