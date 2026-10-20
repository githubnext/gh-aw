package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/rhysd/actionlint"
	"gopkg.in/yaml.v3"
)

const nativeActionsLockfilePath = ".github/workflows/actions.lock"

var fullCommitSHA = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

type nativeActionsLockfile struct {
	Version      string                      `yaml:"version"`
	Workflows    map[string][]string         `yaml:"workflows"`
	Dependencies map[string]nativeDependency `yaml:"dependencies"`
}

type nativeDependency struct {
	Hostname string   `yaml:"hostname,omitempty"`
	Ref      string   `yaml:"ref"`
	Commit   string   `yaml:"commit"`
	OwnerID  int64    `yaml:"owner_id"`
	RepoID   int64    `yaml:"repo_id"`
	Uses     []string `yaml:"uses,omitempty"`
}

type nativeDependencyUse struct {
	repository     string
	ref            string
	actionPath     string
	selfRepository bool
}

type nativeDependencyResolution struct {
	hostname string
	ref      string
	commit   string
	ownerID  int64
	repoID   int64
	uses     []nativeDependencyUse
}

type nativeWorkflowInput struct {
	path     string
	uses     []nativeDependencyUse
	refHints map[string]string
}

func unmarshalNativeActionsLockfile(content []byte, lockfile *nativeActionsLockfile) error {
	var header struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(content, &header); err != nil {
		return err
	}
	if header.Version != "v0.0.2" && header.Version != "v0.0.3" {
		return fmt.Errorf("unsupported GitHub Actions lockfile version %q", header.Version)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(lockfile); err != nil {
		return err
	}
	if lockfile.Workflows == nil || lockfile.Dependencies == nil {
		return errors.New("lockfile must define workflows and dependencies")
	}
	for workflowPath, pins := range lockfile.Workflows {
		if workflowPath == "" {
			return errors.New("lockfile workflow path must not be empty")
		}
		for _, pin := range pins {
			if !validNativePinKey(pin) {
				return fmt.Errorf("invalid dependency key %q in workflow %s", pin, workflowPath)
			}
		}
	}
	for key, dependency := range lockfile.Dependencies {
		if !validNativePinKey(key) || !validNativeRef(dependency.Ref) ||
			!strings.HasPrefix(dependency.Commit, "sha1-") ||
			!fullCommitSHA.MatchString(strings.TrimPrefix(dependency.Commit, "sha1-")) ||
			dependency.OwnerID <= 0 || dependency.RepoID <= 0 {
			return fmt.Errorf("invalid dependency entry %q in lockfile", key)
		}
		for _, child := range dependency.Uses {
			if !validNativePinKey(child) {
				return fmt.Errorf("invalid dependency key %q in dependency %s", child, key)
			}
		}
	}
	return nil
}

func canonicalizeNativeActionsLockfile(lockfile *nativeActionsLockfile) error {
	dependencies := make(map[string]nativeDependency, len(lockfile.Dependencies))
	for key, dependency := range lockfile.Dependencies {
		canonicalKey, ok := canonicalNativePinKey(key)
		if !ok {
			return fmt.Errorf("invalid dependency key %q", key)
		}
		for i, child := range dependency.Uses {
			canonicalChild, ok := canonicalNativePinKey(child)
			if !ok {
				return fmt.Errorf("invalid child dependency key %q in %s", child, key)
			}
			dependency.Uses[i] = canonicalChild
		}
		slices.Sort(dependency.Uses)
		dependency.Uses = slices.Compact(dependency.Uses)
		if previous, found := dependencies[canonicalKey]; found && !sameNativeDependency(previous, dependency) {
			return fmt.Errorf("conflicting dependency entries normalize to %s", canonicalKey)
		}
		dependencies[canonicalKey] = dependency
	}
	lockfile.Dependencies = dependencies
	for workflowPath, pins := range lockfile.Workflows {
		canonicalPins := make([]string, 0, len(pins))
		seen := make(map[string]struct{}, len(pins))
		for _, pin := range pins {
			canonicalPin, ok := canonicalNativePinKey(pin)
			if !ok {
				return fmt.Errorf("invalid dependency key %q in workflow %s", pin, workflowPath)
			}
			if _, found := seen[canonicalPin]; !found {
				seen[canonicalPin] = struct{}{}
				canonicalPins = append(canonicalPins, canonicalPin)
			}
		}
		slices.Sort(canonicalPins)
		lockfile.Workflows[workflowPath] = canonicalPins
	}
	return nil
}

func sameNativeDependency(a, b nativeDependency) bool {
	return a.Hostname == b.Hostname && a.Ref == b.Ref && a.Commit == b.Commit &&
		a.OwnerID == b.OwnerID && a.RepoID == b.RepoID && slices.Equal(a.Uses, b.Uses)
}

func canonicalNativePinKey(key string) (string, bool) {
	use, ok := parseNativeDependencyUse(key)
	if !ok || use.selfRepository || use.actionPath != "" {
		return "", false
	}
	return nativeDependencyKey(use), true
}

func validNativePinKey(key string) bool {
	_, ok := canonicalNativePinKey(key)
	return ok
}

type nativeDependencyResolver func(context.Context, nativeDependencyUse, string, *nativeDependency) (nativeDependencyResolution, error)

func generateNativeActionsLockfile(ctx context.Context, repoRoot string, compiledLockFiles []string, actionResolver *workflow.ActionResolver) error {
	repositoryIDs := make(map[string]nativeRepositoryIDs)
	resolutions := make(map[string]nativeDependencyResolution)
	resolver := func(ctx context.Context, use nativeDependencyUse, refHint string, existing *nativeDependency) (nativeDependencyResolution, error) {
		existingCommit := ""
		if existing != nil {
			existingCommit = existing.Commit
		}
		cacheKey := strings.Join([]string{nativeDependencyVisitKey(use), refHint, existingCommit}, "\x00")
		if existing != nil {
			cacheKey = strings.Join([]string{nativeDependencyVisitKey(use), existingCommit}, "\x00")
		}
		if resolution, ok := resolutions[cacheKey]; ok {
			return resolution, nil
		}
		resolution, err := resolveNativeDependency(ctx, use, refHint, actionResolver, repositoryIDs, existing)
		if err == nil {
			resolutions[cacheKey] = resolution
			resolutions[strings.Join([]string{nativeDependencyVisitKey(use), resolution.commit}, "\x00")] = resolution
		}
		return resolution, err
	}
	return generateNativeActionsLockfileWithResolver(ctx, repoRoot, compiledLockFiles, resolver)
}

func generateNativeActionsLockfileWithResolver(
	ctx context.Context,
	repoRoot string,
	compiledLockFiles []string,
	resolve nativeDependencyResolver,
) error {
	if len(compiledLockFiles) == 0 {
		return nil
	}

	inputs, err := readNativeWorkflowInputs(repoRoot, compiledLockFiles)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return nil
	}

	lockPath := filepath.Join(repoRoot, filepath.FromSlash(nativeActionsLockfilePath))
	lockfile, err := readNativeActionsLockfile(lockPath)
	if err != nil {
		return err
	}

	generatedDependencies := make(map[string]nativeDependency, len(lockfile.Dependencies))
	maps.Copy(generatedDependencies, lockfile.Dependencies)
	for _, input := range inputs {
		pins := make(map[string]struct{})
		visited := make(map[string]struct{})
		for _, use := range input.uses {
			if _, err := collectNativeDependency(ctx, use, input.refHints, resolve, generatedDependencies, pins, visited); err != nil {
				return fmt.Errorf("failed to resolve dependencies for %s: %w", input.path, err)
			}
		}
		lockfile.Workflows[input.path] = sortedNativeKeys(pins)
	}

	maps.Copy(lockfile.Dependencies, generatedDependencies)
	lockfile.Version = "v0.0.3"
	normalizeNativeWorkflowPins(&lockfile)

	output, err := yaml.Marshal(lockfile)
	if err != nil {
		return fmt.Errorf("failed to serialize GitHub Actions lockfile: %w", err)
	}
	output = append([]byte("# Generated by `gh aw compile`; do not edit by hand.\n# Docs: https://gh.io/actions-lockfile\n"), output...)
	if err := writeNativeActionsLockfile(lockPath, output); err != nil {
		return err
	}
	return nil
}

