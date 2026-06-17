//go:build !js && !wasm

package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	goModuleDependencyNameRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*(@[A-Za-z0-9._+\-]+)?$`)
	shellPackageDependencyNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*([:=][A-Za-z0-9.+~:-]+)?$`)
)

func (c *Compiler) validateMCPScriptDependencies(workflowData *WorkflowData) error {
	if workflowData == nil || workflowData.MCPScripts == nil {
		return nil
	}

	for toolName, tool := range workflowData.MCPScripts.Tools {
		if len(tool.Dependencies) == 0 {
			continue
		}

		manager := inferMCPScriptDependencyManager(tool)
		if manager == "" {
			continue
		}

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
		if err := validateNpmPackageName(dependency); err != nil {
			return newInvalidDependencyNameError(toolName, dependency)
		}
	case "pip":
		name := dependency
		if idx := strings.IndexAny(name, "=<>!~"); idx > 0 {
			name = name[:idx]
		}
		if err := validatePipPackageName(name); err != nil {
			return newInvalidDependencyNameError(toolName, dependency)
		}
	case "go":
		if !goModuleDependencyNameRE.MatchString(dependency) {
			return newInvalidDependencyNameError(toolName, dependency)
		}
	case "apt":
		if !shellPackageDependencyNameRE.MatchString(dependency) {
			return newInvalidDependencyNameError(toolName, dependency)
		}
	}

	return nil
}

func newInvalidDependencyNameError(toolName, dependency string) error {
	return fmt.Errorf(
		"invalid dependency name %q for tool %q. Expected a valid package name for the inferred package manager. Example: dependencies: [requests]",
		dependency,
		toolName,
	)
}
