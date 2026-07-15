package cli

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/semverutil"
	"github.com/github/gh-aw/pkg/workflow"
)

var (
	errRepositoryPackageFileNotFound     = errors.New("repository package file not found")
	errRepositoryPackageManifestNotFound = errors.New("repository package manifest not found")
)

var downloadPackageFileFromGitHubForHost = downloadRepositoryPackageFileFromGitHubForHost
var listPackageWorkflowFilesForHost = listRepositoryPackageWorkflowFilesForHost
var listPackageDirFilesForHost = listRepositoryPackageDirFilesForHost
var listPackageDirFilesRecursivelyForHost = listRepositoryPackageDirFilesRecursivelyForHost
var listPackageDirSubdirsForHost = listRepositoryPackageDirSubdirsForHost
var getRepositoryPackageDefaultBranch = resolveRepositoryPackageDefaultBranch
var getRepositoryPackageLatestRelease = resolveRepositoryPackageLatestRelease
var addPackageManifestLog = logger.New("cli:add_package_manifest")

var packageSourceDirectories = []string{"workflows", constants.WorkflowsDir}

const repositoryPackageManifestFileName = "aw.yml"
const repositoryPackageManifestVersion = "1"
const ghAwRepositorySlug = "github/gh-aw"
const packageSkillsDirectory = "skills"
const packageAgentsDirectory = "agents"
const packageSkillMarkerFile = "SKILL.md"

type resolvedRepositoryPackage struct {
	ManifestPath       string
	ResolvedRef        string
	Name               string
	Emoji              string
	Description        string
	License            string
	DocsPath           string
	InstallationSource []string
	Bootstrap          *repositoryPackageBootstrap
	SkillFiles         []resolvedPackageSkillFile
	AgentFiles         []string
	Warnings           []string
}

// resolvedPackageSkillFile represents a single file within a skill directory that
// should be installed to the agentic engine skill folder.
type resolvedPackageSkillFile struct {
	// SourcePath is the file's path in the remote repository (e.g. "skills/my-skill/SKILL.md").
	SourcePath string
	// SkillName is the name of the skill directory (e.g. "my-skill").
	SkillName string
}

type packageRemoteNotFoundError struct {
	cause error
}

func (e packageRemoteNotFoundError) Error() string {
	return e.cause.Error()
}

func (e packageRemoteNotFoundError) Unwrap() []error {
	return []error{errRepositoryPackageFileNotFound, e.cause}
}