func readNativeActionsLockfile(lockPath string) (nativeActionsLockfile, error) {
	lockfile := nativeActionsLockfile{
		Version:      "v0.0.3",
		Workflows:    make(map[string][]string),
		Dependencies: make(map[string]nativeDependency),
	}
	content, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return lockfile, nil
		}
		return nativeActionsLockfile{}, fmt.Errorf("failed to read existing GitHub Actions lockfile: %w", err)
	}
	if err := unmarshalNativeActionsLockfile(content, &lockfile); err != nil {
		return nativeActionsLockfile{}, fmt.Errorf("failed to parse existing GitHub Actions lockfile: %w", err)
	}
	if err := canonicalizeNativeActionsLockfile(&lockfile); err != nil {
		return nativeActionsLockfile{}, fmt.Errorf("failed to validate existing GitHub Actions lockfile: %w", err)
	}
	if lockfile.Version == "v0.0.2" {
		for key, dependency := range lockfile.Dependencies {
			if dependency.Hostname == "" {
				dependency.Hostname = "github.com"
				lockfile.Dependencies[key] = dependency
			}
		}
	}
	return lockfile, nil
}

func readNativeWorkflowInputs(repoRoot string, compiledLockFiles []string) ([]nativeWorkflowInput, error) {
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repository root: %w", err)
	}
	inputs := make([]nativeWorkflowInput, 0, len(compiledLockFiles))
	for _, lockFile := range compiledLockFiles {
		absoluteLockFile, err := filepath.Abs(lockFile)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve compiled workflow path %s: %w", lockFile, err)
		}
		relative, err := filepath.Rel(absoluteRoot, absoluteLockFile)
		if err != nil {
			return nil, fmt.Errorf("failed to locate compiled workflow %s: %w", lockFile, err)
		}
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, logsWorkflowsPathSegment) || !strings.HasSuffix(relative, ".lock.yml") {
			continue
		}
		content, err := os.ReadFile(lockFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read compiled workflow %s: %w", lockFile, err)
		}
		uses, localWorkflowPaths, err := parseNativeWorkflowUsesAndLocalCalls(content, "", "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse compiled workflow %s: %w", lockFile, err)
		}
		refHints := make(map[string]string)
		if err := collectLocalNativeWorkflowUses(repoRoot, localWorkflowPaths, make(map[string]struct{}), &uses, refHints); err != nil {
			return nil, fmt.Errorf("failed to collect local reusable workflows for %s: %w", lockFile, err)
		}
		manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to read gh-aw manifest from %s: %w", lockFile, err)
		}
		if manifest != nil {
			for _, action := range manifest.Actions {
				if use, ok := parseNativeDependencyUse(action.Repo + "@" + action.SHA); ok {
					refHints[nativeDependencyKey(use)] = action.Version
				}
			}
		}
		inputs = append(inputs, nativeWorkflowInput{
			path:     relative,
			uses:     uses,
			refHints: refHints,
		})
	}
	slices.SortFunc(inputs, func(a, b nativeWorkflowInput) int {
		return strings.Compare(a.path, b.path)
	})
	return inputs, nil
}

