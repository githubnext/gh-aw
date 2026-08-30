// add_package_manifest_resolve.go: top-level orchestration of resolving a remote package
// (manifest + files) given a repo spec. See add_package_manifest.go for shared types.

package cli

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

func resolveRepositoryPackage(ctx context.Context, repoSpec *RepoSpec, host string) (*resolvedRepositoryPackage, error) {
	addPackageManifestLog.Printf("Resolving repository package %q (packagePath=%q, host=%q)", repoSpec.RepoSlug, repoSpec.PackagePath, host)
	owner, repo, err := splitRepositoryPackageSlug(repoSpec.RepoSlug)
	if err != nil {
		return nil, err
	}
	ref := resolveRepositoryPackageRef(ctx, repoSpec, host)
	packagePath := strings.Trim(repoSpec.PackagePath, "/")

	manifestPath, manifestContent, err := loadRepositoryPackageManifestFile(ctx, owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	manifest, warnings, err := parseRepositoryPackageManifest(manifestPath, manifestContent)
	if err != nil {
		return nil, err
	}

	installationSources, includeSkillDirs, includeAgentFiles, err := resolveRepositoryPackageInstallablePaths(ctx, owner, repo, packagePath, ref, host, manifest, manifestPath)
	if err != nil {
		return nil, err
	}
	resourceFiles, err := resolveRepositoryPackageResourceFiles(ctx, owner, repo, packagePath, ref, host, manifest, installationSources)
	if err != nil {
		return nil, err
	}

	docsPath, err := resolveRepositoryPackageDocsPath(ctx, owner, repo, packagePath, ref, host)
	if err != nil {
		return nil, err
	}

	extensionFiles, err := resolveRepositoryPackageExtensionFiles(ctx, repositoryPackageExtensionFilesOptions{
		owner:             owner,
		repo:              repo,
		packagePath:       packagePath,
		ref:               ref,
		host:              host,
		manifest:          manifest,
		includeSkillDirs:  includeSkillDirs,
		includeAgentFiles: includeAgentFiles,
	})
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, extensionFiles.warnings...)

	if len(installationSources) == 0 && len(resourceFiles) == 0 && len(extensionFiles.skillFiles) == 0 && len(extensionFiles.agentFiles) == 0 {
		return nil, fmt.Errorf("repository %q does not contain any installable workflows, resources, skills, or agents (either explicitly declared or auto-discovered). Add workflows under 'workflows/', resources in aw.yml, skills under 'skills/', or agents under 'agents/', or declare them explicitly in aw.yml", repositoryPackageIdentifier(repoSpec.RepoSlug, packagePath))
	}

	return newResolvedRepositoryPackage(manifestPath, ref, docsPath, manifest, installationSources, resourceFiles, extensionFiles, warnings), nil
}

func resolveRepositoryPackageResourceFiles(ctx context.Context, owner, repo, packagePath, ref, host string, manifest *repositoryPackageManifest, installationSources []resolvedPackageInstallable) ([]resolvedPackageResource, error) {
	resourceFiles := normalizePackageResourcePaths(manifest.Resources, packagePath)
	return appendPackageGraderEvaluatorResources(ctx, owner, repo, ref, host, packagePath, resourceFiles, installationSources)
}

