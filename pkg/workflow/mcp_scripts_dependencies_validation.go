//go:build !js && !wasm

package workflow

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/semverutil"
)

var (
	goModuleNameRE               = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
	shellPackageDependencyNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*([:=][A-Za-z0-9.+~:-]+)?$`)
)

var mcpScriptDepsValidationLog = newValidationLogger("mcp_script_dependencies")

func (c *Compiler) validateMCPScriptDependencies(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.MCPScripts == nil {
		return nil
	}

	mcpScriptDepsValidationLog.Printf("Validating MCP script dependencies for %d tool(s)", len(workflowData.MCPScripts.Tools))

	for toolName, tool := range workflowData.MCPScripts.Tools {
		if len(tool.Dependencies) == 0 {
			continue
		}

		manager := inferMCPScriptDependencyManager(tool)
		if manager == "" {
			mcpScriptDepsValidationLog.Printf("Tool %q: no dependency manager inferred, skipping %d dependencies", toolName, len(tool.Dependencies))
			continue
		}

		mcpScriptDepsValidationLog.Printf("Tool %q: validating %d %q dependencies", toolName, len(tool.Dependencies), manager)

		for _, dependency := range tool.Dependencies {
			dependency = strings.TrimSpace(dependency)
			if dependency == "" {
				continue
			}
			if err := validateMCPScriptDependencyName(toolName, manager, dependency); err != nil {
				return err
			}
		}
	}

	return nil
}

func inferMCPScriptDependencyManager(tool *MCPScriptToolConfig) string {
	switch {
	case tool.Script != "":
		return "npm"
	case tool.Py != "":
		return "pip"
	case tool.Go != "":
		return "go"
	case tool.Run != "":
		return "apt"
	default:
		return ""
	}
}

func validateMCPScriptDependencyName(toolName, manager, dependency string) error {
	switch manager {
	case "npm":
		name, version, err := splitNpmDependency(dependency)
		if err != nil {
			if nameErr := validateNpmPackageName(dependency); nameErr != nil {
				return newInvalidDependencyNameError(toolName, dependency)
			}
			return newUnpinnedDependencyError(toolName, dependency, "npm", "name@1.2.3", "@scope/name@1.2.3")
		}
		if err := validateNpmPackageName(name); err != nil {
			return newInvalidDependencyNameError(toolName, dependency)
		}
		if !semverutil.IsValid(version) {
			return newUnpinnedDependencyError(toolName, dependency, "npm", "name@1.2.3", "@scope/name@1.2.3")
		}
	case "pip":
		name := dependency
		if idx := strings.IndexAny(name, "=<>!~"); idx > 0 {
			name = name[:idx]
		}
		name = strings.TrimSpace(name)
		if err := validatePipPackageName(name); err != nil {
			return newInvalidDependencyNameError(toolName, dependency)
		}
		_, version, found := strings.Cut(dependency, "==")
		if !found {
			return newUnpinnedDependencyError(toolName, dependency, "pip", "name==1.2.3", "requests==2.32.3")
		}
		version = strings.TrimSpace(version)
		if version == "" {
			return newUnpinnedDependencyError(toolName, dependency, "pip", "name==1.2.3", "requests==2.32.3")
		}
		if !semverutil.IsValid(version) {
			return newUnpinnedDependencyError(toolName, dependency, "pip", "name==1.2.3", "requests==2.32.3")
		}
	case "go":
		moduleName, version, found := strings.Cut(dependency, "@")
		if !found {
			if !goModuleNameRE.MatchString(strings.TrimSpace(dependency)) {
				return newInvalidDependencyNameError(toolName, dependency)
			}
			return newUnpinnedDependencyError(toolName, dependency, "go", "module@v1.2.3", "github.com/google/uuid@v1.6.0")
		}
		moduleName = strings.TrimSpace(moduleName)
		version = strings.TrimSpace(version)
		if moduleName == "" || version == "" {
			return newUnpinnedDependencyError(toolName, dependency, "go", "module@v1.2.3", "github.com/google/uuid@v1.6.0")
		}
		if !goModuleNameRE.MatchString(moduleName) {
			return newInvalidDependencyNameError(toolName, dependency)
		}
		if !strings.HasPrefix(version, "v") || !semverutil.IsValid(version) {
			return newUnpinnedDependencyError(toolName, dependency, "go", "module@v1.2.3", "github.com/google/uuid@v1.6.0")
		}
	case "apt":
		if !shellPackageDependencyNameRE.MatchString(dependency) {
			return newInvalidDependencyNameError(toolName, dependency)
		}
		hasVersionPin := strings.Contains(dependency, "=") || strings.Contains(dependency, ":")
		if !hasVersionPin {
			return newUnpinnedDependencyError(toolName, dependency, "shell", "name=1.2.3", "jq=1.6-2.1")
		}
	}

	return nil
}

func splitNpmDependency(dependency string) (string, string, error) {
	atIndex := strings.LastIndex(dependency, "@")
	if atIndex <= 0 || atIndex == len(dependency)-1 {
		return "", "", errors.New("missing npm package version")
	}
	name := strings.TrimSpace(dependency[:atIndex])
	version := strings.TrimSpace(dependency[atIndex+1:])
	if name == "" || version == "" {
		return "", "", errors.New("missing npm package version")
	}
	return name, version, nil
}

func newInvalidDependencyNameError(toolName, dependency string) error {
	return fmt.Errorf(
		"invalid dependency name %q for tool %q. Expected a valid package name for the inferred package manager. Example: dependencies: [requests]",
		dependency,
		toolName,
	)
}

func newUnpinnedDependencyError(toolName, dependency, manager, expected, example string) error {
	return fmt.Errorf(
		"dependency %q for tool %q is not pinned to a release tag. Expected %s dependency format %q with an exact version. Example: dependencies: [%q]",
		dependency,
		toolName,
		manager,
		expected,
		example,
	)
}