func collectNativeLockFilesForDirectory(workflowsDir string) ([]string, error) {
	var lockFiles []string
	err := filepath.WalkDir(workflowsDir, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(filePath, ".lock.yml") || strings.HasSuffix(filePath, ".campaign.lock.yml") {
			return nil
		}
		lockFiles = append(lockFiles, filePath)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to discover compiled workflows in %s: %w", workflowsDir, err)
	}
	slices.Sort(lockFiles)
	return lockFiles, nil
}

func parseNativeWorkflowUses(content []byte) ([]nativeDependencyUse, error) {
	return parseNativeWorkflowUsesWithContext(content, "", "")
}

func parseNativeWorkflowUsesWithContext(content []byte, repository, commit string) ([]nativeDependencyUse, error) {
	uses, _, err := parseNativeWorkflowUsesAndLocalCalls(content, repository, commit)
	return uses, err
}

func parseNativeWorkflowUsesAndLocalCalls(content []byte, repository, commit string) ([]nativeDependencyUse, []string, error) {
	ast, parseErrors := actionlint.Parse(content)
	if ast == nil {
		if len(parseErrors) > 0 {
			return nil, nil, fmt.Errorf("failed to parse GitHub Actions workflow: %w", parseErrors[0])
		}
		return nil, nil, errors.New("workflow must be a YAML mapping")
	}
	values, selfUses, localWorkflowPaths, err := collectNativeWorkflowReferences(ast, repository, commit)
	if err != nil {
		return nil, nil, err
	}
	uses, err := normalizeNativeUses(values)
	if err != nil {
		return nil, nil, err
	}
	if repository == "" || commit == "" {
		filtered := uses[:0]
		for _, use := range uses {
			if !use.selfRepository {
				filtered = append(filtered, use)
			}
		}
		uses = filtered
	}
	for i := range uses {
		if uses[i].selfRepository {
			uses[i].repository = repository
			uses[i].ref = commit
		}
	}
	uses = append(uses, selfUses...)
	uses = sortUniqueNativeUses(uses)
	slices.Sort(localWorkflowPaths)
	localWorkflowPaths = slices.Compact(localWorkflowPaths)
	return uses, localWorkflowPaths, nil
}

