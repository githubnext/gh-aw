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

type manifestManagedWorkflowUpdate struct {
	wf             *workflowWithSource
	repo           string
	currentPath    string
	latestPath     string
	currentRef     string
	latestRef      string
	manifestSource string
}

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

	groupInfo, ok, failures := resolveUpdateManifestWorkflowGroupInfo(ctx, source, grouped, opts)
	if !ok {
		return successes, failures
	}

	existingByName := make(map[string]*workflowWithSource, len(grouped))
	for _, wf := range grouped {
		existingByName[wf.Name] = wf
	}

	successes, failures = updateManifestWorkflowGroupExisting(ctx, groupInfo, existingByName, opts, successes, failures)
	targetDir := filepath.Dir(grouped[0].Path)
	successes, failures = updateManifestWorkflowGroupAdditions(ctx, groupInfo, existingByName, targetDir, opts, successes, failures)
	return successes, failures
}

type updateManifestWorkflowGroupInfo struct {
	repoSpec       *RepoSpec
	currentRef     string
	latestRef      string
	manifestSource string
	currentByName  map[string]string
	latestByName   map[string]string
}

func resolveUpdateManifestWorkflowGroupInfo(ctx context.Context, source string, grouped []*workflowWithSource, opts UpdateWorkflowsOptions) (updateManifestWorkflowGroupInfo, bool, []updateFailure) {
	repoSpec, _, err := parseManifestSourceSpec(source)
	if err != nil {
		return updateManifestWorkflowGroupInfo{}, false, updateManifestWorkflowGroupFailures(grouped, err.Error())
	}
	if repoSpec == nil {
		return updateManifestWorkflowGroupInfo{}, false, nil
	}
	currentRef := repoSpec.Version
	if currentRef == "" {
		currentRef = "main"
	}
	latestRef, err := resolveLatestRefFn(ctx, repoSpec.RepoSlug, currentRef, opts.AllowMajor, opts.Verbose, opts.CoolDown)
	if err != nil {
		updateManifestLog.Printf("Failed to resolve latest manifest ref for %s: %v", repoSpec.RepoSlug, err)
		return updateManifestWorkflowGroupInfo{}, false, updateManifestWorkflowGroupFailures(grouped, fmt.Sprintf("failed to resolve latest manifest ref: %v", err))
	}
	currentPkg, latestPkg, failures := updateManifestWorkflowGroupPackages(ctx, repoSpec, currentRef, latestRef, grouped)
	if len(failures) > 0 {
		return updateManifestWorkflowGroupInfo{}, false, failures
	}
	sourceFieldRef := latestRef
	if isBranchRef(currentRef) {
		sourceFieldRef = currentRef
	}
	return updateManifestWorkflowGroupInfo{
		repoSpec:       repoSpec,
		currentRef:     currentRef,
		latestRef:      latestRef,
		manifestSource: manifestSourceWithRef(repoSpec, sourceFieldRef),
		currentByName:  manifestWorkflowPathByName(currentPkg.InstallationSource),
		latestByName:   manifestWorkflowPathByName(latestPkg.InstallationSource),
	}, true, nil
}

func updateManifestWorkflowGroupPackages(ctx context.Context, repoSpec *RepoSpec, currentRef, latestRef string, grouped []*workflowWithSource) (*resolvedRepositoryPackage, *resolvedRepositoryPackage, []updateFailure) {
	updateManifestLog.Printf("Resolved manifest refs: current=%s, latest=%s", currentRef, latestRef)
	currentPkg, err := resolveRepositoryPackage(ctx, &RepoSpec{RepoSlug: repoSpec.RepoSlug, PackagePath: repoSpec.PackagePath, Version: currentRef}, "")
	if err != nil {
		return nil, nil, updateManifestWorkflowGroupFailures(grouped, fmt.Sprintf("failed to resolve current manifest package: %v", err))
	}
	latestPkg, err := resolveRepositoryPackage(ctx, &RepoSpec{RepoSlug: repoSpec.RepoSlug, PackagePath: repoSpec.PackagePath, Version: latestRef}, "")
	if err != nil {
		return nil, nil, updateManifestWorkflowGroupFailures(grouped, fmt.Sprintf("failed to resolve latest manifest package: %v", err))
	}
	return currentPkg, latestPkg, nil
}