func appendPackageGraderEvaluatorResources(ctx context.Context, owner, repo, ref, host, packagePath string, resourceFiles []resolvedPackageResource, installationSources []resolvedPackageInstallable) ([]resolvedPackageResource, error) {
	seen := make(map[string]string, len(resourceFiles))
	for _, resource := range resourceFiles {
		seen[packageResourceDestinationKey(resource.DestinationPath)] = resource.SourcePath
	}
	addPackageManifestLog.Printf("resolving grader evaluators from %d installable package source(s)", len(installationSources))
	for _, installable := range installationSources {
		if !strings.HasSuffix(strings.ToLower(installable.SourcePath), ".md") {
			continue
		}
		content, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, installable.SourcePath, ref, host)
		if err != nil {
			if isRepositoryFileNotFound(err) {
				addPackageManifestLog.Printf("skipping grader evaluator resource discovery for unavailable package workflow %q: %v", installable.SourcePath, err)
				continue
			}
			return nil, fmt.Errorf("failed to read package workflow %q while resolving grader evaluator resources: %w", installable.SourcePath, err)
		}
		entries, err := extractResourceEntries(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse package workflow %q grader resources: %w", installable.SourcePath, err)
		}
		for _, entry := range entries {
			if !entry.isGraderEvaluator {
				continue
			}
			resource := packageGraderEvaluatorResource(installable, entry.path, packagePath)
			key := packageResourceDestinationKey(resource.DestinationPath)
			if previousSource, exists := seen[key]; exists {
				if previousSource != resource.SourcePath {
					return nil, fmt.Errorf("package workflows reference multiple grader evaluator resources for %q: %q and %q", resource.DestinationPath, previousSource, resource.SourcePath)
				}
				continue
			}
			seen[key] = resource.SourcePath
			resourceFiles = append(resourceFiles, resource)
		}
	}
	return resourceFiles, nil
}

func packageGraderEvaluatorResource(installable resolvedPackageInstallable, runPath, packagePath string) resolvedPackageResource {
	if localPath, ok := strings.CutPrefix(runPath, "./"); ok {
		return resolvedPackageResource{
			SourcePath:      path.Join(path.Dir(installable.SourcePath), localPath),
			DestinationPath: path.Join(path.Dir(installable.DestinationPath), localPath),
		}
	}
	sourcePath := joinRepositoryPackagePath(packagePath, runPath)
	if localWorkflowsPath, ok := strings.CutPrefix(runPath, constants.WorkflowsDirSlash); ok {
		sourcePath = path.Join(path.Dir(installable.SourcePath), localWorkflowsPath)
	}
	return resolvedPackageResource{
		SourcePath:      sourcePath,
		DestinationPath: runPath,
	}
}

func splitRepositoryPackageSlug(repoSlug string) (string, string, error) {
	parts := strings.SplitN(repoSlug, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("repository slug %q is not in 'owner/repo' format. Example: owner/repo", repoSlug)
	}
	return parts[0], parts[1], nil
}

func resolveRepositoryPackageRef(ctx context.Context, repoSpec *RepoSpec, host string) string {
	// At manifest-fetch time there is no resolved package metadata yet.
	ref := repositoryPackageEffectiveRef(repoSpec, nil)
	if ref != "" {
		return ref
	}
	if isGhAwRepository(repoSpec.RepoSlug) {
		if latestRelease, err := getRepositoryPackageLatestRelease(ctx, repoSpec.RepoSlug, host); err == nil {
			return latestRelease
		} else {
			addPackageManifestLog.Printf("failed to resolve latest release for %s (host=%q): %v", repoSpec.RepoSlug, host, err)
		}
	}
	ref = "main"
	if defaultBranch, err := getRepositoryPackageDefaultBranch(ctx, repoSpec.RepoSlug, host); err == nil {
		ref = defaultBranch
	} else {
		addPackageManifestLog.Printf("failed to resolve default branch for %s (host=%q), falling back to %q: %v", repoSpec.RepoSlug, host, ref, err)
	}
	return ref
}

func resolveRepositoryPackageInstallablePaths(ctx context.Context, owner, repo, packagePath, ref, host string, manifest *repositoryPackageManifest, manifestPath string) ([]resolvedPackageInstallable, []string, []string, error) {
	includeInstallablePaths, includeSkillDirs, includeAgentFiles := splitManifestIncludePaths(manifest.Includes)
	includeInstallablePaths = append(includeInstallablePaths, manifestIncludesFromPaths(manifest.Files)...)

	installationSources := normalizePackageInstallablePaths(includeInstallablePaths, packagePath)
	if len(installationSources) == 0 {
		addPackageManifestLog.Print("No explicit installable paths in manifest, scanning repository for installables")
		scanned, err := scanRepositoryPackageInstallablePaths(ctx, owner, repo, packagePath, ref, host)
		if err != nil {
			return nil, nil, nil, err
		}
		installationSources = packageInstallablesFromSourcePaths(scanned)
	}
	addPackageManifestLog.Printf("Resolved %d installable source(s) for package", len(installationSources))
	if err := validateUniqueManifestWorkflowFilenames(installationSources, manifestPath); err != nil {
		return nil, nil, nil, err
	}
	if err := validateUniqueManifestInstallDestinations(installationSources, manifestPath); err != nil {
		return nil, nil, nil, err
	}
	return installationSources, includeSkillDirs, includeAgentFiles, nil
}