func collectNativeWorkflowReferences(ast *actionlint.Workflow, repository, commit string) ([]string, []nativeDependencyUse, []string, error) {
	var values []string
	var selfUses []nativeDependencyUse
	var localWorkflowPaths []string
	for _, job := range ast.Jobs {
		if job.WorkflowCall != nil && job.WorkflowCall.Uses != nil {
			value := strings.TrimSpace(job.WorkflowCall.Uses.Value)
			if strings.HasPrefix(value, "./") {
				workflowPath, ok := nativeRelativeWorkflowPath(value)
				if !ok {
					return nil, nil, nil, fmt.Errorf("unsupported relative reusable workflow reference %q", value)
				}
				if repository != "" && commit != "" {
					selfUses = append(selfUses, nativeDependencyUse{
						repository:     repository,
						ref:            commit,
						actionPath:     workflowPath,
						selfRepository: true,
					})
				} else {
					localWorkflowPaths = append(localWorkflowPaths, workflowPath)
				}
			} else {
				values = append(values, value)
			}
		}
		for _, step := range job.Steps {
			if action, ok := step.Exec.(*actionlint.ExecAction); ok && action.Uses != nil {
				value := strings.TrimSpace(action.Uses.Value)
				if strings.HasPrefix(value, "./") {
					return nil, nil, nil, fmt.Errorf("local action reference %q is unsupported in native lockfiles", value)
				}
				values = append(values, value)
			}
		}
	}
	return values, selfUses, localWorkflowPaths, nil
}

func collectLocalNativeWorkflowUses(
	repoRoot string,
	paths []string,
	visited map[string]struct{},
	uses *[]nativeDependencyUse,
	refHints map[string]string,
) error {
	for _, workflowPath := range paths {
		if _, found := visited[workflowPath]; found {
			continue
		}
		visited[workflowPath] = struct{}{}
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(workflowPath)))
		if err != nil {
			return fmt.Errorf("failed to read local reusable workflow %s: %w", workflowPath, err)
		}
		children, childPaths, err := parseNativeWorkflowUsesAndLocalCalls(content, "", "")
		if err != nil {
			return fmt.Errorf("failed to parse local reusable workflow %s: %w", workflowPath, err)
		}
		*uses = append(*uses, children...)
		manifest, err := workflow.ExtractGHAWManifestFromLockFile(string(content))
		if err != nil {
			return fmt.Errorf("failed to read gh-aw manifest from local reusable workflow %s: %w", workflowPath, err)
		}
		if manifest != nil {
			for _, action := range manifest.Actions {
				if use, ok := parseNativeDependencyUse(action.Repo + "@" + action.SHA); ok {
					refHints[nativeDependencyKey(use)] = action.Version
				}
			}
		}
		if err := collectLocalNativeWorkflowUses(repoRoot, childPaths, visited, uses, refHints); err != nil {
			return err
		}
	}
	return nil
}

func parseNativeActionUses(content []byte) ([]nativeDependencyUse, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	root := unwrapNativeYAMLNode(&document)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil, errors.New("action metadata must be a YAML mapping")
	}
	runs := unwrapNativeYAMLNode(nativeYAMLMapValue(root, "runs"))
	if runs == nil || runs.Kind != yaml.MappingNode {
		return nil, nil
	}
	steps := unwrapNativeYAMLNode(nativeYAMLMapValue(runs, "steps"))
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return nil, nil
	}
	var values []string
	for _, step := range steps.Content {
		step = unwrapNativeYAMLNode(step)
		if step == nil || step.Kind != yaml.MappingNode {
			continue
		}
		if uses := unwrapNativeYAMLNode(nativeYAMLMapValue(step, "uses")); uses != nil && uses.Kind == yaml.ScalarNode {
			values = append(values, uses.Value)
		}
	}
	uses, err := normalizeNativeUses(values)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "./") {
			return nil, fmt.Errorf("local action reference %q is unsupported in native lockfiles", value)
		}
	}
	return uses, nil
}

func nativeYAMLMapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	matchedKey := false
	for i, child := range node.Content {
		if i%2 == 0 {
			matchedKey = child.Value == key
		} else if matchedKey {
			return child
		}
	}
	return nil
}

func unwrapNativeYAMLNode(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return unwrapNativeYAMLNode(node.Content[0])
	}
	return node
}