func resolveRepositoryPackage(ctx context.Context, repoSpec *RepoSpec, host string) (*resolvedRepositoryPackage, error) {
	parts := strings.SplitN(repoSpec.RepoSlug, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository slug: %s", repoSpec.RepoSlug)
	}

	owner := parts[0]
	repo := parts[1]
	// At manifest-fetch time there is no resolved package metadata yet.
	ref := repositoryPackageEffectiveRef(repoSpec, nil)
	if ref == "" {
		if isGhAwRepository(repoSpec.RepoSlug) {
			if latestRelease, err := getRepositoryPackageLatestRelease(repoSpec.RepoSlug, host); err == nil {
				ref = latestRelease
			} else {
				addPackageManifestLog.Printf("failed to resolve latest release for %s (host=%q): %v", repoSpec.RepoSlug, host, err)
			}
		}
		if ref == "" {
			ref = "main"
			if defaultBranch, err := getRepositoryPackageDefaultBranch(repoSpec.RepoSlug, host); err == nil {
				ref = defaultBranch
			} else {
				addPackageManifestLog.Printf("failed to resolve default branch for %s (host=%q), falling back to %q: %v", repoSpec.RepoSlug, host, ref, err)
			}
		}
	}
	packagePath := strings.Trim(repoSpec.PackagePath, "/")

	manifestPath, manifestContent, err := loadRepositoryPackageManifestFile(ctx, owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	manifest, warnings, err := parseRepositoryPackageManifest(manifestPath, manifestContent)
	if err != nil {
		return nil, err
	}

	includeInstallablePaths, includeSkillDirs, includeAgentFiles := splitManifestIncludePaths(manifest.Includes)
	includeInstallablePaths = append(includeInstallablePaths, manifest.Files...)

	installationSources := normalizePackageInstallablePaths(includeInstallablePaths, packagePath)
	if len(installationSources) == 0 {
		installationSources, err = scanRepositoryPackageInstallablePaths(ctx, owner, repo, packagePath, ref, host)
		if err != nil {
			return nil, err
		}
	}
	if err := validateUniqueManifestWorkflowFilenames(installationSources, manifestPath); err != nil {
		return nil, err
	}

	docsPath, err := resolveRepositoryPackageDocsPath(ctx, owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	// Resolve skill files: explicit from manifest or auto-scanned.
	explicitSkillDirs := append([]string{}, manifest.Skills...)
	explicitSkillDirs = append(explicitSkillDirs, includeSkillDirs...)
	skillFiles, skillWarnings, err := resolvePackageSkillFiles(ctx, owner, repo, packagePath, ref, host, explicitSkillDirs)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, skillWarnings...)

	// Resolve agent files: explicit from manifest or auto-scanned.
	explicitAgentFiles := append([]string{}, manifest.Agents...)
	explicitAgentFiles = append(explicitAgentFiles, includeAgentFiles...)
	agentFiles, agentWarnings, err := resolvePackageAgentFiles(ctx, owner, repo, packagePath, ref, host, explicitAgentFiles)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, agentWarnings...)

	if len(installationSources) == 0 && len(skillFiles) == 0 && len(agentFiles) == 0 {
		return nil, fmt.Errorf("repository %q does not contain any installable workflows, skills, or agents (either explicitly declared or auto-discovered)", repositoryPackageIdentifier(repoSpec.RepoSlug, packagePath))
	}

	return &resolvedRepositoryPackage{
		ManifestPath:       manifestPath,
		ResolvedRef:        ref,
		Name:               manifest.Name,
		Emoji:              manifest.Emoji,
		Description:        manifest.Description,
		License:            manifest.License,
		DocsPath:           docsPath,
		InstallationSource: installationSources,
		Bootstrap:          manifest.Bootstrap,
		SkillFiles:         skillFiles,
		AgentFiles:         agentFiles,
		Warnings:           warnings,
	}, nil
}

func loadRepositoryPackageManifestFile(ctx context.Context, owner, repo, packagePath, ref, host string) (string, []byte, error) {
	manifestPath := joinRepositoryPackagePath(packagePath, repositoryPackageManifestFileName)
	repoSlug := owner + "/" + repo
	packageID := repositoryPackageIdentifier(repoSlug, packagePath)
	content, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, manifestPath, ref, host)
	if err != nil {
		if !isRepositoryFileNotFound(err) {
			return "", nil, fmt.Errorf("failed to read manifest %q from %s/%s@%s: %w", manifestPath, owner, repo, ref, err)
		}
		if packagePath != "" {
			return "", nil, fmt.Errorf("%w: repository %q is not a valid Agentic Workflow package: no aw.yml manifest found in %q; add %s or use an explicit workflow path", errRepositoryPackageManifestNotFound, packageID, packagePath, manifestPath)
		}
		return "", nil, fmt.Errorf("%w: repository %q is not a valid Agentic Workflow package: no aw.yml manifest found at the repository root; add aw.yml or use an explicit workflow path", errRepositoryPackageManifestNotFound, repoSlug)
	}

	return manifestPath, content, nil
}