func updateManifestWorkflowGroupFailures(grouped []*workflowWithSource, msg string) []updateFailure {
	var failures []updateFailure
	for _, wf := range grouped {
		failures = append(failures, updateFailure{Name: wf.Name, Error: msg})
	}
	return failures
}

func updateManifestWorkflowGroupExisting(ctx context.Context, info updateManifestWorkflowGroupInfo, existingByName map[string]*workflowWithSource, opts UpdateWorkflowsOptions, successes []string, failures []updateFailure) ([]string, []updateFailure) {
	for name, wf := range existingByName {
		latestPath, exists := info.latestByName[name]
		if !exists {
			if err := removeManifestManagedWorkflow(wf.Path); err != nil {
				failures = append(failures, updateFailure{Name: wf.Name, Error: err.Error()})
				continue
			}
			successes = append(successes, wf.Name)
			continue
		}
		update := updateManifestWorkflowGroupUpdate(info, wf, name, latestPath)
		if err := updateManifestManagedWorkflow(ctx, update, opts); err != nil {
			failures = append(failures, updateFailure{Name: wf.Name, Error: err.Error()})
			continue
		}
		successes = append(successes, wf.Name)
	}
	return successes, failures
}

func updateManifestWorkflowGroupUpdate(info updateManifestWorkflowGroupInfo, wf *workflowWithSource, name, latestPath string) manifestManagedWorkflowUpdate {
	oldPath := info.currentByName[name]
	if oldPath == "" {
		oldPath = latestPath
	}
	return manifestManagedWorkflowUpdate{
		wf:             wf,
		repo:           info.repoSpec.RepoSlug,
		currentPath:    oldPath,
		latestPath:     latestPath,
		currentRef:     info.currentRef,
		latestRef:      info.latestRef,
		manifestSource: info.manifestSource,
	}
}