func normalizeNativeUses(values []string) ([]nativeDependencyUse, error) {
	seen := make(map[string]struct{})
	uses := make([]nativeDependencyUse, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.HasPrefix(value, "docker://") {
			continue
		}
		use, ok := parseNativeDependencyUse(value)
		if !ok {
			return nil, fmt.Errorf("unsupported GitHub dependency reference %q", value)
		}
		key := nativeDependencyVisitKey(use)
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		uses = append(uses, use)
	}
	return sortUniqueNativeUses(uses), nil
}

func sortUniqueNativeUses(uses []nativeDependencyUse) []nativeDependencyUse {
	slices.SortFunc(uses, func(a, b nativeDependencyUse) int {
		return strings.Compare(nativeDependencyVisitKey(a), nativeDependencyVisitKey(b))
	})
	return slices.CompactFunc(uses, func(a, b nativeDependencyUse) bool {
		return nativeDependencyVisitKey(a) == nativeDependencyVisitKey(b)
	})
}

func parseNativeDependencyUse(value string) (nativeDependencyUse, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "./") || strings.HasPrefix(value, "docker://") {
		return nativeDependencyUse{}, false
	}
	if rawPath, selfRepository := strings.CutPrefix(value, "$/"); selfRepository {
		for part := range strings.SplitSeq(rawPath, "/") {
			if !validNativeRepositoryPart(part) {
				return nativeDependencyUse{}, false
			}
		}
		actionPath := path.Clean(rawPath)
		if actionPath == "" || strings.HasPrefix(actionPath, "../") || actionPath == ".." {
			return nativeDependencyUse{}, false
		}
		for part := range strings.SplitSeq(actionPath, "/") {
			if !validNativeRepositoryPart(part) {
				return nativeDependencyUse{}, false
			}
		}
		return nativeDependencyUse{actionPath: actionPath, selfRepository: true}, true
	}
	at := strings.IndexByte(value, '@')
	if at <= 0 || at == len(value)-1 {
		return nativeDependencyUse{}, false
	}
	owner, repositoryAndPath, found := strings.Cut(value[:at], "/")
	if !found {
		return nativeDependencyUse{}, false
	}
	repository, actionPath, hasActionPath := strings.Cut(repositoryAndPath, "/")
	if !validNativeRepositoryPart(owner) || !validNativeRepositoryPart(repository) {
		return nativeDependencyUse{}, false
	}
	if hasActionPath {
		for part := range strings.SplitSeq(actionPath, "/") {
			if !validNativeRepositoryPart(part) {
				return nativeDependencyUse{}, false
			}
		}
	}
	ref := value[at+1:]
	if !validNativeRef(ref) {
		return nativeDependencyUse{}, false
	}
	return nativeDependencyUse{
		repository: strings.ToLower(path.Join(owner, repository)),
		ref:        ref,
		actionPath: actionPath,
	}, true
}

func nativeRelativeWorkflowPath(value string) (string, bool) {
	if !strings.HasPrefix(value, "./") {
		return "", false
	}
	relativePath := strings.TrimPrefix(value, "./")
	for part := range strings.SplitSeq(relativePath, "/") {
		if !validNativeRepositoryPart(part) {
			return "", false
		}
	}
	cleanPath := path.Clean(relativePath)
	if !strings.HasPrefix(cleanPath, logsWorkflowsPathSegment) ||
		(!strings.HasSuffix(cleanPath, ".yml") && !strings.HasSuffix(cleanPath, ".yaml")) {
		return "", false
	}
	return cleanPath, true
}

func validNativeRef(ref string) bool {
	if ref == "" || strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") ||
		strings.Contains(ref, "//") || strings.Contains(ref, "..") || strings.Contains(ref, "@{") {
		return false
	}
	for _, char := range ref {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9', char == '-', char == '_',
			char == '.', char == '/', char == '+', char == '@':
		default:
			return false
		}
	}
	return true
}

func validNativeRepositoryPart(part string) bool {
	if part == "" || part == "." || part == ".." {
		return false
	}
	for _, char := range part {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z',
			char >= '0' && char <= '9', char == '-', char == '_', char == '.':
		default:
			return false
		}
	}
	return true
}

func nativeDependencyKey(use nativeDependencyUse) string {
	return strings.ToLower(use.repository) + "@" + use.ref
}

func nativeDependencyVisitKey(use nativeDependencyUse) string {
	return strings.Join([]string{nativeDependencyKey(use), use.actionPath, strconv.FormatBool(use.selfRepository)}, "\x00")
}

