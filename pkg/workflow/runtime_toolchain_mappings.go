package workflow

import (
	"sort"
)

// RuntimeToolchainMapping represents environment variables and mounts that need to be
// passed to the agent container (e.g., Serena MCP server) so that the toolchain works properly.
// These mappings are collected from runtime setup actions like actions/setup-go, actions/setup-node, etc.
type RuntimeToolchainMapping struct {
	// RuntimeID identifies which runtime this mapping is for (e.g., "go", "node", "python")
	RuntimeID string

	// EnvVars maps environment variable names to their values/expressions
	// Example: {"GOPATH": "${{ env.GOPATH }}", "GOCACHE": "$HOME/go/cache"}
	EnvVars map[string]string

	// Mounts lists volume mounts needed for the toolchain to work
	// Format: "source:dest:mode" (e.g., "/home/runner/go:/home/runner/go:rw")
	Mounts []string
}

// ToolchainMappings holds all runtime toolchain mappings
type ToolchainMappings struct {
	// Mappings by runtime ID
	Mappings map[string]*RuntimeToolchainMapping
}

// NewToolchainMappings creates a new ToolchainMappings instance
func NewToolchainMappings() *ToolchainMappings {
	return &ToolchainMappings{
		Mappings: make(map[string]*RuntimeToolchainMapping),
	}
}

// AddMapping adds or updates a runtime toolchain mapping
func (tm *ToolchainMappings) AddMapping(runtimeID string, envVars map[string]string, mounts []string) {
	if tm.Mappings == nil {
		tm.Mappings = make(map[string]*RuntimeToolchainMapping)
	}

	if _, exists := tm.Mappings[runtimeID]; !exists {
		tm.Mappings[runtimeID] = &RuntimeToolchainMapping{
			RuntimeID: runtimeID,
			EnvVars:   make(map[string]string),
			Mounts:    []string{},
		}
	}

	// Merge environment variables
	for key, value := range envVars {
		tm.Mappings[runtimeID].EnvVars[key] = value
	}

	// Merge mounts (will be deduplicated later)
	tm.Mappings[runtimeID].Mounts = append(tm.Mappings[runtimeID].Mounts, mounts...)
}

// GetAllEnvVars returns all environment variables from all runtime mappings
func (tm *ToolchainMappings) GetAllEnvVars() map[string]string {
	result := make(map[string]string)
	for _, mapping := range tm.Mappings {
		for key, value := range mapping.EnvVars {
			result[key] = value
		}
	}
	return result
}

// GetAllMounts returns all mounts from all runtime mappings, sorted and deduplicated
func (tm *ToolchainMappings) GetAllMounts() []string {
	mountSet := make(map[string]bool)
	for _, mapping := range tm.Mappings {
		for _, mount := range mapping.Mounts {
			mountSet[mount] = true
		}
	}

	// Convert to slice and sort
	var mounts []string
	for mount := range mountSet {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)

	return mounts
}

// CollectToolchainMappings collects environment variables and mounts from detected runtime requirements
// This function determines what needs to be passed to the agent container for toolchains to work
func CollectToolchainMappings(requirements []RuntimeRequirement) *ToolchainMappings {
	mappings := NewToolchainMappings()

	for _, req := range requirements {
		runtime := req.Runtime
		runtimeID := runtime.ID

		envVars := make(map[string]string)
		var mounts []string

		// Collect runtime-specific environment variables and mounts
		switch runtimeID {
		case "go":
			// Go requires GOPATH, GOCACHE, and GOMODCACHE to be accessible
			envVars["GOPATH"] = "/home/runner/go"
			envVars["GOCACHE"] = "/home/runner/.cache/go-build"
			envVars["GOMODCACHE"] = "/home/runner/go/pkg/mod"

			// Mount Go directories
			mounts = append(mounts, "/home/runner/go:/home/runner/go:rw")
			mounts = append(mounts, "/home/runner/.cache/go-build:/home/runner/.cache/go-build:rw")

		case "node":
			// Node.js requires npm cache and node_modules
			envVars["NPM_CONFIG_CACHE"] = "/home/runner/.npm"

			// Mount npm cache
			mounts = append(mounts, "/home/runner/.npm:/home/runner/.npm:rw")

		case "python":
			// Python requires pip cache
			envVars["PIP_CACHE_DIR"] = "/home/runner/.cache/pip"

			// Mount pip cache
			mounts = append(mounts, "/home/runner/.cache/pip:/home/runner/.cache/pip:rw")

		case "uv":
			// uv uses its own cache directory
			envVars["UV_CACHE_DIR"] = "/home/runner/.cache/uv"

			// Mount uv cache
			mounts = append(mounts, "/home/runner/.cache/uv:/home/runner/.cache/uv:rw")

		case "ruby":
			// Ruby requires gem home
			envVars["GEM_HOME"] = "/home/runner/.gem"

			// Mount gem directory
			mounts = append(mounts, "/home/runner/.gem:/home/runner/.gem:rw")

		case "java":
			// Java/Maven requires M2 repository
			envVars["MAVEN_OPTS"] = "-Dmaven.repo.local=/home/runner/.m2/repository"

			// Mount Maven repository
			mounts = append(mounts, "/home/runner/.m2:/home/runner/.m2:rw")

		case "dotnet":
			// .NET requires NuGet packages directory
			envVars["NUGET_PACKAGES"] = "/home/runner/.nuget/packages"

			// Mount NuGet packages
			mounts = append(mounts, "/home/runner/.nuget:/home/runner/.nuget:rw")

			// Other runtimes can be added here as needed
		}

		if len(envVars) > 0 || len(mounts) > 0 {
			mappings.AddMapping(runtimeID, envVars, mounts)
		}
	}

	return mappings
}

// MergeMountsWithDedup merges two lists of mounts, removes duplicates, and sorts them
func MergeMountsWithDedup(existingMounts []string, newMounts []string) []string {
	mountSet := make(map[string]bool)

	// Add existing mounts
	for _, mount := range existingMounts {
		mountSet[mount] = true
	}

	// Add new mounts
	for _, mount := range newMounts {
		mountSet[mount] = true
	}

	// Convert to slice and sort
	var result []string
	for mount := range mountSet {
		result = append(result, mount)
	}
	sort.Strings(result)

	return result
}
