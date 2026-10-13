import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { execFileSync } from "child_process";
import { prepare, finish, parseRepositories, parseAllowedRepos, workspacePath } from "./checkout_dynamic.cjs";
import { buildCheckoutManifest, readManifestEntriesFromEnv } from "./build_checkout_manifest.cjs";

describe("runtime repository checkout", () => {
  let temp;
  let workspace;
  let runnerTemp;
  let runtime;
  const metadata = (repository, visibility = "public") => ({
    full_name: repository,
    visibility,
    private: visibility !== "public",
    default_branch: "main",
  });

  beforeEach(() => {
    temp = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-checkout-"));
    workspace = path.join(temp, "workspace");
    runnerTemp = path.join(temp, "temp");
    fs.mkdirSync(workspace);
    fs.mkdirSync(runnerTemp);
    for (const [key, value] of Object.entries({
      GITHUB_WORKSPACE: workspace,
      RUNNER_TEMP: runnerTemp,
      GITHUB_REPOSITORY: "host/workflows",
      GITHUB_SERVER_URL: "https://github.com",
      GH_AW_DYNAMIC_CHECKOUT_INDEX: "0",
      GH_AW_DYNAMIC_PREFIX: "targets",
      GH_AW_DYNAMIC_REPOSITORIES: "org/a\r\norg/b\n",
      GH_AW_DYNAMIC_ALLOWED_REPOS: "org/*",
      GH_AW_DYNAMIC_TOKEN: "secret-token",
      GH_AW_DYNAMIC_CHECKOUT_ACTION: `actions/checkout@${"a".repeat(40)}`,
      GH_AW_DYNAMIC_SUBMODULES: "false",
      GH_AW_DYNAMIC_PERSIST: "false",
      GH_AW_DYNAMIC_FETCH_COUNT: "0",
      GH_AW_DYNAMIC_SPARSE: "",
    }))
      vi.stubEnv(key, value);
    runtime = {
      core: { setSecret: vi.fn(), setOutput: vi.fn() },
      github: { request: vi.fn().mockResolvedValue({ data: metadata("host/workflows") }) },
      getOctokit: vi.fn().mockReturnValue({
        request: vi.fn().mockImplementation(async (_route, { owner, repo }) => ({ data: metadata(`${owner}/${repo}`) })),
      }),
    };
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    fs.rmSync(temp, { recursive: true, force: true });
  });

  it("expands and deduplicates newline-separated repositories into pinned, credential-free checkouts", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a\nORG/A\norg/b\n");
    const entries = await prepare(runtime);
    expect(entries).toEqual([
      { repository: "org/a", path: "targets/org/a", default_branch: "main" },
      { repository: "org/b", path: "targets/org/b", default_branch: "main" },
    ]);
    expect(runtime.getOctokit).toHaveBeenCalledWith("secret-token");
    const action = JSON.parse(fs.readFileSync(path.join(workspace, ".gh-aw/checkouts/0/action.yml"), "utf8"));
    expect(action.runs.steps).toHaveLength(2);
    expect(action.runs.steps[0].uses).toBe(`actions/checkout@${"a".repeat(40)}`);
    expect(action.runs.steps[0].with.repository).toBe("org/a");
    expect(action.runs.steps[0].with["persist-credentials"]).toBe("${{ env.GH_AW_DYNAMIC_PERSIST }}");
    expect(JSON.stringify(action)).not.toContain("secret-token");
    expect(JSON.stringify(entries)).not.toContain("secret-token");
  });

  it("handles an empty scheduled batch without expanding to all allowed repositories", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "\n");
    expect(await prepare(runtime)).toEqual([]);
    const action = JSON.parse(fs.readFileSync(path.join(workspace, ".gh-aw/checkouts/0/action.yml"), "utf8"));
    expect(action.runs.steps).toEqual([{ shell: "bash", run: "true" }]);
  });

  it.each(["", "  \n", "org", "https://github.com/org/a", "*", " * ", "*/*", "**/**", "org/*\n*", "org/*,*"])("rejects invalid or missing allowed-repos: %s", async allow => {
    vi.stubEnv("GH_AW_DYNAMIC_ALLOWED_REPOS", allow);
    await expect(prepare(runtime)).rejects.toThrow(/allowed-repos/);
    expect(runtime.getOctokit).not.toHaveBeenCalled();
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", expect.stringContaining("allowed-repos"));
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it("checks exact and wildcard patterns case-insensitively and denies unmatched repositories", async () => {
    expect(parseAllowedRepos("ORG/a\nother/*,*/c")).toEqual(new Set(["org/a", "other/*", "*/c"]));
    vi.stubEnv("GH_AW_DYNAMIC_ALLOWED_REPOS", "org/a");
    await expect(prepare(runtime)).rejects.toThrow("org/b is not permitted");
    expect(runtime.getOctokit).not.toHaveBeenCalled();
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", "Runtime checkout repository org/b is not permitted by allowed-repos");
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it.each(["org/*", "*/a", "org/a*"])("uses allowed-repos pattern %s only to filter explicit selections", async pattern => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    vi.stubEnv("GH_AW_DYNAMIC_ALLOWED_REPOS", pattern);
    expect(await prepare(runtime)).toEqual([{ repository: "org/a", path: "targets/org/a", default_branch: "main" }]);
  });

  it.each(["org/a\n../secret", "org/a;echo pwned", "${{ secrets.TOKEN }}", "org/..", "org/a b", "org/a/b"])("rejects malformed repository data without executing it: %s", async value => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", value);
    await expect(prepare(runtime)).rejects.toThrow(/owner\/repo/);
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", expect.stringContaining("owner/repo"));
    expect(runtime.getOctokit).not.toHaveBeenCalled();
  });

  it.each(["*", "org/*", "*/a", "org/a*", "org/a?", "org/[ab]", "org/a\norg/*"])("rejects repository wildcards instead of expanding them: %s", async value => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", value);
    await expect(prepare(runtime)).rejects.toThrow("wildcards are only supported in checkout.allowed-repos");
    expect(runtime.getOctokit).not.toHaveBeenCalled();
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", expect.stringContaining("wildcards are only supported in checkout.allowed-repos"));
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it.each(["private", "internal"])("blocks %s repositories in a public host before creating any checkout action", async visibility => {
    runtime.getOctokit().request.mockImplementation(async (_route, { owner, repo }) => ({
      data: metadata(`${owner}/${repo}`, repo === "b" ? visibility : "public"),
    }));
    await expect(prepare(runtime)).rejects.toThrow("Public workflows cannot check out non-public repository org/b");
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", "Public workflows cannot check out non-public repository org/b");
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it("allows private repositories in private hosts with the entry's own token", async () => {
    runtime.github.request.mockResolvedValue({ data: metadata("host/workflows", "private") });
    runtime.getOctokit().request.mockResolvedValue({ data: metadata("org/a", "private") });
    expect(await prepare(runtime)).toHaveLength(1);
  });

  it("fails closed if host or target visibility cannot be verified", async () => {
    runtime.github.request.mockRejectedValue(new Error("secret-token"));
    await expect(prepare(runtime)).rejects.toThrow("Unable to verify workflow repository visibility");
    expect(runtime.core.setOutput).toHaveBeenLastCalledWith("checkout_error", "Unable to verify workflow repository visibility; runtime checkout refused");
    runtime.github.request.mockResolvedValue({ data: metadata("host/workflows") });
    runtime.getOctokit().request.mockRejectedValue(new Error("secret-token"));
    await expect(prepare(runtime)).rejects.toThrow("Unable to verify runtime checkout repository org/a");
    expect(runtime.core.setOutput).toHaveBeenLastCalledWith("checkout_error", "Unable to verify runtime checkout repository org/a; checkout refused");
    runtime.getOctokit().request.mockResolvedValue({ data: { full_name: "org/a", default_branch: "main" } });
    await expect(prepare(runtime)).rejects.toThrow("Unable to verify allowlist and metadata");
    expect(JSON.stringify(runtime.core.setOutput.mock.calls)).not.toContain("secret-token");
  });

  it("reports unexpected preparation failures without publishing dependency error details or swallowing the error", async () => {
    const error = new Error("dependency diagnostic containing secret-token");
    runtime.getOctokit.mockImplementation(() => {
      throw error;
    });
    await expect(prepare(runtime)).rejects.toBe(error);
    expect(runtime.core.setSecret).toHaveBeenCalledWith("secret-token");
    expect(runtime.core.setOutput).toHaveBeenCalledExactlyOnceWith("checkout_error", "Runtime checkout preparation failed; inspect the failed step for details");
  });

  it("reports missing checkout credentials before contacting GitHub", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_TOKEN", "");
    await expect(prepare(runtime)).rejects.toThrow("Runtime checkout requires a GitHub token");
    expect(runtime.core.setOutput).toHaveBeenCalledExactlyOnceWith("checkout_error", "Runtime checkout requires a GitHub token");
    expect(runtime.getOctokit).not.toHaveBeenCalled();
    expect(runtime.github.request).not.toHaveBeenCalled();
  });

  it("revalidates the canonical repository after redirects", async () => {
    runtime.getOctokit().request.mockResolvedValue({ data: metadata("outside/a") });
    await expect(prepare(runtime)).rejects.toThrow("Unable to verify allowlist");
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", "Unable to verify allowlist and metadata for runtime checkout repository org/a");
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it("fails closed when workflow repository visibility metadata is incomplete", async () => {
    runtime.github.request.mockResolvedValue({ data: { private: false } });
    await expect(prepare(runtime)).rejects.toThrow("Unable to verify workflow repository visibility");
    expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
  });

  it("blocks submodule checkout in public hosts", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_SUBMODULES", "recursive");
    await expect(prepare(runtime)).rejects.toThrow("cannot enable submodules");
  });

  it.each(["../outside", "/absolute", ".git/targets", ".gh-aw/targets", "${{ inputs.path }}"])("rejects unsafe path prefixes: %s", async prefix => {
    vi.stubEnv("GH_AW_DYNAMIC_PREFIX", prefix);
    await expect(prepare(runtime)).rejects.toThrow(/path|prefix/);
    expect(runtime.core.setOutput).toHaveBeenCalledWith("checkout_error", expect.stringMatching(/path|prefix/));
    expect(runtime.getOctokit).not.toHaveBeenCalled();
  });

  it("rejects symlinked ancestors, including dangling links", async () => {
    fs.symlinkSync(path.join(temp, "absent"), path.join(workspace, "targets"));
    await expect(prepare(runtime)).rejects.toThrow("symlinks");
    fs.unlinkSync(path.join(workspace, "targets"));
    fs.symlinkSync(runnerTemp, path.join(workspace, ".gh-aw"));
    await expect(prepare(runtime)).rejects.toThrow("symlinks");
  });

  it("does not let actions/checkout clean an existing checkout destination", async () => {
    fs.mkdirSync(path.join(workspace, "targets/org/a"), { recursive: true });
    await expect(prepare(runtime)).rejects.toThrow("destination already exists");
  });

  it("requires SHA-pinned checkout actions", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_CHECKOUT_ACTION", "actions/checkout@v7");
    await expect(prepare(runtime)).rejects.toThrow("SHA-pinned");
  });

  it("records all runtime entries and their API-resolved default branches in the shared manifest", async () => {
    await prepare(runtime);
    vi.stubEnv("GH_AW_DYNAMIC_CHECKOUT_INDEXES", "0");
    vi.stubEnv("GH_AW_CHECKOUT_MANIFEST_COUNT", "0");
    const entries = readManifestEntriesFromEnv();
    expect(entries).toHaveLength(2);
    const runGH = vi.fn();
    const result = buildCheckoutManifest(entries, { runnerTemp, workspace, runGH });
    expect(result.manifest["org/b"]).toEqual({ repository: "org/b", path: "targets/org/b", default_branch: "main" });
    expect(runGH).not.toHaveBeenCalled();
  });

  it("rejects a missing runtime checkout plan instead of writing an incomplete manifest", () => {
    vi.stubEnv("GH_AW_DYNAMIC_CHECKOUT_INDEXES", "0");
    vi.stubEnv("GH_AW_CHECKOUT_MANIFEST_COUNT", "0");
    expect(() => readManifestEntriesFromEnv()).toThrow("Unable to read runtime checkouts");
  });

  it("fetches additional refs with transient per-checkout credentials and never places the token in arguments", async () => {
    await prepare(runtime);
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_COUNT", "3");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_0", "main");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_1", "*");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_2", "refs/pulls/open/*");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_DEPTH", "0");
    const runGit = vi.fn().mockReturnValue("");
    await finish({ runGit });
    expect(runGit).toHaveBeenCalledTimes(2);
    const [args, options] = runGit.mock.calls[0];
    expect(args).toEqual([
      "-C",
      path.join(fs.realpathSync(workspace), "targets/org/a"),
      "fetch",
      "origin",
      "+refs/heads/main:refs/remotes/origin/main",
      "+refs/heads/*:refs/remotes/origin/*",
      "+refs/pull/*/head:refs/remotes/origin/pull/*/head",
    ]);
    expect(JSON.stringify(args)).not.toContain("secret-token");
    expect(options.env.GIT_CONFIG_VALUE_0).toBe("");
    expect(options.env.GIT_CONFIG_VALUE_1).toContain(Buffer.from("x-access-token:secret-token").toString("base64"));
    expect(fs.existsSync(path.join(workspace, ".gh-aw/checkouts/0"))).toBe(false);
    runGit.mockImplementation(() => {
      throw new Error("secret-token");
    });
    await expect(finish({ runGit })).rejects.toThrow("Unable to finalize runtime checkout of org/a");
  });

  it("does not accept root paths or path traversal", () => {
    expect(() => workspacePath(workspace, ".")).toThrow();
    expect(() => workspacePath(workspace, "targets/../other")).toThrow();
    expect(parseRepositories("org/a\nORG/A")).toEqual(["org/a"]);
  });

  it("fetches real local git history and removes sparse partial-clone markers", async () => {
    vi.stubEnv("GIT_CONFIG_GLOBAL", "/dev/null");
    vi.stubEnv("GIT_CONFIG_NOSYSTEM", "1");
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    await prepare(runtime);
    const source = path.join(temp, "source");
    fs.mkdirSync(source);
    const git = (cwd, args) => execFileSync("git", ["-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "-C", cwd, ...args], { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
    git(source, ["init", "--initial-branch=main"]);
    git(source, ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "Initial"]);
    git(source, ["checkout", "-b", "feature"]);
    git(source, ["-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "Feature"]);
    git(source, ["checkout", "main"]);
    const target = path.join(workspace, "targets/org/a");
    fs.mkdirSync(path.dirname(target), { recursive: true });
    execFileSync("git", ["clone", `file://${source}`, target], { stdio: ["ignore", "pipe", "pipe"] });
    git(target, ["config", "remote.origin.promisor", "true"]);
    git(target, ["config", "remote.origin.partialclonefilter", "blob:none"]);
    vi.stubEnv("GH_AW_DYNAMIC_SPARSE", "src");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_COUNT", "1");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_0", "feature");
    await finish();
    expect(git(target, ["rev-parse", "origin/feature"])).toBe(git(source, ["rev-parse", "feature"]));
    expect(git(target, ["config", "--local", "--list"])).not.toMatch(/promisor|partialclonefilter|extraheader|secret-token/);
    expect(fs.existsSync(path.join(workspace, ".gh-aw/checkouts/0"))).toBe(false);
  }, 30000);
});