func collectNativeDependency(
	ctx context.Context,
	use nativeDependencyUse,
	refHints map[string]string,
	resolve nativeDependencyResolver,
	dependencies map[string]nativeDependency,
	pins map[string]struct{},
	visited map[string]struct{},
) ([]string, error) {
	key := nativeDependencyKey(use)
	if !use.selfRepository {
		pins[key] = struct{}{}
	}
	visitKey := nativeDependencyVisitKey(use)
	if _, ok := visited[visitKey]; ok {
		return nil, nil
	}
	visited[visitKey] = struct{}{}
	defer delete(visited, visitKey)

	var existing *nativeDependency
	if dependency, ok := dependencies[key]; ok && !use.selfRepository {
		existing = &dependency
	}
	resolution, err := resolve(ctx, use, refHints[key], existing)
	if err != nil {
		return nil, err
	}
	if use.selfRepository {
		return collectNativeSelfDependencies(ctx, resolution.uses, refHints, resolve, dependencies, pins, visited)
	}
	dependency := dependencies[key]
	if dependency.Commit != "" && (dependency.Commit != resolution.commit || dependency.RepoID != resolution.repoID) {
		return nil, fmt.Errorf("conflicting resolutions for %s", key)
	}
	dependency.Hostname = resolution.hostname
	dependency.Ref = resolution.ref
	dependency.Commit = resolution.commit
	dependency.OwnerID = resolution.ownerID
	dependency.RepoID = resolution.repoID
	dependencies[key] = dependency
	childPins, err := collectNativeDependencyChildren(ctx, key, resolution.uses, refHints, resolve, dependencies, pins, visited)
	if err != nil {
		return nil, err
	}
	dependency = dependencies[key]
	dependency.Uses = sortedNativeKeys(childPins)
	dependencies[key] = dependency
	return []string{key}, nil
}

func collectNativeSelfDependencies(
	ctx context.Context,
	children []nativeDependencyUse,
	refHints map[string]string,
	resolve nativeDependencyResolver,
	dependencies map[string]nativeDependency,
	pins map[string]struct{},
	visited map[string]struct{},
) ([]string, error) {
	childPins := make(map[string]struct{})
	for _, child := range children {
		childKeys, err := collectNativeDependency(ctx, child, refHints, resolve, dependencies, pins, visited)
		if err != nil {
			return nil, err
		}
		for _, childKey := range childKeys {
			childPins[childKey] = struct{}{}
		}
	}
	return sortedNativeKeys(childPins), nil
}

func collectNativeDependencyChildren(
	ctx context.Context,
	key string,
	children []nativeDependencyUse,
	refHints map[string]string,
	resolve nativeDependencyResolver,
	dependencies map[string]nativeDependency,
	pins map[string]struct{},
	visited map[string]struct{},
) (map[string]struct{}, error) {
	childPins := make(map[string]struct{}, len(children)+len(dependencies[key].Uses))
	for _, childKey := range dependencies[key].Uses {
		childPins[childKey] = struct{}{}
	}
	for _, child := range children {
		childKey := nativeDependencyKey(child)
		if child.selfRepository {
			childKeys, err := collectNativeDependency(ctx, child, refHints, resolve, dependencies, pins, visited)
			if err != nil {
				return nil, err
			}
			for _, key := range childKeys {
				childPins[key] = struct{}{}
			}
		} else {
			childKeys, err := collectNativeDependency(ctx, child, refHints, resolve, dependencies, pins, visited)
			if err != nil {
				return nil, err
			}
			if childKey != key && slices.Contains(childKeys, childKey) {
				childPins[childKey] = struct{}{}
			}
		}
	}
	for _, childKey := range dependencies[key].Uses {
		if childKey != key {
			childPins[childKey] = struct{}{}
		}
	}
	delete(childPins, key)
	return childPins, nil
}

func sortedNativeKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func normalizeNativeWorkflowPins(lockfile *nativeActionsLockfile) {
	referenced := make(map[string]struct{})
	for workflowPath, pins := range lockfile.Workflows {
		workflowPins := make(map[string]struct{}, len(pins))
		pending := make(map[string]struct{}, len(pins))
		for _, pin := range pins {
			pending[pin] = struct{}{}
		}
		for len(pending) > 0 {
			keys := sortedNativeKeys(pending)
			pending = make(map[string]struct{})
			for _, key := range keys {
				if _, seen := workflowPins[key]; seen {
					continue
				}
				workflowPins[key] = struct{}{}
				referenced[key] = struct{}{}
				if dependency, ok := lockfile.Dependencies[key]; ok {
					for _, child := range dependency.Uses {
						pending[child] = struct{}{}
					}
				}
			}
		}
		lockfile.Workflows[workflowPath] = sortedNativeKeys(workflowPins)
	}
	for key := range lockfile.Dependencies {
		if _, used := referenced[key]; !used {
			delete(lockfile.Dependencies, key)
		}
	}
}

