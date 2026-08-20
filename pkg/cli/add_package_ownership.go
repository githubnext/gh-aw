package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/workflow"
)

const packageOwnershipSchemaVersion = 1

type packageOwnershipRecord struct {
	SchemaVersion  int                         `json:"schemaVersion"`
	Package        string                      `json:"package"`
	Source         string                      `json:"source"`
	ResolvedCommit string                      `json:"resolvedCommit,omitempty"`
	Installer      string                      `json:"installer"`
	Files          []packageOwnershipFileEntry `json:"files"`
}

type packageOwnershipFileEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	SHA256      string `json:"sha256"`
}

func writePackageOwnershipRecords(workflows []*ResolvedWorkflow, tracker *FileTracker, opts AddOptions) error {
	groups := packageManagedWorkflowGroups(workflows)
	if len(groups) == 0 {
		return nil
	}
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root for package ownership records: %w", err)
	}
	for packageSource, group := range groups {
		record, err := buildPackageOwnershipRecord(gitRoot, packageSource, group, opts)
		if err != nil {
			return err
		}
		recordPath := packageOwnershipRecordPath(gitRoot, packageSource)
		if err := os.MkdirAll(filepath.Dir(recordPath), constants.DirPermPublic); err != nil {
			return fmt.Errorf("failed to create package ownership directory: %w", err)
		}
		existed := fileutil.FileExists(recordPath)
		if tracker != nil {
			if existed {
				tracker.TrackModified(recordPath)
			} else {
				tracker.TrackCreated(recordPath)
			}
		}
		data, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode package ownership record: %w", err)
		}
		data = append(data, '\n')
		if err := os.WriteFile(recordPath, data, constants.FilePermPublic); err != nil {
			return fmt.Errorf("failed to write package ownership record %s: %w", recordPath, err)
		}
	}
	return nil
}

func packageManagedWorkflowGroups(workflows []*ResolvedWorkflow) map[string][]*ResolvedWorkflow {
	groups := make(map[string][]*ResolvedWorkflow)
	for _, resolved := range workflows {
		if resolved == nil || resolved.Spec == nil || !resolved.Spec.FromRepositoryManifest {
			continue
		}
		source := packageSourceForSpec(resolved.Spec, resolved.SourceInfo)
		if source == "" {
			continue
		}
		groups[source] = append(groups[source], resolved)
	}
	return groups
}

func packageSourceForSpec(spec *WorkflowSpec, sourceInfo *FetchedWorkflow) string {
	if spec.RepoSlug == "" {
		return ""
	}
	ref := spec.Version
	if sourceInfo != nil && sourceInfo.CommitSHA != "" {
		ref = sourceInfo.CommitSHA
	}
	return manifestSourceWithRef(&RepoSpec{
		RepoSlug:    spec.RepoSlug,
		PackagePath: spec.PackagePath,
	}, ref)
}

func buildPackageOwnershipRecord(gitRoot, packageSource string, workflows []*ResolvedWorkflow, opts AddOptions) (*packageOwnershipRecord, error) {
	record := &packageOwnershipRecord{
		SchemaVersion:  packageOwnershipSchemaVersion,
		Package:        strings.Split(packageSource, "@")[0],
		Source:         packageSource,
		ResolvedCommit: packageSourceRef(packageSource),
		Installer:      "gh-aw " + GetVersion(),
	}
	for _, resolved := range workflows {
		destination, err := packageManagedDestination(resolved, opts)
		if err != nil {
			return nil, err
		}
		digest, err := fileSHA256(filepath.Join(gitRoot, filepath.FromSlash(destination)))
		if err != nil {
			return nil, err
		}
		record.Files = append(record.Files, packageOwnershipFileEntry{
			Source:      filepath.ToSlash(resolved.Spec.WorkflowPath),
			Destination: destination,
			SHA256:      digest,
		})
	}
	sort.Slice(record.Files, func(i, j int) bool {
		return record.Files[i].Destination < record.Files[j].Destination
	})
	return record, nil
}

