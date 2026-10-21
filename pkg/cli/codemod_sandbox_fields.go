package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/github/gh-aw/pkg/workflow/codemod"
)

type sandboxFieldRename struct {
	parent []string
	oldKey string
	newKey string
}

var sandboxFieldRenames = []sandboxFieldRename{
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "apiProxy", newKey: "api-proxy"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "cliProxy", newKey: "cli-proxy"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "buildTools", newKey: "build-tools"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "dohProxy", newKey: "doh-proxy"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "enclaveScript", newKey: "enclave-script"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "enclaveAgent", newKey: "enclave-agent"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "enclaveMcpServer", newKey: "enclave-mcp-server"},
	{parent: []string{"sandbox", "agent", "images"}, oldKey: "dindStaging", newKey: "dind-staging"},
	{parent: []string{"sandbox", "agent", "config", "filesystem"}, oldKey: "denyRead", newKey: "deny-read"},
	{parent: []string{"sandbox", "agent", "config", "filesystem"}, oldKey: "allowWrite", newKey: "allow-write"},
	{parent: []string{"sandbox", "agent", "config", "filesystem"}, oldKey: "denyWrite", newKey: "deny-write"},
	{parent: []string{"sandbox", "agent", "config"}, oldKey: "ignoreViolations", newKey: "ignore-violations"},
	{parent: []string{"sandbox", "agent", "config"}, oldKey: "enableWeakerNestedSandbox", newKey: "enable-weaker-nested-sandbox"},
	{parent: []string{"sandbox", "agent", "config", "network"}, oldKey: "allowedDomains", newKey: "allowed-domains"},
	{parent: []string{"sandbox", "agent", "config", "network"}, oldKey: "blockedDomains", newKey: "blocked-domains"},
	{parent: []string{"sandbox", "agent", "config", "network"}, oldKey: "allowUnixSockets", newKey: "allow-unix-sockets"},
	{parent: []string{"sandbox", "agent", "config", "network"}, oldKey: "allowLocalBinding", newKey: "allow-local-binding"},
	{parent: []string{"sandbox", "agent", "config", "network"}, oldKey: "allowAllUnixSockets", newKey: "allow-all-unix-sockets"},
	{parent: []string{"sandbox", "agent", "targets", "*"}, oldKey: "authHeader", newKey: "auth-header"},
	{parent: []string{"sandbox", "agent", "targets", "*"}, oldKey: "extraHeaders", newKey: "extra-headers"},
	{parent: []string{"sandbox", "agent", "targets", "*"}, oldKey: "extraBodyFields", newKey: "extra-body-fields"},
	{parent: []string{"sandbox", "agent", "targets", "*"}, oldKey: "sessionId", newKey: "session-id"},
	{parent: []string{"sandbox", "config", "filesystem"}, oldKey: "denyRead", newKey: "deny-read"},
	{parent: []string{"sandbox", "config", "filesystem"}, oldKey: "allowWrite", newKey: "allow-write"},
	{parent: []string{"sandbox", "config", "filesystem"}, oldKey: "denyWrite", newKey: "deny-write"},
	{parent: []string{"sandbox", "config"}, oldKey: "ignoreViolations", newKey: "ignore-violations"},
	{parent: []string{"sandbox", "config"}, oldKey: "enableWeakerNestedSandbox", newKey: "enable-weaker-nested-sandbox"},
	{parent: []string{"sandbox", "config", "network"}, oldKey: "allowedDomains", newKey: "allowed-domains"},
	{parent: []string{"sandbox", "config", "network"}, oldKey: "blockedDomains", newKey: "blocked-domains"},
	{parent: []string{"sandbox", "config", "network"}, oldKey: "allowUnixSockets", newKey: "allow-unix-sockets"},
	{parent: []string{"sandbox", "config", "network"}, oldKey: "allowLocalBinding", newKey: "allow-local-binding"},
	{parent: []string{"sandbox", "config", "network"}, oldKey: "allowAllUnixSockets", newKey: "allow-all-unix-sockets"},
	{parent: []string{"sandbox", "mcp"}, oldKey: "entrypointArgs", newKey: "entrypoint-args"},
}

func getSandboxFieldsCodemod() Codemod {
	return Codemod{
		ID:           "sandbox-fields-kebab-case",
		Name:         "Normalize sandbox fields to kebab-case",
		Description:  "Renames camel-cased sandbox frontmatter fields to their kebab-case equivalents",
		IntroducedIn: "0.86.0",
		Apply: func(content string, _ map[string]any) (string, bool, error) {
			return codemod.Apply(content, func(f *codemod.Frontmatter) error {
				for _, rename := range sandboxFieldRenames {
					if len(rename.parent) < 1 {
						continue
					}
					parents := [][]string{rename.parent}
					if rename.parent[len(rename.parent)-1] == "*" { //nolint:uncheckedsliceindex // The nonempty parent guard above makes len(parent)-1 an in-bounds index.
						prefix := rename.parent[:len(rename.parent)-1]
						value, _ := f.Lookup(prefix...)
						if _, sequence := value.([]any); sequence {
							if err := renameSandboxSequenceTargetField(f, rename); err != nil {
								return err
							}
							continue
						}
						targets, ok := value.(map[string]any)
						if !ok {
							continue
						}
						names := make([]string, 0, len(targets))
						for name := range targets {
							names = append(names, name)
						}
						slices.Sort(names)
						parents = nil
						for _, name := range names {
							parents = append(parents, append(slices.Clone(prefix), name))
						}
					}

					for _, parent := range parents {
						path := append(slices.Clone(parent), rename.oldKey)
						if _, exists := f.Lookup(path...); !exists {
							continue
						}
						if _, exists := f.Lookup(append(slices.Clone(parent), rename.newKey)...); exists {
							return fmt.Errorf("cannot migrate sandbox field %q to %q at %s: both spellings are present; resolve the duplicate values manually and rerun gh aw fix", rename.oldKey, rename.newKey, strings.Join(path, "."))
						}
						if err := f.Rename(path, rename.newKey); err != nil {
							return err
						}
					}
				}
				return nil
			})
		},
	}
}

func renameSandboxSequenceTargetField(f *codemod.Frontmatter, rename sandboxFieldRename) error {
	return f.Edit(func(document *aw.Document) error {
		sandbox, ok := document.Frontmatter["sandbox"].(map[string]any)
		if !ok {
			return nil
		}
		agent, ok := sandbox["agent"].(map[string]any)
		if !ok {
			return nil
		}
		targets, ok := agent["targets"].([]any)
		if !ok {
			return nil
		}
		for index, value := range targets {
			target, ok := value.(map[string]any)
			if !ok {
				continue
			}
			value, exists := target[rename.oldKey]
			if !exists {
				continue
			}
			if _, exists := target[rename.newKey]; exists {
				return fmt.Errorf("cannot migrate sandbox field %q to %q at sandbox.agent.targets.%d.%s: both spellings are present; resolve the duplicate values manually and rerun gh aw fix", rename.oldKey, rename.newKey, index, rename.oldKey)
			}
			target[rename.newKey] = value
			delete(target, rename.oldKey)
		}
		return nil
	})
}
