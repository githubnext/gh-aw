//go:build !integration

package workflow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRestoreGitHistory(t *testing.T) {
	for _, scenario := range []string{
		"newer trusted base", "unchanged trusted base", "sparse trusted base",
		"event default branch", "PR descendant", "feature named main",
		"spoofed remote ref", "reserved pr-head", "missing ref",
		"missing default branch", "nested workspace", "no repository",
		"missing trusted blob",
	} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			workspace := filepath.Join(root, "workspace")
			require.NoError(t, os.MkdirAll(workspace, 0755))
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = workspace
				cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%v: %s", args, output)
				return strings.TrimSpace(string(output))
			}
			write := func(name, content string) {
				t.Helper()
				file := filepath.Join(workspace, name)
				require.NoError(t, os.MkdirAll(filepath.Dir(file), 0755))
				require.NoError(t, os.WriteFile(file, []byte(content), 0644))
			}
			git("init", "-b", "main")
			git("config", "user.name", "Test")
			git("config", "user.email", "test@example.com")
			write(".github/agents/edited.md", "activation")
			write(".github/agents/deleted.md", "activation")
			write(".claude/deleted.md", "activation")
			write("AGENTS.md", "activation")
			write("CLAUDE.md", "activation")
			write(".gitattributes", ".github export-ignore\n")
			write("src/output.txt", "before")
			git("add", ".")
			git("commit", "-m", "Activation A")
			activationSHA := git("rev-parse", "HEAD")
			base := filepath.Join(root, "base")
			ambient := filepath.Join(root, "ambient-folders")
			data := &WorkflowData{
				EngineConfig:   &EngineConfig{ID: "claude"},
				AmbientFolders: []string{".github/agents", ".claude"},
			}
			folders, files := resolveAgentManifestPaths(NewEngineRegistry(), data)
			files = append(files, "GEMINI.md")
			env := append(os.Environ(),
				"GITHUB_WORKSPACE="+workspace,
				"GH_AW_AGENT_FOLDERS="+strings.Join(folders, " "),
				"GH_AW_AGENT_FILES="+strings.Join(files, " "),
				"GH_AW_AMBIENT_FOLDERS="+strings.Join(data.AmbientFolders, " "),
				"DEFAULT_BRANCH=main", "GITHUB_EVENT_PATH=",
			)
			runShell := func(script string) string {
				t.Helper()
				script = strings.ReplaceAll(script, "/tmp/gh-aw/base", base)
				script = strings.ReplaceAll(script, "/tmp/gh-aw/ambient-folders", ambient)
				cmd := exec.Command("bash", "-e", "-c", script)
				cmd.Dir, cmd.Env = workspace, env
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
				return string(output)
			}
			readScript := func(name string) string {
				t.Helper()
				content, err := os.ReadFile(filepath.Join("..", "..", "actions", "setup", "sh", name))
				require.NoError(t, err)
				return string(content)
			}
			runShell(readScript("save_base_github_folders.sh"))
			_, stage, found := strings.Cut(strings.Join(generateStageAmbientFoldersStep(data), ""), "        run: |\n")
			require.True(t, found)
			runShell(stage)

			trusted := scenario == "newer trusted base" || scenario == "unchanged trusted base" ||
				scenario == "sparse trusted base" || scenario == "event default branch"
			if scenario != "unchanged trusted base" {
				write(".github/agents/edited.md", "new base")
				write(".github/agents/added.md", "new base")
				write(".agents/added.md", "new base")
				write("AGENTS.md", "new base")
				write("GEMINI.md", "new base")
				git("rm", ".github/agents/deleted.md", ".claude/deleted.md", "CLAUDE.md")
				git("add", ".")
				git("commit", "-m", "Concurrent base B")
			}
			baseSHA := git("rev-parse", "HEAD")
			git("update-ref", "refs/remotes/origin/main", baseSHA)
			switch scenario {
			case "PR descendant", "feature named main", "spoofed remote ref", "reserved pr-head":
				write(".github/agents/edited.md", "untrusted")
				write(".github/injected.md", "untrusted")
				write("AGENTS.md", "untrusted")
				git("add", ".")
				git("commit", "-m", "Untrusted head")
				if scenario == "spoofed remote ref" {
					git("checkout", "-b", "refs/remotes/origin/main")
					git("update-ref", "-d", "refs/remotes/origin/main")
				}
				if scenario == "reserved pr-head" {
					env = append(env, "DEFAULT_BRANCH=pr-head")
					git("update-ref", "refs/remotes/origin/pr-head", "HEAD")
				}
			case "missing ref":
				git("update-ref", "-d", "refs/remotes/origin/main")
			case "missing default branch":
				env = append(env, "DEFAULT_BRANCH=")
			case "event default branch":
				event := filepath.Join(root, "event.json")
				require.NoError(t, os.WriteFile(event, []byte(`{"repository":{"default_branch":"main"}}`), 0644))
				env = append(env, "DEFAULT_BRANCH=", "GITHUB_EVENT_PATH="+event)
			case "sparse trusted base":
				git("sparse-checkout", "set", "src")
				require.NoError(t, os.Remove(filepath.Join(workspace, "AGENTS.md")))
			case "nested workspace":
				workspace = filepath.Join(workspace, "nested")
				require.NoError(t, os.MkdirAll(workspace, 0755))
				env = append(env, "GITHUB_WORKSPACE="+workspace)
			case "no repository":
				workspace = filepath.Join(root, "no-repository")
				require.NoError(t, os.MkdirAll(workspace, 0755))
				env = append(env, "GITHUB_WORKSPACE="+workspace)
			}
			write(".mcp.json", `{"mcpServers":{"untrusted":{}}}`)

			// Exercise the emitted ambient -> config -> ambient sequence, not just
			// the config script in isolation. The early copy deliberately dirties B.
			var yaml strings.Builder
			generateRestoreAmbientFoldersStep(&yaml, data)
			_, restoreAmbient, found := strings.Cut(yaml.String(), "        run: |\n")
			require.True(t, found)
			runShell(restoreAmbient)
			if scenario == "missing trusted blob" {
				blob := git("rev-parse", "HEAD:.github/agents/edited.md")
				object := filepath.Join(workspace, ".git", "objects", blob[:2], blob[2:])
				require.NoError(t, os.Remove(object))
				script := strings.ReplaceAll(readScript("restore_base_github_folders.sh"), "/tmp/gh-aw/ambient-folders", ambient)
				script = strings.ReplaceAll(script, "/tmp/gh-aw/base", base)
				cmd := exec.Command("bash", "-e", "-c", script)
				cmd.Dir, cmd.Env = workspace, env
				output, err := cmd.CombinedOutput()
				require.Error(t, err)
				assert.Contains(t, string(output), "::error::Cannot materialize trusted config")
				return
			}
			output := runShell(readScript("restore_base_github_folders.sh"))
			runShell(restoreAmbient)
			assert.NoFileExists(t, filepath.Join(workspace, ".mcp.json"))
			assert.NoFileExists(t, filepath.Join(workspace, ".github/injected.md"))
			expected := "activation"
			if trusted && scenario != "unchanged trusted base" {
				expected = "new base"
			}
			content, err := os.ReadFile(filepath.Join(workspace, ".github/agents/edited.md"))
			require.NoError(t, err)
			assert.Equal(t, expected, string(content))
			content, err = os.ReadFile(filepath.Join(workspace, "AGENTS.md"))
			require.NoError(t, err)
			assert.Equal(t, expected, string(content))
			if trusted && scenario != "unchanged trusted base" {
				assert.FileExists(t, filepath.Join(workspace, ".github/agents/added.md"))
				assert.FileExists(t, filepath.Join(workspace, ".agents/added.md"))
				assert.FileExists(t, filepath.Join(workspace, "GEMINI.md"))
				assert.NoFileExists(t, filepath.Join(workspace, ".github/agents/deleted.md"))
				assert.NoDirExists(t, filepath.Join(workspace, ".claude"))
				assert.NoFileExists(t, filepath.Join(workspace, "CLAUDE.md"))
			} else {
				assert.NoFileExists(t, filepath.Join(workspace, ".github/agents/added.md"))
				assert.NoDirExists(t, filepath.Join(workspace, ".agents"))
				assert.NoFileExists(t, filepath.Join(workspace, "GEMINI.md"))
				assert.FileExists(t, filepath.Join(workspace, ".github/agents/deleted.md"))
				assert.FileExists(t, filepath.Join(workspace, ".claude/deleted.md"))
				assert.FileExists(t, filepath.Join(workspace, "CLAUDE.md"))
			}
			if !trusted {
				assert.Contains(t, output, "restoring activation snapshot")
				return
			}
			assert.Contains(t, output, "verified default-branch checkout "+baseSHA)
			assert.Empty(t, git("status", "--porcelain"), "restoration must not roll back B")
			git("checkout", "-b", "agent-change")
			write("src/output.txt", "agent change")
			git("add", "-A")
			git("commit", "-m", "Allowed agent change")
			assert.Equal(t, "src/output.txt", git("diff", "--name-only", baseSHA+"...HEAD"))
			assert.Equal(t, baseSHA, git("rev-parse", "HEAD^"))

			// Run production patch generation and both file-protection guards.
			// Apply that exact patch on B to measure the final PR diff as well.
			jsDir, err := filepath.Abs(filepath.Join("..", "..", "actions", "setup", "js"))
			require.NoError(t, err)
			node := exec.Command("node", "-e", `
const fs = require("fs");
const path = require("path");
const { generateGitPatch } = require(path.join(process.argv[1], "generate_git_patch.cjs"));
const { checkFileProtection, checkFileProtectionPostApply } = require(path.join(process.argv[1], "manifest_file_helpers.cjs"));
const { execFileSync } = require("child_process");
global.core = { debug() {}, info() {}, warning() {} };
(async () => {
  const result = await generateGitPatch("agent-change", "main", { repoSlug: process.argv[2] });
  if (!result.success) throw new Error(JSON.stringify(result));
  try {
    const patch = fs.readFileSync(result.patchPath, "utf8");
    const config = { allowed_files: ["src/output.txt"], protect_top_level_dot_folders: true };
    if (checkFileProtection(patch, config).action !== "allow") throw new Error("Patch exceeds allowed_files");
    const git = (...args) => execFileSync("git", args, { encoding: "utf8" }).trim();
    git("checkout", "-b", "final-pr", process.argv[3]);
    git("am", result.patchPath);
    const files = git("diff", "--name-only", process.argv[3] + "...HEAD").split("\n");
    if (JSON.stringify(files) !== JSON.stringify(["src/output.txt"])) throw new Error("Unexpected final PR diff: " + files);
    if (checkFileProtectionPostApply(files, config).action !== "allow") throw new Error("Final diff exceeds allowed_files");
  } finally {
    fs.unlinkSync(result.patchPath);
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
`, jsDir, filepath.Base(filepath.Dir(root)), baseSHA)
			node.Dir = workspace
			node.Env = append(env, "GITHUB_SHA="+activationSHA)
			nodeOutput, err := node.CombinedOutput()
			require.NoError(t, err, "%s", nodeOutput)
		})
	}
}
