package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var updateManifestLog = logger.New("cli:update_manifest")

func parseManifestSourceSpec(source string) (*RepoSpec, bool, error) {
	repoSpec, ok, err := parseRepositoryPackageSpec(strings.TrimSpace(source))
	if !ok {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("invalid manifest source %q: %w", source, err)
	}
	if repoSpec == nil {
		return nil, false, nil
	}
	return repoSpec, true, nil
}

func manifestSourceWithRef(repoSpec *RepoSpec, ref string) string {
	base := repositoryPackageIdentifier(repoSpec.RepoSlug, repoSpec.PackagePath)
	if ref == "" {
		return base
	}
	return base + "@" + ref
}

func manifestWorkflowPathByName(paths []string) map[string]string {
	byName := make(map[string]string, len(paths))
	for _, p := range paths {
		if !strings.HasSuffix(strings.ToLower(p), ".md") {
			continue
		}
		workflowID := normalizeWorkflowID(filepath.Base(p))
		byName[workflowID] = p
	}
	return byName
}

func updateManifestWorkflowGroup(ctx context.Context, source string, grouped []*workflowWithSource, opts UpdateWorkflowsOptions) ([]string, []updateFailure) {
	updateManifestLog.Printf("updateManifestWorkflowGroup: source=%s, workflows=%d, force=%v, no_merge=%v", source, len(grouped), opts.Force, opts.NoMerge)
	var successes []string
	var failures []updateFailure

	if len(grouped) == 0 {
		return successes, failures
	}

	repoSpec, _, err := parseManifestSourceSpec(source)
	if err != nil {
		for _, wf := range grouped {
			failures = append(failures, updateFailure{Name: wf.Name, Error: err.Error()})
		}
		return successes, failures
	}
	if repoSpec == nil {
		return successes, failures
	}

	currentRef := repoSpec.Version
	if currentRef == "" {
		currentRef = "main"
	}
	latestRef, err := resolveLatestRefFn(ctx, repoSpec.RepoSlug, currentRef, opts.AllowMajor, opts.Verbose, opts.CoolDown)
	if err != nil {
		updateManifestLog.Printf("Failed to resolve latest manifest ref for %s: %v", repoSpec.RepoSlug, err)
		for _, wf := range grouped {
			failures = append(failures, updateFailure{Name: wf.Name, Error: fmt.Sprintf("failed to resolve latest manifest ref: %v", err)})
		}
		return successes, failures
	}
	updateManifestLog.Printf("Resolved manifest refs: current=%s, latest=%s", currentRef, latestRef)
	sourceFieldRef := latestRef
	// Preserve branch-tracking behavior: when source points to a branch, keep the
	// branch name in source so future updates continue following that branch.
	// For tags/SHAs, pin to the resolved latest ref.
	if isBranchRef(currentRef) {
		sourceFieldRef = currentRef
	}

	currentPkg, err := resolveRepositoryPackage(&RepoSpec{
		RepoSlug:    repoSpec.RepoSlug,
		PackagePath: repoSpec.PackagePath,
		Version:     currentRef,
	}, "")
	if err != nil {
		for _, wf := range grouped {
			failures = append(failures, updateFailure{Name: wf.Name, Error: fmt.Sprintf("failed to resolve current manifest package: %v", err)})
		}
		return successes, failures
	}
	latestPkg, err := resolveRepositoryPackage(&RepoSpec{
		RepoSlug:    repoSpec.RepoSlug,
		PackagePath: repoSpec.PackagePath,
		Version:     latestRef,
	}, "")
	if err != nil {
		for _, wf := range grouped {
			failures = append(failures, updateFailure{Name: wf.Name, Error: fmt.Sprintf("failed to resolve latest manifest package: %v", err)})
		}
		return successes, failures
	}

	currentByName := manifestWorkflowPathByName(currentPkg.InstallationSource)
	latestByName := manifestWorkflowPathByName(latestPkg.InstallationSource)
	existingByName := make(map[string]*workflowWithSource, len(grouped))
	for _, wf := range grouped {
		existingByName[wf.Name] = wf
	}

	manifestSource := manifestSourceWithRef(repoSpec, sourceFieldRef)
	for name, wf := range existingByName {
		latestPath, exists := latestByName[name]
		if !exists {
			if err := removeManifestManagedWorkflow(wf.Path); err != nil {
				failures = append(failures, updateFailure{Name: wf.Name, Error: err.Error()})
				continue
			}
			successes = append(successes, wf.Name)
			continue
		}

		oldPath := currentByName[name]
		if oldPath == "" {
			oldPath = latestPath
		}
		if err := updateManifestManagedWorkflow(ctx, wf, repoSpec.RepoSlug, oldPath, latestPath, currentRef, latestRef, manifestSource, opts); err != nil {
			failures = append(failures, updateFailure{Name: wf.Name, Error: err.Error()})
			continue
		}
		successes = append(successes, wf.Name)
	}

	targetDir := filepath.Dir(grouped[0].Path)
	for name, latestPath := range latestByName {
		if _, exists := existingByName[name]; exists {
			continue
		}
		if err := addManifestManagedWorkflow(ctx, targetDir, name, repoSpec.RepoSlug, latestPath, latestRef, manifestSource, opts); err != nil {
			failures = append(failures, updateFailure{Name: name, Error: err.Error()})
			continue
		}
		successes = append(successes, name)
	}

	return successes, failures
}

