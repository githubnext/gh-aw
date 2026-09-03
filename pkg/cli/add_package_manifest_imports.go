package cli

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

type repositoryPackageManifestNode struct {
	Path        string
	PackagePath string
	Manifest    *repositoryPackageManifest
}

type repositoryPackageManifestGraphResolver struct {
	rootPackagePath string
	readManifest    func(string) ([]byte, error)
	states          map[string]uint8
	stack           []string
	nodes           []repositoryPackageManifestNode
	warnings        []string
}

func isManifestImportPath(importPath string) bool {
	_, err := cleanManifestImportPath(importPath)
	return err == nil && path.Base(filepath.ToSlash(importPath)) == repositoryPackageManifestFileName
}

func cleanManifestImportPath(importPath string) (string, error) {
	if importPath == "" {
		return "", errors.New("path must not be empty")
	}
	slashed := filepath.ToSlash(importPath)
	if slashed[0] == '/' || slashed[0] == '\\' || filepath.IsAbs(importPath) || isWindowsDriveRelativePath(slashed) {
		return "", errors.New("absolute paths are not allowed")
	}
	cleaned := path.Clean(slashed)
	if cleaned == "." {
		return "", errors.New("path must not be empty")
	}
	if path.Base(cleaned) != repositoryPackageManifestFileName {
		return "", fmt.Errorf("path must name an %s manifest", repositoryPackageManifestFileName)
	}
	return cleaned, nil
}

func resolveRepositoryPackageManifestGraph(
	rootPath string,
	root *repositoryPackageManifest,
	readManifest func(string) ([]byte, error),
) ([]repositoryPackageManifestNode, []string, error) {
	rootPath = path.Clean(filepath.ToSlash(rootPath))
	rootPackagePath := path.Dir(rootPath)
	if rootPackagePath == "." {
		rootPackagePath = ""
	}

	resolver := &repositoryPackageManifestGraphResolver{
		rootPackagePath: rootPackagePath,
		readManifest:    readManifest,
		states:          make(map[string]uint8),
	}
	if err := resolver.visit(rootPath, root); err != nil {
		return nil, nil, err
	}
	return resolver.nodes, resolver.warnings, nil
}

func (r *repositoryPackageManifestGraphResolver) visit(manifestPath string, manifest *repositoryPackageManifest) error {
	switch r.states[manifestPath] {
	case 1:
		cycleStart := 0
		for i, item := range r.stack {
			if item == manifestPath {
				cycleStart = i
				break
			}
		}
		cycle := append(append([]string{}, r.stack[cycleStart:]...), manifestPath)
		return fmt.Errorf("package manifest import cycle detected: %s", strings.Join(cycle, " -> "))
	case 2:
		return nil
	}

	r.states[manifestPath] = 1
	r.stack = append(r.stack, manifestPath)
	manifestDir := path.Dir(manifestPath)
	if manifestDir == "." {
		manifestDir = ""
	}
	for _, relativeImport := range manifest.Imports {
		importPath := path.Clean(path.Join(manifestDir, relativeImport))
		if !isPathWithinPackageRoot(importPath, r.rootPackagePath) {
			return fmt.Errorf("invalid Agentic Workflow manifest %q: import %q resolves outside the package root", manifestPath, relativeImport)
		}
		switch r.states[importPath] {
		case 1:
			if err := r.visit(importPath, nil); err != nil {
				return err
			}
		case 2:
			continue
		}
		content, err := r.readManifest(importPath)
		if err != nil {
			return fmt.Errorf("failed to read imported Agentic Workflow manifest %q from %q: %w", importPath, manifestPath, err)
		}
		imported, importedWarnings, err := parseRepositoryPackageManifest(importPath, content)
		if err != nil {
			return err
		}
		r.warnings = append(r.warnings, importedWarnings...)
		if err := r.visit(importPath, imported); err != nil {
			return err
		}
	}
	r.stack = r.stack[:len(r.stack)-1]
	r.states[manifestPath] = 2
	r.nodes = append(r.nodes, repositoryPackageManifestNode{Path: manifestPath, PackagePath: manifestDir, Manifest: manifest})
	return nil
}

func isPathWithinPackageRoot(candidate, root string) bool {
	if root == "" {
		return candidate != ".." && !strings.HasPrefix(candidate, "../")
	}
	return candidate == root || strings.HasPrefix(candidate, root+"/")
}

func validateUniqueResolvedPackageFiles(
	installables []resolvedPackageInstallable,
	resources []resolvedPackageResource,
	skillFiles []resolvedPackageSkillFile,
	agentFiles []string,
	manifestPath string,
) error {
	seen := make(map[string]string)
	add := func(destination, source string) error {
		key := strings.ToLower(filepath.ToSlash(filepath.Clean(destination)))
		if previous, exists := seen[key]; exists {
			return fmt.Errorf("invalid Agentic Workflow manifest %q: files %q and %q both install to %q", manifestPath, previous, source, destination)
		}
		seen[key] = source
		return nil
	}
	for _, installable := range installables {
		if err := add(installable.DestinationPath, installable.SourcePath); err != nil {
			return err
		}
	}
	for _, resource := range resources {
		if err := add(resource.DestinationPath, resource.SourcePath); err != nil {
			return err
		}
	}
	for _, skillFile := range skillFiles {
		relative := packageSkillFileRelativePath(skillFile)
		destination := path.Join(constants.GithubDir+packageSkillsDirectory, skillFile.SkillName, filepath.ToSlash(relative))
		if err := add(destination, skillFile.SourcePath); err != nil {
			return err
		}
	}
	for _, agentFile := range agentFiles {
		destination := path.Join(constants.GithubDir+packageAgentsDirectory, filepath.Base(agentFile))
		if err := add(destination, agentFile); err != nil {
			return err
		}
	}
	return nil
}

func packageSkillFileRelativePath(skillFile resolvedPackageSkillFile) string {
	parts := strings.Split(filepath.ToSlash(skillFile.SourcePath), "/")
	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == skillFile.SkillName && parts[i-1] == "skills" {
			return path.Join(parts[i+1:]...)
		}
	}
	return filepath.Base(skillFile.SourcePath)
}

func readLocalImportedManifest(manifestPath, packageRoot string) ([]byte, error) {
	evaluatedPath, err := filepath.EvalSymlinks(manifestPath)
	if err != nil {
		return nil, err
	}
	evaluatedRoot, err := filepath.EvalSymlinks(packageRoot)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(evaluatedRoot, evaluatedPath)
	if err != nil {
		return nil, err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("import %q resolves outside the package root", manifestPath)
	}
	declaredRelative, err := filepath.Rel(filepath.Clean(packageRoot), filepath.Clean(manifestPath))
	if err != nil {
		return nil, err
	}
	if filepath.Clean(relative) != filepath.Clean(declaredRelative) {
		return nil, errors.New("imported manifests must not use symbolic links")
	}
	return os.ReadFile(manifestPath)
}