func packageManagedDestination(resolved *ResolvedWorkflow, opts AddOptions) (string, error) {
	spec := resolved.Spec
	switch {
	case resolved.IsPackageResourceFile:
		return filepath.ToSlash(filepath.Clean(filepath.FromSlash(spec.DestinationPath))), nil
	case resolved.IsPackageSkillFile:
		relPath, err := resolveSkillRelativePath(resolved)
		if err != nil {
			return "", err
		}
		return filepath.ToSlash(filepath.Join(workflow.GetEngineSkillDir(opts.EngineOverride), resolved.SkillName, relPath)), nil
	case resolved.IsPackageAgentFile:
		return filepath.ToSlash(filepath.Join(workflow.GetEngineSubAgentDir(opts.EngineOverride), filepath.Base(spec.WorkflowPath))), nil
	case resolved.IsActionWorkflow:
		return filepath.ToSlash(filepath.Join(packageOwnershipWorkflowDir(opts), spec.WorkflowName+".yml")), nil
	default:
		return filepath.ToSlash(filepath.Join(packageOwnershipWorkflowDir(opts), spec.WorkflowName+".md")), nil
	}
}

func packageOwnershipWorkflowDir(opts AddOptions) string {
	if opts.WorkflowDir != "" {
		return filepath.Clean(opts.WorkflowDir)
	}
	return constants.GetWorkflowDir()
}

func packageSourceRef(source string) string {
	if _, ref, ok := strings.Cut(source, "@"); ok {
		return ref
	}
	return ""
}

func packageOwnershipRecordPath(gitRoot, packageSource string) string {
	return filepath.Join(gitRoot, ".github", "aw", "packages", stablePackageID(packageSource)+".json")
}

func stablePackageID(packageSource string) string {
	base := strings.Split(packageSource, "@")[0]
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "@", "-")
	slug := strings.Trim(replacer.Replace(strings.ToLower(base)), "-")
	sum := sha256.Sum256([]byte(base))
	return fmt.Sprintf("%s-%s", slug, hex.EncodeToString(sum[:])[:12])
}

func fileSHA256(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s for digest: %w", path, err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func packageOwnershipAllowsOverwrite(gitRoot, destination, packageSource string) (owned bool, drifted bool) {
	records, err := readPackageOwnershipRecords(gitRoot)
	if err != nil {
		return false, false
	}
	normalized := filepath.ToSlash(filepath.Clean(destination))
	packageID := strings.Split(packageSource, "@")[0]
	for _, record := range records {
		if record.Package != packageID {
			continue
		}
		for _, file := range record.Files {
			if !strings.EqualFold(filepath.ToSlash(filepath.Clean(file.Destination)), normalized) {
				continue
			}
			current, err := fileSHA256(filepath.Join(gitRoot, filepath.FromSlash(file.Destination)))
			if err != nil {
				return true, true
			}
			return true, current != file.SHA256
		}
	}
	return false, false
}

func readPackageOwnershipRecords(gitRoot string) ([]packageOwnershipRecord, error) {
	dir := filepath.Join(gitRoot, ".github", "aw", "packages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []packageOwnershipRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var record packageOwnershipRecord
		if err := json.Unmarshal(data, &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func syncManifestManagedResources(ctx context.Context, repoSpec *RepoSpec, pkg *resolvedRepositoryPackage, ref string, opts UpdateWorkflowsOptions) error {
	if pkg == nil || repoSpec == nil {
		return nil
	}
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find git root for package resources: %w", err)
	}
	owner, repo, err := splitRepositoryPackageSlug(repoSpec.RepoSlug)
	if err != nil {
		return err
	}
	packageBase := repositoryPackageIdentifier(repoSpec.RepoSlug, repoSpec.PackagePath)
	recordPath := packageOwnershipRecordPath(gitRoot, packageBase)
	record := packageOwnershipRecord{
		SchemaVersion:  packageOwnershipSchemaVersion,
		Package:        packageBase,
		Source:         manifestSourceWithRef(repoSpec, ref),
		ResolvedCommit: ref,
		Installer:      "gh-aw " + GetVersion(),
	}
	if existing, err := readPackageOwnershipRecord(recordPath); err == nil && existing != nil {
		record.Files = existing.Files
	}

	desired := make(map[string]resolvedPackageResource, len(pkg.ResourceFiles))
	for _, resource := range pkg.ResourceFiles {
		desired[filepath.ToSlash(filepath.Clean(resource.DestinationPath))] = resource
	}

	var kept []packageOwnershipFileEntry
	for _, entry := range record.Files {
		destination := filepath.ToSlash(filepath.Clean(entry.Destination))
		if _, stillDesired := desired[destination]; stillDesired || !isPackageResourceDestination(destination) {
			kept = append(kept, entry)
			continue
		}
		path := filepath.Join(gitRoot, filepath.FromSlash(destination))
		current, digestErr := fileSHA256(path)
		if digestErr == nil && current == entry.SHA256 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove stale package resource %s: %w", destination, err)
			}
			fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Removed stale package resource: "+destination))
			continue
		}
		kept = append(kept, entry)
	}
	record.Files = kept

	for _, resource := range pkg.ResourceFiles {
		destination := filepath.ToSlash(filepath.Clean(resource.DestinationPath))
		destPath := filepath.Join(gitRoot, filepath.FromSlash(destination))
		if fileutil.FileExists(destPath) && !opts.Force {
			if owned, drifted := packageOwnershipAllowsOverwrite(gitRoot, destination, packageBase); !owned || drifted {
				if owned {
					return fmt.Errorf("resource %q has local modifications; use --force to overwrite", destination)
				}
				return fmt.Errorf("resource %q already exists; use --force to overwrite", destination)
			}
		}
		content, err := downloadPackageFileFromGitHubForHost(ctx, owner, repo, resource.SourcePath, ref, "")
		if err != nil {
			return fmt.Errorf("failed to download package resource %s: %w", resource.SourcePath, err)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), constants.DirPermPublic); err != nil {
			return fmt.Errorf("failed to create package resource directory: %w", err)
		}
		if err := os.WriteFile(destPath, content, constants.FilePermPublic); err != nil {
			return fmt.Errorf("failed to write package resource %s: %w", destination, err)
		}
		digest, err := fileSHA256(destPath)
		if err != nil {
			return err
		}
		record.Files = upsertPackageOwnershipFile(record.Files, packageOwnershipFileEntry{
			Source:      resource.SourcePath,
			Destination: destination,
			SHA256:      digest,
		})
	}
	if len(record.Files) == 0 {
		return nil
	}
	sort.Slice(record.Files, func(i, j int) bool {
		return record.Files[i].Destination < record.Files[j].Destination
	})
	if err := os.MkdirAll(filepath.Dir(recordPath), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create package ownership directory: %w", err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode package ownership record: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(recordPath, data, constants.FilePermPublic); err != nil {
		return fmt.Errorf("failed to write package ownership record: %w", err)
	}
	return nil
}