func removeManifestManagedWorkflow(workflowPath string) error {
	updateManifestLog.Printf("Removing manifest-managed workflow no longer in manifest: %s", filepath.Base(workflowPath))
	if err := os.Remove(workflowPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove workflow %s: %w", filepath.Base(workflowPath), err)
	}
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file %s: %w", filepath.Base(lockPath), err)
	}
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed workflow no longer listed in manifest: "+filepath.Base(workflowPath)))
	return nil
}

func updateManifestManagedWorkflow(ctx context.Context, wf *workflowWithSource, repo, currentPath, latestPath, currentRef, latestRef, manifestSource string, opts UpdateWorkflowsOptions) error {
	updateManifestLog.Printf("Updating manifest-managed workflow %s: %s@%s -> %s@%s", wf.Name, currentPath, currentRef, latestPath, latestRef)
	sourceSpecCurrent := sourceSpecWithRef(&SourceSpec{Repo: repo, Path: currentPath}, currentRef)
	newContent, err := downloadWorkflowContentFn(ctx, repo, latestPath, latestRef, opts.Verbose)
	if err != nil {
		return fmt.Errorf("failed to download workflow %s/%s@%s: %w", repo, latestPath, latestRef, err)
	}

	if !opts.Force && currentRef == latestRef && currentPath == latestPath {
		sourceContent, err := downloadWorkflowContentFn(ctx, repo, currentPath, currentRef, opts.Verbose)
		if err == nil {
			currentContent, readErr := os.ReadFile(wf.Path)
			if readErr == nil && !hasLocalModifications(string(sourceContent), string(currentContent), sourceSpecCurrent, filepath.Dir(wf.Path), opts.Verbose) {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", wf.Name, shortRef(currentRef))))
				return nil
			}
		}
	}

	merge := !opts.NoMerge
	var finalContent string
	var hasConflicts bool
	if merge {
		baseContent, err := downloadWorkflowContentFn(ctx, repo, currentPath, currentRef, opts.Verbose)
		if err != nil {
			updateManifestLog.Printf("Cannot fetch base for 3-way merge of %s, falling back to overwrite: %v", wf.Name, err)
			merge = false
		} else {
			currentContent, err := os.ReadFile(wf.Path)
			if err != nil {
				return fmt.Errorf("failed to read current workflow: %w", err)
			}
			newSourceSpec := sourceSpecWithRef(&SourceSpec{Repo: repo, Path: latestPath}, latestRef)
			mergedContent, conflicts, mergeErr := MergeWorkflowContent(string(baseContent), string(currentContent), string(newContent), sourceSpecCurrent, newSourceSpec, wf.Path, opts.Verbose)
			if mergeErr != nil {
				return fmt.Errorf("failed to merge workflow content: %w", mergeErr)
			}
			finalContent = mergedContent
			hasConflicts = conflicts
		}
	}
	if !merge {
		finalContent = string(newContent)
		processedContent, err := processIncludesInContent(finalContent, &WorkflowSpec{
			RepoSpec: RepoSpec{
				RepoSlug: repo,
				Version:  latestRef,
			},
			WorkflowPath: latestPath,
		}, latestRef, filepath.Dir(wf.Path), opts.Verbose)
		if err == nil {
			finalContent = processedContent
		}
	}

	finalContent, err = UpdateFieldInFrontmatter(finalContent, "source", manifestSource)
	if err != nil {
		return fmt.Errorf("failed to update source frontmatter: %w", err)
	}

	if opts.NoStopAfter {
		cleanedContent, err := RemoveFieldFromOnTrigger(finalContent, "stop-after")
		if err == nil {
			finalContent = cleanedContent
		}
	} else if opts.StopAfter != "" {
		updatedContent, err := SetFieldInOnTrigger(finalContent, "stop-after", opts.StopAfter)
		if err == nil {
			finalContent = updatedContent
		}
	}

	if !opts.DisableSecurityScanner {
		if findings := workflow.ScanMarkdownSecurity(finalContent); len(findings) > 0 {
			return fmt.Errorf("workflow '%s' failed security scan: %d issue(s) detected", wf.Name, len(findings))
		}
	}

	if err := os.WriteFile(wf.Path, []byte(finalContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}
	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", wf.Name, shortRef(currentRef), shortRef(latestRef))))
		return nil
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", wf.Name, shortRef(currentRef), shortRef(latestRef))))
	if !opts.NoCompile {
		if err := compileWorkflowWithRefresh(ctx, wf.Path, opts.Verbose, false, opts.EngineOverride, true); err != nil {
			return fmt.Errorf("failed to compile updated workflow: %w", err)
		}
	}
	return nil
}

func addManifestManagedWorkflow(ctx context.Context, targetDir, name, repo, latestPath, latestRef, manifestSource string, opts UpdateWorkflowsOptions) error {
	updateManifestLog.Printf("Adding new manifest-managed workflow %s from %s/%s@%s", name, repo, latestPath, latestRef)
	newContent, err := downloadWorkflowContentFn(ctx, repo, latestPath, latestRef, opts.Verbose)
	if err != nil {
		return fmt.Errorf("failed to download new manifest workflow %s/%s@%s: %w", repo, latestPath, latestRef, err)
	}

	content, err := UpdateFieldInFrontmatter(string(newContent), "source", manifestSource)
	if err != nil {
		return fmt.Errorf("failed to add source frontmatter for %s: %w", name, err)
	}
	if opts.NoStopAfter {
		cleanedContent, err := RemoveFieldFromOnTrigger(content, "stop-after")
		if err == nil {
			content = cleanedContent
		}
	} else if opts.StopAfter != "" {
		updatedContent, err := SetFieldInOnTrigger(content, "stop-after", opts.StopAfter)
		if err == nil {
			content = updatedContent
		}
	}
	if !opts.DisableSecurityScanner {
		if findings := workflow.ScanMarkdownSecurity(content); len(findings) > 0 {
			return fmt.Errorf("workflow '%s' failed security scan: %d issue(s) detected", name, len(findings))
		}
	}

	destPath := filepath.Join(targetDir, name+".md")
	if err := os.WriteFile(destPath, []byte(content), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write new manifest workflow %s: %w", destPath, err)
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Added new workflow from manifest: "+filepath.Base(destPath)))
	if !opts.NoCompile {
		if err := compileWorkflowWithRefresh(ctx, destPath, opts.Verbose, false, opts.EngineOverride, true); err != nil {
			return fmt.Errorf("failed to compile new manifest workflow: %w", err)
		}
	}
	return nil
}
