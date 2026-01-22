package workflow

import (
	"sort"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var runtimeMountsLog = logger.New("workflow:runtime_mounts")

// RuntimeMountDefinition defines mount paths for a specific runtime
type RuntimeMountDefinition struct {
	// RuntimeID is the unique identifier for the runtime (e.g., "node", "python")
	RuntimeID string

	// Mounts is a list of mount specifications in the format "source:dest:mode"
	// These mounts make runtime binaries, caches, and supporting folders
	// available inside the agent container
	Mounts []string
}

// GetRuntimeMounts returns mount specifications for detected runtimes
// This ensures that runtime binaries, caches, and other supporting folders
// are available inside the agent container
func GetRuntimeMounts(requirements []RuntimeRequirement) []string {
	runtimeMountsLog.Printf("Generating runtime mounts for %d requirements", len(requirements))

	// Collect unique mounts across all detected runtimes
	mountSet := make(map[string]bool)

	for _, req := range requirements {
		mountDef := getRuntimeMountDefinition(req.Runtime.ID)
		if mountDef != nil {
			for _, mount := range mountDef.Mounts {
				mountSet[mount] = true
			}
			runtimeMountsLog.Printf("Added %d mounts for runtime: %s", len(mountDef.Mounts), req.Runtime.ID)
		}
	}

	// Convert set to sorted slice for consistent output
	mounts := make([]string, 0, len(mountSet))
	for mount := range mountSet {
		mounts = append(mounts, mount)
	}
	sort.Strings(mounts)

	runtimeMountsLog.Printf("Total unique runtime mounts: %d", len(mounts))
	return mounts
}

// getRuntimeMountDefinition returns mount definitions for a specific runtime
// Returns nil if the runtime has no special mount requirements
func getRuntimeMountDefinition(runtimeID string) *RuntimeMountDefinition {
	// Map of runtime IDs to their mount definitions
	// These paths are based on GitHub Actions runner standard locations
	definitions := map[string]*RuntimeMountDefinition{
		"node": {
			RuntimeID: "node",
			Mounts: []string{
				"/opt/hostedtoolcache/node:/opt/hostedtoolcache/node:ro",
				"/home/runner/.npm:/home/runner/.npm:rw",
			},
		},
		"python": {
			RuntimeID: "python",
			Mounts: []string{
				"/opt/hostedtoolcache/Python:/opt/hostedtoolcache/Python:ro",
				"/home/runner/.cache/pip:/home/runner/.cache/pip:rw",
			},
		},
		"go": {
			RuntimeID: "go",
			Mounts: []string{
				"/opt/hostedtoolcache/go:/opt/hostedtoolcache/go:ro",
				"/home/runner/go:/home/runner/go:rw",
				"/home/runner/.cache/go-build:/home/runner/.cache/go-build:rw",
			},
		},
		"ruby": {
			RuntimeID: "ruby",
			Mounts: []string{
				"/opt/hostedtoolcache/Ruby:/opt/hostedtoolcache/Ruby:ro",
				"/home/runner/.gem:/home/runner/.gem:rw",
			},
		},
		"java": {
			RuntimeID: "java",
			Mounts: []string{
				"/opt/hostedtoolcache/Java_Temurin-Hotspot_jdk:/opt/hostedtoolcache/Java_Temurin-Hotspot_jdk:ro",
				"/home/runner/.m2:/home/runner/.m2:rw",
				"/home/runner/.gradle:/home/runner/.gradle:rw",
			},
		},
		"dotnet": {
			RuntimeID: "dotnet",
			Mounts: []string{
				"/opt/hostedtoolcache/dotnet:/opt/hostedtoolcache/dotnet:ro",
				"/home/runner/.nuget:/home/runner/.nuget:rw",
			},
		},
		"bun": {
			RuntimeID: "bun",
			Mounts: []string{
				"/opt/hostedtoolcache/bun-x64:/opt/hostedtoolcache/bun-x64:ro",
				"/home/runner/.bun:/home/runner/.bun:rw",
			},
		},
		"deno": {
			RuntimeID: "deno",
			Mounts: []string{
				"/opt/hostedtoolcache/deno:/opt/hostedtoolcache/deno:ro",
				"/home/runner/.cache/deno:/home/runner/.cache/deno:rw",
			},
		},
		"uv": {
			RuntimeID: "uv",
			Mounts: []string{
				"/home/runner/.cache/uv:/home/runner/.cache/uv:rw",
			},
		},
		"elixir": {
			RuntimeID: "elixir",
			Mounts: []string{
				"/opt/hostedtoolcache/elixir:/opt/hostedtoolcache/elixir:ro",
			},
		},
		"haskell": {
			RuntimeID: "haskell",
			Mounts: []string{
				"/opt/hostedtoolcache/ghc:/opt/hostedtoolcache/ghc:ro",
				"/home/runner/.cabal:/home/runner/.cabal:rw",
				"/home/runner/.stack:/home/runner/.stack:rw",
			},
		},
	}

	return definitions[runtimeID]
}

// ContributeRuntimeMounts adds runtime-specific mounts to the sandbox agent configuration
// This is called during sandbox configuration to ensure runtime binaries and caches
// are available inside the agent container
func ContributeRuntimeMounts(agentConfig *AgentSandboxConfig, requirements []RuntimeRequirement) {
	if agentConfig == nil {
		runtimeMountsLog.Print("Agent config is nil, skipping runtime mounts contribution")
		return
	}

	runtimeMounts := GetRuntimeMounts(requirements)
	if len(runtimeMounts) == 0 {
		runtimeMountsLog.Print("No runtime mounts to contribute")
		return
	}

	// Initialize Mounts slice if nil
	if agentConfig.Mounts == nil {
		agentConfig.Mounts = make([]string, 0)
	}

	// Add runtime mounts to agent config
	// User-specified mounts take precedence (added first)
	// Runtime mounts are added at the end
	agentConfig.Mounts = append(agentConfig.Mounts, runtimeMounts...)
	runtimeMountsLog.Printf("Contributed %d runtime mounts to agent config", len(runtimeMounts))
}

// GetRuntimeMountsForWorkflow is a convenience function that extracts runtime requirements
// from workflow data and returns the corresponding mounts
func GetRuntimeMountsForWorkflow(workflowData *WorkflowData) []string {
	if workflowData == nil {
		return []string{}
	}

	// Detect runtime requirements
	requirements := DetectRuntimeRequirements(workflowData)

	// Get mounts for detected runtimes
	return GetRuntimeMounts(requirements)
}