type repositoryPackageManifest struct {
	ManifestVersion string
	MinVersion      string
	Name            string
	Emoji           string
	Description     string
	License         string
	Includes        []string
	Files           []string
	Bootstrap       *repositoryPackageBootstrap
	Skills          []string // skill directory paths (e.g. "skills/my-skill")
	Agents          []string // agent .md file paths (e.g. "agents/my-agent.md")
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

	// Validate name before schema validation to provide a clear error message for
	// the most common manifest authoring error (missing or empty name).
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

	if emoji, ok := stringValue(root["emoji"]); ok {
		manifest.Emoji = emoji
	}

	if license, ok := stringValue(root["license"]); ok {
		manifest.License = license
	}

	if includesValue, ok := root["includes"]; ok {
		includes, includeWarnings := extractManifestIncludes(includesValue, manifestPath)
		manifest.Includes = includes
		warnings = append(warnings, includeWarnings...)
	}

	if filesValue, ok := root["files"]; ok {
		files, fileWarnings := extractManifestFiles(filesValue, manifestPath)
		manifest.Files = files
		warnings = append(warnings, fileWarnings...)
		if len(files) > 0 {
			warnings = append(warnings, fmt.Sprintf("Field 'files' in %s is deprecated; use 'includes' instead.", manifestPath))
			warnings = append(warnings, "Codemod suggestion:\n"+formatIncludesCodemodSuggestion(codemodManifestFilesToIncludes(files)))
		}
	}

	if skillsValue, ok := root["skills"]; ok {
		skills, skillWarnings := extractManifestSkillDirs(skillsValue, manifestPath)
		manifest.Skills = skills
		warnings = append(warnings, skillWarnings...)
	}

	if agentsValue, ok := root["agents"]; ok {
		agents, agentWarnings := extractManifestAgentFiles(agentsValue, manifestPath)
		manifest.Agents = agents
		warnings = append(warnings, agentWarnings...)
	}

	if bootstrapValue, ok := root["bootstrap"]; ok {
		warnings = append(warnings, "Using experimental feature: manifest.bootstrap")
		bootstrap, err := extractManifestBootstrap(bootstrapValue, manifestPath)
		if err != nil {
			return nil, nil, err
		}
		manifest.Bootstrap = bootstrap
	}

	return manifest, warnings, nil
}

