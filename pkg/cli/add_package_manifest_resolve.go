// add_package_manifest_resolve.go: top-level orchestration of resolving a remote package
// (manifest + files) given a repo spec. See add_package_manifest.go for shared types.

package cli

import (
	"context"
	"fmt"
	"strings"
)

func resolveRepositoryPackage(ctx context.Context, repoSpec *RepoSpec, host string) (*resolvedRepositoryPackage, error) {
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

	if len(installationSources) == 0 && len(extensionFiles.skillFiles) == 0 && len(extensionFiles.agentFiles) == 0 {
		return nil, fmt.Errorf("repository %q does not contain any installable workflows, skills, or agents (either explicitly declared or auto-discovered). Add workflows under 'workflows/', skills under 'skills/', or agents under 'agents/', or declare them explicitly in aw.yml", repositoryPackageIdentifier(repoSpec.RepoSlug, packagePath))
	}

	return newResolvedRepositoryPackage(manifestPath, ref, docsPath, manifest, installationSources, extensionFiles, warnings), nil
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
		scanned, err := scanRepositoryPackageInstallablePaths(ctx, owner, repo, packagePath, ref, host)
		if err != nil {
			return nil, nil, nil, err
		}
		installationSources = packageInstallablesFromSourcePaths(scanned)
	}
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

func newResolvedRepositoryPackage(manifestPath, ref, docsPath string, manifest *repositoryPackageManifest, installationSources []resolvedPackageInstallable, extensionFiles *repositoryPackageExtensionFiles, warnings []string) *resolvedRepositoryPackage {
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
		SkillFiles:         extensionFiles.skillFiles,
		AgentFiles:         extensionFiles.agentFiles,
		Warnings:           warnings,
	}
}

func loadRepositoryPackageManifestFile(ctx context.Context, owner, repo, packagePath, ref, host string) (string, []byte, error) {
	manifestPath := joinRepositoryPackagePath(packagePath, repositoryPackageManifestFileName)
	repoSlug := owner + "/" + repo
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