func writeNativeActionsLockfile(lockPath string, content []byte) error {
	if existing, err := os.ReadFile(lockPath); err == nil && bytes.Equal(existing, content) {
		return nil
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read GitHub Actions lockfile: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create GitHub Actions lockfile directory: %w", err)
	}
	tempFile, err := os.CreateTemp(filepath.Dir(lockPath), ".actions.lock-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary GitHub Actions lockfile: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	if err := tempFile.Chmod(constants.FilePermPublic); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("failed to set GitHub Actions lockfile permissions: %w", err), closeErr)
		}
		return fmt.Errorf("failed to set GitHub Actions lockfile permissions: %w", err)
	}
	if _, err := tempFile.Write(content); err != nil {
		if closeErr := tempFile.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("failed to write GitHub Actions lockfile: %w", err), closeErr)
		}
		return fmt.Errorf("failed to write GitHub Actions lockfile: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close GitHub Actions lockfile: %w", err)
	}
	if err := os.Rename(tempPath, lockPath); err != nil {
		return fmt.Errorf("failed to replace GitHub Actions lockfile: %w", err)
	}
	return nil
}

func resolveNativeDependency(
	ctx context.Context,
	use nativeDependencyUse,
	refHint string,
	actionResolver *workflow.ActionResolver,
	repositoryIDs map[string]nativeRepositoryIDs,
	existing *nativeDependency,
) (nativeDependencyResolution, error) {
	commit, repoIDs, hostname, resolvedRef, err := resolveNativeDependencyPin(ctx, use, refHint, actionResolver, repositoryIDs, existing)
	if err != nil {
		return nativeDependencyResolution{}, err
	}
	_, children, err := readNativeDependencyUses(ctx, use, commit)
	if err != nil {
		return nativeDependencyResolution{}, err
	}
	return nativeDependencyResolution{
		hostname: hostname,
		ref:      resolvedRef,
		commit:   "sha1-" + strings.ToLower(commit),
		ownerID:  repoIDs.ownerID,
		repoID:   repoIDs.repoID,
		uses:     children,
	}, nil
}

func resolveNativeDependencyPin(
	ctx context.Context,
	use nativeDependencyUse,
	refHint string,
	actionResolver *workflow.ActionResolver,
	repositoryIDs map[string]nativeRepositoryIDs,
	existing *nativeDependency,
) (string, nativeRepositoryIDs, string, string, error) {
	commit := use.ref
	repoIDs := nativeRepositoryIDs{}
	hostname := "github.com"
	resolvedRef := use.ref
	if use.selfRepository {
		commit = strings.TrimPrefix(commit, "sha1-")
		if !fullCommitSHA.MatchString(commit) || use.repository == "" || use.actionPath == "" {
			return "", repoIDs, "", "", fmt.Errorf("invalid self-repository reference %q", use.actionPath)
		}
		return commit, repoIDs, hostname, resolvedRef, nil
	}
	if existing != nil {
		commit = strings.TrimPrefix(existing.Commit, "sha1-")
		repoIDs = nativeRepositoryIDs{ownerID: existing.OwnerID, repoID: existing.RepoID}
		hostname = existing.Hostname
		resolvedRef = existing.Ref
		if !fullCommitSHA.MatchString(commit) || repoIDs.ownerID <= 0 || repoIDs.repoID <= 0 {
			return "", repoIDs, "", "", fmt.Errorf("invalid existing resolution for %s", nativeDependencyKey(use))
		}
		return commit, repoIDs, hostname, resolvedRef, nil
	}
	if !fullCommitSHA.MatchString(commit) {
		if actionResolver == nil {
			return "", repoIDs, "", "", fmt.Errorf("no action resolver is available for %s", nativeDependencyKey(use))
		}
		resolvedCommit, err := actionResolver.ResolveSHA(ctx, use.repository, use.ref)
		if err != nil {
			return "", repoIDs, "", "", fmt.Errorf("failed to resolve %s: %w", nativeDependencyKey(use), err)
		}
		commit = resolvedCommit
	}
	if !fullCommitSHA.MatchString(commit) {
		return "", repoIDs, "", "", fmt.Errorf("resolved %s to an invalid commit SHA", nativeDependencyKey(use))
	}
	var ok bool
	repoIDs, ok = repositoryIDs[use.repository]
	if !ok {
		var err error
		repoIDs, err = getNativeRepositoryIDs(ctx, use.repository)
		if err != nil {
			return "", repoIDs, "", "", err
		}
		repositoryIDs[use.repository] = repoIDs
	}
	if actionResolver != nil {
		if validatedHint := validateNativeRefHint(ctx, use.repository, refHint, use.ref, commit, actionResolver.ResolveSHA); validatedHint != "" {
			resolvedRef = validatedHint
		}
	}
	return commit, repoIDs, hostname, resolvedRef, nil
}