func extractManifestIncludes(value any, manifestPath string) ([]string, []string) {
	var rawIncludes []string
	switch includes := value.(type) {
	case []any:
		for _, item := range includes {
			if include, ok := stringValue(item); ok {
				rawIncludes = append(rawIncludes, include)
			}
		}
	case []string:
		rawIncludes = append(rawIncludes, includes...)
	default:
		return nil, []string{fmt.Sprintf("Ignoring includes entry in %s because it is not a list of strings", manifestPath)}
	}

	var warnings []string
	normalized := make([]string, 0, len(rawIncludes))
	seen := make(map[string]struct{})
	for _, include := range rawIncludes {
		if !isSupportedManifestIncludePath(include) {
			warnings = append(warnings, fmt.Sprintf("Ignoring includes entry %q in %s: use workflow files (workflows/, agentic-workflows/, .github/workflows/), skill directories (skills/, .github/skills/), or agent markdown files (agents/, .github/agents/)", include, manifestPath))
			continue
		}
		if _, exists := seen[include]; exists {
			continue
		}
		seen[include] = struct{}{}
		normalized = append(normalized, include)
	}
	return normalized, warnings
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
			warnings = append(warnings, fmt.Sprintf("Ignoring files entry %q in %s: supported files are markdown (.md) files under workflows/, agentic-workflows/, or .github/workflows/, or action workflow (.yml) files under .github/workflows/", file, manifestPath))
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

func codemodManifestFilesToIncludes(files []string) []string {
	converted := make([]string, 0, len(files))
	for _, file := range files {
		converted = append(converted, path.Clean(filepath.ToSlash(file)))
	}
	return converted
}

func formatIncludesCodemodSuggestion(paths []string) string {
	if len(paths) == 0 {
		return "includes: []"
	}
	lines := []string{"includes:"}
	for _, p := range paths {
		lines = append(lines, "  - "+p)
	}
	return strings.Join(lines, "\n")
}

func splitManifestIncludePaths(includes []string) (installable, skillDirs, agentFiles []string) {
	for _, include := range includes {
		switch {
		case isSupportedSkillDirPath(include):
			skillDirs = append(skillDirs, include)
		case isSupportedAgentFilePath(include):
			agentFiles = append(agentFiles, include)
		case isSupportedPackageInstallablePath(include):
			installable = append(installable, include)
		}
	}
	return installable, skillDirs, agentFiles
}

// extractManifestSkillDirs parses the skills array from an aw.yml manifest, validating
// and normalizing each entry. Each entry must be a path under skills/ that represents
// the directory for a skill (e.g. "skills/my-skill").
func extractManifestSkillDirs(value any, manifestPath string) ([]string, []string) {
	var rawDirs []string
	switch dirs := value.(type) {
	case []any:
		for _, item := range dirs {
			if dir, ok := stringValue(item); ok {
				rawDirs = append(rawDirs, dir)
			}
		}
	case []string:
		rawDirs = append(rawDirs, dirs...)
	default:
		return nil, []string{fmt.Sprintf("Ignoring skills entry in %s because it is not a list of strings", manifestPath)}
	}

	var warnings []string
	normalized := make([]string, 0, len(rawDirs))
	seen := make(map[string]struct{})
	for _, dir := range rawDirs {
		if !isSupportedSkillDirPath(dir) {
			warnings = append(warnings, fmt.Sprintf("Ignoring skills entry %q in %s: skill entries must be directory paths under skills/ (e.g. \"skills/my-skill\")", dir, manifestPath))
			continue
		}
		if _, exists := seen[dir]; exists {
			continue
		}
		seen[dir] = struct{}{}
		normalized = append(normalized, dir)
	}
	return normalized, warnings
}

// extractManifestAgentFiles parses the agents array from an aw.yml manifest, validating
// and normalizing each entry. Each entry must be a .md file path under agents/.
func extractManifestAgentFiles(value any, manifestPath string) ([]string, []string) {
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
		return nil, []string{fmt.Sprintf("Ignoring agents entry in %s because it is not a list of strings", manifestPath)}
	}

	var warnings []string
	normalized := make([]string, 0, len(rawFiles))
	seen := make(map[string]struct{})
	for _, file := range rawFiles {
		if !isSupportedAgentFilePath(file) {
			warnings = append(warnings, fmt.Sprintf("Ignoring agents entry %q in %s: agent entries must be .md file paths under agents/ (e.g. \"agents/my-agent.md\")", file, manifestPath))
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

// isSupportedSkillDirPath returns true when p is a valid skill directory path.
// Valid skill directory paths must be directly under skills/ (e.g. "skills/my-skill")
// with no further nesting.
func isSupportedSkillDirPath(p string) bool {
	cleaned := path.Clean(filepath.ToSlash(p))
	if !isSupportedSkillDirectoryPrefix(cleaned) {
		return false
	}
	root := skillDirectoryRoot(cleaned)
	remaining := strings.TrimPrefix(cleaned, root+"/")
	// Must have exactly one path component (direct child of skills/)
	return remaining != "" && !strings.Contains(remaining, "/")
}

// isSupportedAgentFilePath returns true when p is a valid agent file path.
// Valid agent paths must be .md files directly under agents/ (e.g. "agents/my-agent.md").
func isSupportedAgentFilePath(p string) bool {
	cleaned := path.Clean(filepath.ToSlash(p))
	if !isSupportedAgentDirectoryPrefix(cleaned) {
		return false
	}
	if !strings.HasSuffix(strings.ToLower(cleaned), ".md") {
		return false
	}
	root := agentDirectoryRoot(cleaned)
	remaining := strings.TrimPrefix(cleaned, root+"/")
	// Must be a direct child of agents/ (no subdirectories)
	return remaining != "" && !strings.Contains(remaining, "/")
}

func isSupportedManifestIncludePath(p string) bool {
	return isSupportedPackageInstallablePath(p) || isSupportedSkillDirPath(p) || isSupportedAgentFilePath(p)
}

func isSupportedSkillDirectoryPrefix(cleaned string) bool {
	return strings.HasPrefix(cleaned, packageSkillsDirectory+"/") ||
		strings.HasPrefix(cleaned, constants.GithubDir+packageSkillsDirectory+"/")
}

func skillDirectoryRoot(cleaned string) string {
	switch {
	case strings.HasPrefix(cleaned, constants.GithubDir+packageSkillsDirectory+"/"):
		return constants.GithubDir + packageSkillsDirectory
	default:
		return packageSkillsDirectory
	}
}

func isSupportedAgentDirectoryPrefix(cleaned string) bool {
	return strings.HasPrefix(cleaned, packageAgentsDirectory+"/") ||
		strings.HasPrefix(cleaned, constants.GithubDir+packageAgentsDirectory+"/")
}

func agentDirectoryRoot(cleaned string) string {
	switch {
	case strings.HasPrefix(cleaned, constants.GithubDir+packageAgentsDirectory+"/"):
		return constants.GithubDir + packageAgentsDirectory
	default:
		return packageAgentsDirectory
	}
}

// resolvePackageSkillFiles returns the list of resolvedPackageSkillFile for a package.
// Manifest-specified skills (explicitSkillDirs) are resolved first. After that, the
// skills/ directory is always auto-scanned for any additional skill subdirectories that
// contain a SKILL.md file but are not already covered by the manifest. Each skill folder
// is traversed recursively so that all nested files are included.
func resolvePackageSkillFiles(ctx context.Context, owner, repo, packagePath, ref, host string, explicitSkillDirs []string) ([]resolvedPackageSkillFile, []string, error) {
	// seenSkillDirs tracks full skill directories already added so that auto-scanned
	// duplicates of manifest-specified skills are not added a second time.
	seenSkillDirs := make(map[string]struct{})
	var warnings []string

	// Step 1: resolve manifest skills first (explicit dirs).
	var manifestSkillDirs []string
	for _, dir := range explicitSkillDirs {
		manifestSkillDirs = append(manifestSkillDirs, joinRepositoryPackagePath(packagePath, dir))
	}

	// Step 2: always auto-scan and append any skills not already in the manifest.
	autoScanned, err := scanPackageSkillDirs(ctx, owner, repo, packagePath, ref, host)
	if err != nil {
		// Auto-scan is supplementary for manifest-declared skills; preserve manifest
		// resolution even when scan fails transiently.
		if len(manifestSkillDirs) > 0 {
			warnings = append(warnings, fmt.Sprintf("failed to auto-scan skills directory, proceeding with manifest skills only: %v", err))
		} else {
			return nil, nil, err
		}
	}

	// Build the final ordered list: manifest skills first, then auto-scanned extras.
	var skillDirs []string
	appendIfNew := func(dir string) {
		if _, exists := seenSkillDirs[dir]; !exists {
			seenSkillDirs[dir] = struct{}{}
			skillDirs = append(skillDirs, dir)
		}
	}
	for _, dir := range manifestSkillDirs {
		appendIfNew(dir)
	}
	for _, dir := range autoScanned {
		appendIfNew(dir)
	}

	// manifestSkillDirSet is used to know which dirs require a SKILL.md marker check.
	manifestSkillDirSet := make(map[string]struct{}, len(manifestSkillDirs))
	for _, d := range manifestSkillDirs {
		manifestSkillDirSet[d] = struct{}{}
	}

	var skillFiles []resolvedPackageSkillFile
	for _, skillDir := range skillDirs {
		// For skills that came from the manifest, validate that the SKILL.md marker
		// exists so that typos in the manifest surface as clear warnings.
		if _, fromManifest := manifestSkillDirSet[skillDir]; fromManifest {
			markerPath := joinRepositoryPackagePath(skillDir, packageSkillMarkerFile)
			if _, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, markerPath, ref, host); err != nil {
				if isRepositoryFileNotFound(err) {
					warnings = append(warnings, fmt.Sprintf("Skill directory %q is missing required %s marker file", skillDir, packageSkillMarkerFile))
					continue
				}
				return nil, nil, fmt.Errorf("failed to validate skill marker %q: %w", markerPath, err)
			}
		}
		skillName := filepath.Base(skillDir)
		// Use recursive listing so that the entire skill folder (including any
		// subdirectories) is copied, not just the top-level files.
		files, err := listPackageDirFilesRecursivelyForHost(ctx, owner, repo, ref, skillDir, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				warnings = append(warnings, fmt.Sprintf("Skill directory %q not found in package, skipping", skillDir))
				continue
			}
			return nil, nil, fmt.Errorf("failed to list files in skill directory %q: %w", skillDir, err)
		}
		for _, file := range files {
			skillFiles = append(skillFiles, resolvedPackageSkillFile{
				SourcePath: file,
				SkillName:  skillName,
			})
		}
	}
	return skillFiles, warnings, nil
}

// resolvePackageAgentFiles returns the list of agent file source paths for a package.
// If explicitAgentFiles is non-empty it is used; otherwise the agents/ directory is
// auto-scanned for .md files.
func resolvePackageAgentFiles(ctx context.Context, owner, repo, packagePath, ref, host string, explicitAgentFiles []string) ([]string, []string, error) {
	if len(explicitAgentFiles) > 0 {
		var agentFiles []string
		for _, f := range explicitAgentFiles {
			agentFiles = append(agentFiles, joinRepositoryPackagePath(packagePath, f))
		}
		return agentFiles, nil, nil
	}

	var agentFiles []string
	for _, root := range []string{packageAgentsDirectory, constants.GithubDir + packageAgentsDirectory} {
		agentsDir := joinRepositoryPackagePath(packagePath, root)
		files, err := listPackageDirFilesForHost(ctx, owner, repo, ref, agentsDir, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return nil, nil, fmt.Errorf("failed to scan agents directory %q: %w", agentsDir, err)
		}
		for _, f := range files {
			if strings.HasSuffix(strings.ToLower(f), ".md") {
				agentFiles = append(agentFiles, f)
			}
		}
	}
	return agentFiles, nil, nil
}

// scanPackageSkillDirs auto-scans the skills/ directory of a package and returns the paths
// of skill subdirectories (those that contain a SKILL.md file).
func scanPackageSkillDirs(ctx context.Context, owner, repo, packagePath, ref, host string) ([]string, error) {
	var skillDirs []string
	for _, root := range []string{packageSkillsDirectory, constants.GithubDir + packageSkillsDirectory} {
		skillsDir := joinRepositoryPackagePath(packagePath, root)
		subdirs, err := listPackageDirSubdirsForHost(ctx, owner, repo, ref, skillsDir, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan skills directory %q: %w", skillsDir, err)
		}
		for _, subdir := range subdirs {
			markerPath := joinRepositoryPackagePath(subdir, packageSkillMarkerFile)
			if _, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, markerPath, ref, host); err == nil {
				skillDirs = append(skillDirs, subdir)
			}
		}
	}
	return skillDirs, nil
}

