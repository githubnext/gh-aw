package cli

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

var sandboxRuntimeProfileCodemodLog = logger.New("cli:codemod_sandbox_runtime_profile")

const (
	sandboxRuntimeDocker             = "docker"
	sandboxRuntimeDockerSudoIptables = "docker-sudo-iptables"
	sandboxRuntimeCloudHypervisor    = "cloud-hypervisor"
)

// getSandboxRuntimeProfileCodemod creates a codemod that migrates the removed
// sandbox.agent.sudo and sandbox.agent.legacy-security fields to the equivalent
// sandbox.agent.runtime profile.
//
//	sudo: false (or omitted)          -> runtime omitted (equivalent to runtime: docker)
//	legacy-security: enable           -> runtime: docker-sudo-iptables
//	sudo: true (no other runtime)     -> runtime: docker-sudo-iptables
//
// Other mixed profiles that cannot be migrated unambiguously return an actionable error
// so the author can choose between strict isolation and the privileged iptables profile.
func getSandboxRuntimeProfileCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-runtime-profiles",
		Name:         "Migrate sandbox.agent.sudo and legacy-security to runtime profiles",
		Description:  "Replaces the removed 'sandbox.agent.sudo' and 'sandbox.agent.legacy-security' fields with the equivalent 'sandbox.agent.runtime' profile",
		IntroducedIn: "0.42.0",
		Apply: func(content string, frontmatter map[string]any) (string, bool, error) {
			agent, ok := getSandboxAgentMap(frontmatter)
			if !ok {
				return content, false, nil
			}

			sudoVal, hasSudo := agent["sudo"]
			legacyVal, hasLegacy := agent["legacy-security"]
			if !hasSudo && !hasLegacy {
				return content, false, nil
			}

			sudoEnabled := sudoVal == true
			legacyEnabled := false
			if legacyStr, isStr := legacyVal.(string); isStr && legacyStr == "enable" {
				legacyEnabled = true
			}

			var runtime string
			if configured, ok := agent["runtime"].(string); ok {
				runtime = configured
			}
			targetRuntime, err := resolveMigratedSandboxRuntime(runtime, sudoEnabled, legacyEnabled)
			if err != nil {
				return content, false, err
			}

			newContent, applied, err := codemod.Apply(content, func(f *codemod.Frontmatter) error {
				if targetRuntime != "" {
					if err := f.Set([]string{"sandbox", "agent", "runtime"}, targetRuntime); err != nil {
						return err
					}
				}
				for _, key := range []string{"sudo", "legacy-security"} {
					if err := f.Remove("sandbox", "agent", key); err != nil {
						return err
					}
				}
				if err := f.PruneEmpty("sandbox", "agent"); err != nil {
					return err
				}
				return f.PruneEmpty("sandbox")
			})
			if applied {
				sandboxRuntimeProfileCodemodLog.Printf("Migrated sandbox.agent security fields to runtime profile %q", targetRuntime)
			}
			return newContent, applied, err
		},
	}
}

// resolveMigratedSandboxRuntime returns the runtime value the workflow should use
// after migration, or an empty string when no runtime key is required.
func resolveMigratedSandboxRuntime(runtime string, sudoEnabled, legacyEnabled bool) (string, error) {
	privileged := sudoEnabled || legacyEnabled

	// sudo: false / legacy-security absent keeps the secure default: the fields are
	// simply dropped and the runtime (if any) is preserved.
	if !privileged {
		return "", nil
	}

	switch runtime {
	case "", sandboxRuntimeDocker, sandboxRuntimeDockerSudoIptables:
		return sandboxRuntimeDockerSudoIptables, nil
	case sandboxRuntimeCloudHypervisor:
		// This runtime only used sudo as an installation marker; the compiler now
		// derives the required privileges, so the runtime is kept unchanged.
		if legacyEnabled {
			return "", mixedSandboxProfileError(runtime)
		}
		return "", nil
	default:
		return "", mixedSandboxProfileError(runtime)
	}
}

func mixedSandboxProfileError(runtime string) error {
	return fmt.Errorf(
		"sandbox.agent.runtime: %s combined with privileged security options (sudo/legacy-security) is no longer supported. "+
			"Choose one runtime profile: keep 'runtime: %s' for strict network isolation and remove 'sudo'/'legacy-security', "+
			"or use 'runtime: %s' for the privileged iptables profile with host access",
		runtime, runtime, sandboxRuntimeDockerSudoIptables,
	)
}

// getSandboxAgentMap returns the parsed sandbox.agent mapping from frontmatter.
func getSandboxAgentMap(frontmatter map[string]any) (map[string]any, bool) {
	sandboxVal, ok := frontmatter["sandbox"]
	if !ok {
		return nil, false
	}
	sandboxMap, ok := sandboxVal.(map[string]any)
	if !ok {
		return nil, false
	}
	agentMap, ok := sandboxMap["agent"].(map[string]any)
	return agentMap, ok
}