type repositoryPackageExtensionFiles struct {
	skillFiles []resolvedPackageSkillFile
	agentFiles []string
	warnings   []string
}

type repositoryPackageExtensionFilesOptions struct {
	owner             string
	repo              string
	packagePath       string
	ref               string
	host              string
	manifest          *repositoryPackageManifest
	includeSkillDirs  []string
	includeAgentFiles []string
}

func resolveRepositoryPackageExtensionFiles(ctx context.Context, options repositoryPackageExtensionFilesOptions) (*repositoryPackageExtensionFiles, error) {
	// Resolve skill files: explicit from manifest or auto-scanned.
	explicitSkillDirs := append([]string{}, options.manifest.Skills...)
	explicitSkillDirs = append(explicitSkillDirs, options.includeSkillDirs...)
	skillFiles, skillWarnings, err := resolvePackageSkillFiles(ctx, options.owner, options.repo, options.packagePath, options.ref, options.host, explicitSkillDirs)
	if err != nil {
		return nil, err
	}

	// Resolve agent files: explicit from manifest or auto-scanned.
	explicitAgentFiles := append([]string{}, options.manifest.Agents...)
	explicitAgentFiles = append(explicitAgentFiles, options.includeAgentFiles...)
	agentFiles, agentWarnings, err := resolvePackageAgentFiles(ctx, options.owner, options.repo, options.packagePath, options.ref, options.host, explicitAgentFiles)
	if err != nil {
		return nil, err
	}

	warnings := append(skillWarnings, agentWarnings...)
	return &repositoryPackageExtensionFiles{
		skillFiles: skillFiles,
		agentFiles: agentFiles,
		warnings:   warnings,
	}, nil
}

func newResolvedRepositoryPackage(manifestPath, ref, docsPath string, manifest *repositoryPackageManifest, installationSources []resolvedPackageInstallable, resourceFiles []resolvedPackageResource, extensionFiles *repositoryPackageExtensionFiles, warnings []string) *resolvedRepositoryPackage {
	return &resolvedRepositoryPackage{
		ManifestPath:       manifestPath,
		ResolvedRef:        ref,
		Name:               manifest.Name,
		Emoji:              manifest.Emoji,
		Description:        manifest.Description,
		License:            manifest.License,
		DocsPath:           docsPath,
		InstallationSource: installationSources,
		ResourceFiles:      resourceFiles,
		Bootstrap:          manifest.Bootstrap,
		SkillFiles:         extensionFiles.skillFiles,
		AgentFiles:         extensionFiles.agentFiles,
		Warnings:           warnings,
	}
}

func loadRepositoryPackageManifestFile(ctx context.Context, owner, repo, packagePath, ref, host string) (string, []byte, error) {
	manifestPath := joinRepositoryPackagePath(packagePath, repositoryPackageManifestFileName)
	repoSlug := fmt.Sprintf("%s/%s", owner, repo)
	packageID := repositoryPackageIdentifier(repoSlug, packagePath)
	content, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, manifestPath, ref, host)
	if err != nil {
		if !isRepositoryFileNotFound(err) {
			return "", nil, fmt.Errorf("failed to read manifest %q from %s/%s@%s (check the repository, ref, and network connectivity): %w", manifestPath, owner, repo, ref, err)
		}
		if packagePath != "" {
			return "", nil, fmt.Errorf("%w: repository %q is not a valid Agentic Workflow package: no aw.yml manifest found in %q. Add %s or use an explicit workflow path", errRepositoryPackageManifestNotFound, packageID, packagePath, manifestPath)
		}
		return "", nil, fmt.Errorf("%w: repository %q is not a valid Agentic Workflow package: no aw.yml manifest found at the repository root. Add aw.yml or use an explicit workflow path", errRepositoryPackageManifestNotFound, repoSlug)
	}

	return manifestPath, content, nil
}