func readNativeDependencyUses(ctx context.Context, use nativeDependencyUse, commit string) (string, []nativeDependencyUse, error) {
	sourcePath := use.actionPath
	isReusableWorkflow := strings.HasPrefix(sourcePath, logsWorkflowsPathSegment)
	if sourcePath == "" {
		sourcePath = "action.yml"
	} else if !isReusableWorkflow {
		sourcePath = path.Join(sourcePath, "action.yml")
	}
	content, err := getNativeDependencyFile(ctx, use.repository, sourcePath, commit)
	if err != nil && !isReusableWorkflow {
		content, err = getNativeDependencyFile(ctx, use.repository, strings.TrimSuffix(sourcePath, ".yml")+".yaml", commit)
	}
	if err != nil {
		return sourcePath, nil, err
	}
	var children []nativeDependencyUse
	if isReusableWorkflow {
		children, err = parseNativeWorkflowUsesWithContext(content, use.repository, commit)
	} else {
		children, err = parseNativeActionUses(content)
	}
	if err != nil {
		return sourcePath, nil, fmt.Errorf("failed to parse %s at %s: %w", sourcePath, nativeDependencyKey(use), err)
	}
	return sourcePath, children, nil
}

func validateNativeRefHint(
	ctx context.Context,
	repository, hint, sourceRef, commit string,
	resolveSHA func(context.Context, string, string) (string, error),
) string {
	if hint == "" || hint == sourceRef || !validNativeRef(hint) || resolveSHA == nil {
		return ""
	}
	hintedCommit, err := resolveSHA(ctx, repository, hint)
	if err != nil || !strings.EqualFold(hintedCommit, commit) {
		return ""
	}
	return hint
}

type nativeRepositoryIDs struct {
	ownerID int64
	repoID  int64
}

func getNativeRepositoryIDs(ctx context.Context, repository string) (nativeRepositoryIDs, error) {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := workflow.ExecGHContext(callCtx, "api", "/repos/"+repository, "--jq", "[.owner.id, .id] | @tsv")
	workflow.ForceGHHostEnv(cmd, "github.com")
	output, err := cmd.Output()
	if err != nil {
		return nativeRepositoryIDs{}, fmt.Errorf("failed to retrieve repository identity for %s: %w", repository, err)
	}
	ownerIDText, repoIDText, found := strings.Cut(strings.TrimSpace(string(output)), "\t")
	if !found || strings.Contains(repoIDText, "\t") {
		return nativeRepositoryIDs{}, fmt.Errorf("unexpected repository identity response for %s", repository)
	}
	ownerID, err := strconv.ParseInt(ownerIDText, 10, 64)
	if err != nil || ownerID <= 0 {
		return nativeRepositoryIDs{}, fmt.Errorf("invalid owner ID returned for %s", repository)
	}
	repoID, err := strconv.ParseInt(repoIDText, 10, 64)
	if err != nil || repoID <= 0 {
		return nativeRepositoryIDs{}, fmt.Errorf("invalid repository ID returned for %s", repository)
	}
	return nativeRepositoryIDs{ownerID: ownerID, repoID: repoID}, nil
}

func getNativeDependencyFile(ctx context.Context, repository, filePath, ref string) ([]byte, error) {
	apiPath := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repository, filePath, url.QueryEscape(ref))
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := workflow.ExecGHContext(callCtx, "api", apiPath, "--jq", ".content")
	workflow.ForceGHHostEnv(cmd, "github.com")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve %s from %s at %s: %w", filePath, repository, ref, err)
	}
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s from %s at %s: %w", filePath, repository, ref, err)
	}
	return content, nil
}