func updateManifestWorkflowGroupAdditions(ctx context.Context, info updateManifestWorkflowGroupInfo, existingByName map[string]*workflowWithSource, targetDir string, opts UpdateWorkflowsOptions, successes []string, failures []updateFailure) ([]string, []updateFailure) {
	for name, latestPath := range info.latestByName {
		if _, exists := existingByName[name]; exists {
			continue
		}
		if err := addManifestManagedWorkflow(ctx, targetDir, name, info.repoSpec.RepoSlug, latestPath, info.latestRef, info.manifestSource, opts); err != nil {
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

func updateManifestManagedWorkflow(ctx context.Context, update manifestManagedWorkflowUpdate, opts UpdateWorkflowsOptions) error {
	updateManifestLog.Printf("Updating manifest-managed workflow %s: %s@%s -> %s@%s", update.wf.Name, update.currentPath, update.currentRef, update.latestPath, update.latestRef)
	sourceSpecCurrent := sourceSpecWithRef(&SourceSpec{Repo: update.repo, Path: update.currentPath}, update.currentRef)
	newContent, err := downloadWorkflowContentFn(ctx, update.repo, update.latestPath, update.latestRef, opts.Verbose)
	if err != nil {
		return fmt.Errorf("failed to download workflow %s/%s@%s: %w", update.repo, update.latestPath, update.latestRef, err)
	}

	upToDate, err := updateManifestManagedWorkflowAlreadyCurrent(ctx, update, sourceSpecCurrent, opts)
	if err != nil || upToDate {
		return err
	}

	finalContent, hasConflicts, err := updateManifestManagedWorkflowContent(ctx, update, sourceSpecCurrent, string(newContent), opts)
	if err != nil {
		return err
	}
	finalContent, err = updateManifestManagedWorkflowFinalizeContent(update, finalContent, opts)
	if err != nil {
		return err
	}

	if err := os.WriteFile(update.wf.Path, []byte(finalContent), constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write updated workflow: %w", err)
	}
	if hasConflicts {
		fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Updated %s from %s to %s with CONFLICTS - please review and resolve manually", update.wf.Name, shortRef(update.currentRef), shortRef(update.latestRef))))
		return nil
	}
	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Updated %s from %s to %s", update.wf.Name, shortRef(update.currentRef), shortRef(update.latestRef))))
	if !opts.NoCompile {
		if err := compileWorkflowWithRefresh(ctx, update.wf.Path, opts.Verbose, false, opts.EngineOverride, true, opts.Approve); err != nil {
			return fmt.Errorf("failed to compile updated workflow: %w", err)
		}
	}
	return nil
}

func updateManifestManagedWorkflowAlreadyCurrent(ctx context.Context, update manifestManagedWorkflowUpdate, sourceSpecCurrent string, opts UpdateWorkflowsOptions) (bool, error) {
	if opts.Force || update.currentRef != update.latestRef || update.currentPath != update.latestPath {
		return false, nil
	}
	sourceContent, err := downloadWorkflowContentFn(ctx, update.repo, update.currentPath, update.currentRef, opts.Verbose)
	if err != nil {
		return false, nil
	}
	currentContent, readErr := os.ReadFile(update.wf.Path)
	if readErr == nil && !hasLocalModifications(string(sourceContent), string(currentContent), sourceSpecCurrent, filepath.Dir(update.wf.Path), opts.Verbose) {
		fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Workflow %s is already up to date (%s)", update.wf.Name, shortRef(update.currentRef))))
		return true, nil
	}
	return false, nil
}

func updateManifestManagedWorkflowContent(ctx context.Context, update manifestManagedWorkflowUpdate, sourceSpecCurrent string, newContent string, opts UpdateWorkflowsOptions) (string, bool, error) {
	if opts.NoMerge {
		return updateManifestManagedWorkflowOverwrite(update, newContent, opts)
	}
	baseContent, err := downloadWorkflowContentFn(ctx, update.repo, update.currentPath, update.currentRef, opts.Verbose)
	if err != nil {
		updateManifestLog.Printf("Cannot fetch base for 3-way merge of %s, falling back to overwrite: %v", update.wf.Name, err)
		return updateManifestManagedWorkflowOverwrite(update, newContent, opts)
	}
	currentContent, err := os.ReadFile(update.wf.Path)
	if err != nil {
		return "", false, fmt.Errorf("failed to read current workflow: %w", err)
	}
	newSourceSpec := sourceSpecWithRef(&SourceSpec{Repo: update.repo, Path: update.latestPath}, update.latestRef)
	mergedContent, conflicts, mergeErr := MergeWorkflowContent(string(baseContent), string(currentContent), newContent, sourceSpecCurrent, newSourceSpec, update.wf.Path, opts.Verbose)
	if mergeErr != nil {
		return "", false, fmt.Errorf("failed to merge workflow content: %w", mergeErr)
	}
	return mergedContent, conflicts, nil
}

func updateManifestManagedWorkflowOverwrite(update manifestManagedWorkflowUpdate, newContent string, opts UpdateWorkflowsOptions) (string, bool, error) {
	finalContent := newContent
	processedContent, err := processIncludesInContent(finalContent, &WorkflowSpec{
		RepoSpec: RepoSpec{
			RepoSlug: update.repo,
			Version:  update.latestRef,
		},
		WorkflowPath: update.latestPath,
	}, update.latestRef, filepath.Dir(update.wf.Path), opts.Verbose)
	if err == nil {
		finalContent = processedContent
	}
	return finalContent, false, nil
}

func updateManifestManagedWorkflowFinalizeContent(update manifestManagedWorkflowUpdate, finalContent string, opts UpdateWorkflowsOptions) (string, error) {
	finalContent, err := UpdateFieldInFrontmatter(finalContent, "source", update.manifestSource)
	if err != nil {
		return "", fmt.Errorf("failed to update source frontmatter: %w", err)
	}
	finalContent = updateManifestManagedWorkflowStopAfter(finalContent, opts)
	if !opts.DisableSecurityScanner {
		if findings := workflow.ScanMarkdownSecurity(finalContent); len(findings) > 0 {
			return "", fmt.Errorf("workflow '%s' failed security scan: %d issue(s) detected", update.wf.Name, len(findings))
		}
	}
	return finalContent, nil
}

func updateManifestManagedWorkflowStopAfter(finalContent string, opts UpdateWorkflowsOptions) string {
	if opts.NoStopAfter {
		cleanedContent, err := RemoveFieldFromOnTrigger(finalContent, "stop-after")
		if err == nil {
			return cleanedContent
		}
	} else if opts.StopAfter != "" {
		updatedContent, err := SetFieldInOnTrigger(finalContent, "stop-after", opts.StopAfter)
		if err == nil {
			return updatedContent
		}
	}
	return finalContent
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
		if err := compileWorkflowWithRefresh(ctx, destPath, opts.Verbose, false, opts.EngineOverride, true, opts.Approve); err != nil {
			return fmt.Errorf("failed to compile new manifest workflow: %w", err)
		}
	}
	return nil
}