func scanRepositoryPackageInstallablePaths(ctx context.Context, owner, repo, packagePath, ref, host string) ([]string, error) {
	var collected []string
	seen := make(map[string]struct{})

	for _, sourceDir := range packageSourceDirectories {
		sourcePath := joinRepositoryPackagePath(packagePath, sourceDir)
		files, err := listPackageWorkflowFilesForHost(ctx, owner, repo, ref, sourcePath, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("failed to scan %q in %s/%s@%s: %w", sourcePath, owner, repo, ref, err)
		}

		for _, file := range files {
			// listPackageWorkflowFilesForHost returns full repo-root-relative paths
			// (e.g. "folder/workflows/foo.md" when scanning "folder/workflows/").
			// isSupportedPackageInstallablePath expects package-relative paths, so
			// strip the package prefix before validation for nested bundles.
			pathToValidate := file
			if packagePath != "" {
				pathToValidate = strings.TrimPrefix(file, packagePath+"/")
			}
			if !isSupportedPackageInstallablePath(pathToValidate) {
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

func resolveRepositoryPackageDocsPath(ctx context.Context, owner, repo, packagePath, ref, host string) (string, error) {
	readmePath := joinRepositoryPackagePath(packagePath, "README.md")
	repoSlug := owner + "/" + repo
	packageID := repositoryPackageIdentifier(repoSlug, packagePath)
	if _, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, readmePath, ref, host); err == nil {
		return readmePath, nil
	} else if isRepositoryFileNotFound(err) {
		return "", fmt.Errorf("repository %q is not a valid Agentic Workflow package: missing required README.md at %q", packageID, readmePath)
	} else {
		return "", fmt.Errorf("failed to read package README %q from %s/%s@%s: %w", readmePath, owner, repo, ref, err)
	}
}

func repositoryPackageIdentifier(repoSlug, packagePath string) string {
	if packagePath == "" {
		return repoSlug
	}
	return repoSlug + "/" + packagePath
}

func normalizePackageInstallablePaths(paths []string, packagePath string) []string {
	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{})
	for _, path := range paths {
		if !isSupportedPackageInstallablePath(path) {
			continue
		}
		// Paths under .github/ are treated as repo-root-relative even in nested
		// bundles (e.g. a bundle at "dependabot/" with ".github/workflows/foo.md"
		// refers to the repository-root ".github/workflows/foo.md", not to
		// "dependabot/.github/workflows/foo.md"). All other paths (e.g. workflows/,
		// agentic-workflows/) remain relative to the package root.
		if packagePath != "" && strings.HasPrefix(path, constants.GithubDir) {
			path = filepath.ToSlash(path)
		} else {
			path = joinRepositoryPackagePath(packagePath, path)
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized
}

func validateManifestInstallableWorkflowPrivacy(manifestPath string, installationSources []string, readWorkflow func(string) ([]byte, error)) error {
	for _, installationSource := range installationSources {
		if isActionWorkflowPath(installationSource) {
			continue
		}

		content, err := readWorkflow(installationSource)
		if err != nil {
			return fmt.Errorf("invalid Agentic Workflow manifest %q: %w", manifestPath, err)
		}

		privateValue, hasPrivate := ExtractWorkflowPrivateSetting(string(content))
		if hasPrivate && privateValue {
			return fmt.Errorf("invalid Agentic Workflow manifest %q: workflow %q sets private: true and cannot be included because private workflows cannot be added", manifestPath, installationSource)
		}
	}

	return nil
}

func isSupportedPackageInstallablePath(p string) bool {
	// Normalize separators to forward slashes (consistent with joinRepositoryPackagePath) then
	// clean to reject path traversal (e.g. "workflows/../README.md" → "README.md").
	cleaned := path.Clean(filepath.ToSlash(p))
	lowerCleaned := strings.ToLower(cleaned)
	if strings.HasSuffix(lowerCleaned, ".md") {
		return strings.HasPrefix(cleaned, "workflows/") ||
			strings.HasPrefix(cleaned, "agentic-workflows/") ||
			strings.HasPrefix(cleaned, constants.WorkflowsDirSlash)
	}
	if isActionWorkflowPath(cleaned) {
		if !strings.HasPrefix(cleaned, constants.WorkflowsDirSlash) {
			return false
		}
		// Reject nested subdirectories: only direct children of .github/workflows/ are allowed.
		remaining := strings.TrimPrefix(cleaned, constants.WorkflowsDirSlash)
		return !strings.Contains(remaining, "/")
	}
	return false
}

func isActionWorkflowPath(p string) bool {
	lowerPath := strings.ToLower(p)
	return strings.HasSuffix(lowerPath, ".yml") && !strings.HasSuffix(lowerPath, ".lock.yml")
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
	if !parser.IsValidGitHubIdentifier(slashParts[0]) || !parser.IsValidGitHubRepositoryName(slashParts[1]) {
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
	return errors.Is(err, errRepositoryPackageFileNotFound)
}

func isRepositoryPackageManifestNotFound(err error) bool {
	return errors.Is(err, errRepositoryPackageManifestNotFound)
}

func isSupportedManifestMinVersion(version string) bool {
	const expectedManifestMinVersionDotCount = 2
	return semverutil.IsActionVersionTag(version) && strings.Count(strings.TrimPrefix(version, "v"), ".") == expectedManifestMinVersionDotCount
}

func validateUniqueManifestWorkflowFilenames(paths []string, manifestPath string) error {
	seen := make(map[string]string, len(paths))
	for _, installPath := range paths {
		if !strings.HasSuffix(strings.ToLower(installPath), ".md") {
			continue
		}
		filenameWithoutExt := strings.TrimSuffix(filepath.Base(installPath), filepath.Ext(installPath))
		key := strings.ToLower(strings.TrimSpace(filenameWithoutExt))
		if key == "" { //nolint:tolowerequalfold
			continue
		}
		if previous, exists := seen[key]; exists {
			return fmt.Errorf("invalid Agentic Workflow manifest %q: duplicate workflow filename %q in files entries %q and %q (filenames must be unique across a package)", manifestPath, filenameWithoutExt, previous, installPath)
		}
		seen[key] = installPath
	}
	return nil
}

func downloadRepositoryPackageFileFromGitHubForHost(ctx context.Context, owner, repo, path, ref, host string) ([]byte, error) {
	content, err := parser.DownloadFileFromGitHubForHost(ctx, owner, repo, path, ref, host)
	return content, normalizeRepositoryPackageRemoteError(err)
}

func listRepositoryPackageWorkflowFilesForHost(ctx context.Context, owner, repo, ref, workflowPath, host string) ([]string, error) {
	files, err := parser.ListWorkflowFilesForHost(ctx, owner, repo, ref, workflowPath, host)
	return files, normalizeRepositoryPackageRemoteError(err)
}

func listRepositoryPackageDirFilesForHost(ctx context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
	files, err := parser.ListDirAllFilesForHost(ctx, owner, repo, ref, dirPath, host)
	return files, normalizeRepositoryPackageRemoteError(err)
}

func listRepositoryPackageDirFilesRecursivelyForHost(ctx context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
	files, err := parser.ListDirAllFilesRecursivelyForHost(ctx, owner, repo, ref, dirPath, host)
	return files, normalizeRepositoryPackageRemoteError(err)
}

func listRepositoryPackageDirSubdirsForHost(ctx context.Context, owner, repo, ref, dirPath, host string) ([]string, error) {
	dirs, err := parser.ListDirSubdirsForHost(ctx, owner, repo, ref, dirPath, host)
	return dirs, normalizeRepositoryPackageRemoteError(err)
}

func normalizeRepositoryPackageRemoteError(err error) error {
	if err == nil || !isRepositoryPackageRemoteNotFound(err) {
		return err
	}
	return packageRemoteNotFoundError{cause: err}
}

func isRepositoryPackageRemoteNotFound(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "404") || strings.Contains(errText, "not found")
}

func resolveRepositoryPackageDefaultBranch(repoSlug, host string) (string, error) {
	args := []string{"api", "/repos/" + repoSlug, "--jq", ".default_branch"}
	var output []byte
	var err error
	if host != "" {
		output, err = workflow.RunGHWithHost("Fetching repo info...", host, args...)
		if err != nil {
			return "", err
		}
	} else {
		output, err = workflow.RunGH("Fetching repo info...", args...)
		if err != nil {
			return "", err
		}
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		targetHost := host
		if targetHost == "" {
			targetHost = "the configured host"
		}
		return "", fmt.Errorf("repository %s on %s returned an empty default branch; ensure the repository exists and is accessible", repoSlug, targetHost)
	}
	return branch, nil
}

// repositoryPackageEffectiveRef returns the effective ref for repository package
// operations. Explicit user-provided versions always win; otherwise this uses a
// previously resolved package ref when available.
func repositoryPackageEffectiveRef(repoSpec *RepoSpec, pkg *resolvedRepositoryPackage) string {
	if repoSpec != nil && repoSpec.Version != "" {
		return repoSpec.Version
	}
	if pkg != nil && pkg.ResolvedRef != "" {
		return pkg.ResolvedRef
	}
	return ""
}

// isGhAwRepository reports whether repoSlug identifies github/gh-aw.
// Matching is case-insensitive and ignores surrounding whitespace.
func isGhAwRepository(repoSlug string) bool {
	return strings.EqualFold(strings.TrimSpace(repoSlug), ghAwRepositorySlug)
}

// resolveRepositoryPackageLatestRelease resolves the latest stable release tag
// for a repository package source.
//
// repoSlug must be in "owner/repo" format. host is an optional explicit GitHub
// hostname (for example "github.com" or a GHES host); when provided, gh API
// calls are executed against that host.
func resolveRepositoryPackageLatestRelease(repoSlug, host string) (string, error) {
	deps := workflowUpdateDeps{
		runReleasesAPI: func(ctx context.Context, repo string) ([]byte, error) {
			args := []string{"api", fmt.Sprintf("/repos/%s/releases", repo), "--jq", ".[].tag_name"}
			if host != "" {
				return workflow.RunGHWithHost("Fetching releases...", host, args...)
			}
			return workflow.RunGHContext(ctx, "Fetching releases...", args...)
		},
	}

	return resolveLatestReleaseWithDeps(context.Background(), deps, repoSlug, "", true, false, 0)
}