func readPackageOwnershipRecord(path string) (*packageOwnershipRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record packageOwnershipRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func upsertPackageOwnershipFile(entries []packageOwnershipFileEntry, next packageOwnershipFileEntry) []packageOwnershipFileEntry {
	for i := range entries {
		if strings.EqualFold(filepath.ToSlash(filepath.Clean(entries[i].Destination)), filepath.ToSlash(filepath.Clean(next.Destination))) {
			entries[i] = next
			return entries
		}
	}
	return append(entries, next)
}

func isPackageResourceDestination(destination string) bool {
	return strings.EqualFold(destination, constants.GithubDir+"CODEOWNERS") ||
		strings.HasPrefix(destination, constants.GithubDir+"ISSUE_TEMPLATE/") ||
		strings.HasPrefix(destination, constants.GithubDir+"aw/")
}

func removePackageOwnedFilesIfUnused(packageBase string) error {
	gitRoot, err := gitutil.FindGitRoot()
	if err != nil {
		return err
	}
	if packageBase == "" || packageHasRemainingWorkflows(gitRoot, packageBase) {
		return nil
	}
	recordPath := packageOwnershipRecordPath(gitRoot, packageBase)
	record, err := readPackageOwnershipRecord(recordPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var kept []packageOwnershipFileEntry
	for _, entry := range record.Files {
		destination := filepath.ToSlash(filepath.Clean(entry.Destination))
		if strings.HasPrefix(destination, constants.WorkflowsDirSlash) {
			continue
		}
		path := filepath.Join(gitRoot, filepath.FromSlash(destination))
		current, digestErr := fileSHA256(path)
		if digestErr == nil && current == entry.SHA256 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				kept = append(kept, entry)
				continue
			}
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Removed package-owned file: "+destination))
			continue
		}
		kept = append(kept, entry)
	}
	if len(kept) == 0 {
		if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	record.Files = kept
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(recordPath, data, constants.FilePermPublic)
}

func packageHasRemainingWorkflows(gitRoot, packageBase string) bool {
	pattern := filepath.Join(gitRoot, constants.GetWorkflowDir(), "*.md")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	for _, file := range files {
		source := readFullSourceFromFile(file)
		repoSpec, ok, err := parseManifestSourceSpec(source)
		if err != nil || !ok || repoSpec == nil {
			continue
		}
		if repositoryPackageIdentifier(repoSpec.RepoSlug, repoSpec.PackagePath) == packageBase {
			return true
		}
	}
	return false
}
