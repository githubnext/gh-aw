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
      GH_AW_DYNAMIC_LFS: "false",
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
    expect(action.runs.steps.every(step => step.with.lfs === false)).toBe(true);
    expect(action.runs.steps.every(step => step.with.submodules === false)).toBe(true);
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

  it.each(["public", "private", "internal"])("rejects enabled submodules before authenticated reads in a %s host", async visibility => {
    runtime.github.request.mockResolvedValue({ data: metadata("host/workflows", visibility) });
    for (const submodules of ["true", "recursive", "${{ inputs.submodules }}"]) {
      vi.stubEnv("GH_AW_DYNAMIC_SUBMODULES", submodules);
      await expect(prepare(runtime)).rejects.toThrow("submodule URLs can bypass allowed-repos");
      expect(runtime.core.setOutput).toHaveBeenLastCalledWith("checkout_error", expect.stringContaining("submodule URLs can bypass allowed-repos"));
      expect(runtime.github.request).not.toHaveBeenCalled();
      expect(runtime.getOctokit).not.toHaveBeenCalled();
      expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
      expect(fs.existsSync(path.join(runnerTemp, "gh-aw"))).toBe(false);
    }
  });

  it.each(["public", "private", "internal"].flatMap(visibility => ["", "false"].map(submodules => [visibility, submodules])))("keeps submodules disabled in a %s host with setting %s", async (visibility, submodules) => {
    runtime.github.request.mockResolvedValue({ data: metadata("host/workflows", visibility) });
    vi.stubEnv("GH_AW_DYNAMIC_SUBMODULES", submodules);
    await prepare(runtime);
    const action = JSON.parse(fs.readFileSync(path.join(workspace, ".gh-aw/checkouts/0/action.yml"), "utf8"));
    expect(action.runs.steps.every(step => step.with.submodules === false)).toBe(true);
  });

  it.each(["public", "private", "internal"])("rejects LFS before authenticated reads in a %s host, with or without sparse checkout", async visibility => {
    runtime.github.request.mockResolvedValue({ data: metadata("host/workflows", visibility) });
    vi.stubEnv("GH_AW_DYNAMIC_LFS", "true");
    for (const sparse of ["", "assets"]) {
      vi.stubEnv("GH_AW_DYNAMIC_SPARSE", sparse);
      await expect(prepare(runtime)).rejects.toThrow("LFS endpoints can bypass allowed-repos");
      expect(runtime.core.setOutput).toHaveBeenLastCalledWith("checkout_error", expect.stringContaining("LFS endpoints can bypass allowed-repos"));
      expect(runtime.github.request).not.toHaveBeenCalled();
      expect(runtime.getOctokit).not.toHaveBeenCalled();
      expect(fs.existsSync(path.join(workspace, ".gh-aw"))).toBe(false);
      expect(fs.existsSync(path.join(runnerTemp, "gh-aw"))).toBe(false);
    }
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

  function installCredentialHelpers(cleaner) {
    const actions = path.join(runnerTemp, "gh-aw", "actions");
    fs.mkdirSync(actions, { recursive: true });
    for (const name of ["clean_git_credentials.sh", "verify_git_credentials.sh"]) {
      fs.copyFileSync(path.join(__dirname, "..", "sh", name), path.join(actions, name));
    }
    if (cleaner) fs.writeFileSync(path.join(actions, "clean_git_credentials.sh"), cleaner);
    // The production runner has GNU realpath; adapt existing paths for macOS tests.
    const bin = path.join(temp, "bin");
    fs.mkdirSync(bin);
    fs.writeFileSync(path.join(bin, "realpath"), '#!/usr/bin/env node\nprocess.stdout.write(require("fs").realpathSync(process.argv.at(-1)) + "\\n");\n', { mode: 0o755 });
    vi.stubEnv("PATH", `${bin}${path.delimiter}${process.env.PATH}`);
    vi.stubEnv("GIT_CONFIG_GLOBAL", "/dev/null");
    vi.stubEnv("GIT_CONFIG_NOSYSTEM", "1");
  }

  it("cleans and verifies real persisted credential includes beyond discovery depth, including nested submodules", async () => {
    const prefix = Array(12).fill("nested").join("/");
    vi.stubEnv("GH_AW_DYNAMIC_PREFIX", prefix);
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    vi.stubEnv("GH_AW_DYNAMIC_PERSIST", "true");
    const [entry] = await prepare(runtime);
    installCredentialHelpers();
    const target = path.join(workspace, entry.path);
    execFileSync("git", ["init", "--quiet", target], { stdio: ["ignore", "pipe", "pipe"] });
    const gitDir = path.join(target, ".git");
    const module = path.join(gitDir, "modules", ...Array(8).fill("nested"), "child");
    execFileSync("git", ["init", "--bare", "--quiet", module], { stdio: ["ignore", "pipe", "pipe"] });
    const includes = [];
    for (const [index, dir] of [gitDir, module].entries()) {
      const include = path.join(fs.realpathSync(runnerTemp), `git-credentials-${index}.config`);
      includes.push(include);
      fs.writeFileSync(include, "[http]\nextraheader = Authorization: test-only-checkout-token\n");
      const config = path.join(dir, "config");
      execFileSync("git", ["config", "--file", config, `includeIf.gitdir:${fs.realpathSync(dir)}.path`, include]);
      expect(fs.readFileSync(config, "utf8")).toContain("git-credentials-");
    }
    expect(path.relative(workspace, path.join(gitDir, "config")).split(path.sep)).toHaveLength(16);
    await finish({ cleanCredentials: true });
    for (const dir of [gitDir, module]) {
      expect(fs.readFileSync(path.join(dir, "config"), "utf8")).not.toContain("git-credentials-");
    }
    for (const include of includes) expect(fs.existsSync(include)).toBe(false);
    expect(fs.existsSync(path.join(workspace, ".gh-aw/checkouts/0"))).toBe(false);
  }, 30000);

  it.each(["exit 7", "exit 0"])("fails closed when explicit cleanup returns '%s' but credentials remain", async cleaner => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_COUNT", "1");
    vi.stubEnv("GH_AW_DYNAMIC_FETCH_0", "main");
    const [entry] = await prepare(runtime);
    installCredentialHelpers(`#!/usr/bin/env bash\n${cleaner}\n`);
    const target = path.join(workspace, entry.path);
    execFileSync("git", ["init", "--quiet", target], { stdio: ["ignore", "pipe", "pipe"] });
    execFileSync("git", ["-C", target, "config", "http.extraheader", "Authorization: test-only-checkout-token"]);
    const runGit = vi.fn();
    await expect(finish({ cleanCredentials: true, runGit })).rejects.toThrow("Unable to clean and verify runtime checkout credentials; refusing agent execution");
    expect(runGit).not.toHaveBeenCalled();
    expect(fs.existsSync(path.join(workspace, ".gh-aw/checkouts/0/action.yml"))).toBe(true);
  });

  it("fails closed when an expected checkout is missing even if filesystem discovery would find no configs", async () => {
    await prepare(runtime);
    await expect(finish({ cleanCredentials: true })).rejects.toThrow("Unable to clean and verify runtime checkout credentials; refusing agent execution");
  });

  it("fails closed when the trusted cleanup helpers are unavailable", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    const [entry] = await prepare(runtime);
    execFileSync("git", ["init", "--quiet", path.join(workspace, entry.path)], { stdio: ["ignore", "pipe", "pipe"] });
    await expect(finish({ cleanCredentials: true })).rejects.toThrow("Unable to clean and verify runtime checkout credentials; refusing agent execution");
  });

  it.each(["missing-config", "symlink-config", "symlink-modules"])("fails closed for an unverifiable checkout: %s", async invalid => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    const [entry] = await prepare(runtime);
    const target = path.join(workspace, entry.path);
    execFileSync("git", ["init", "--quiet", target], { stdio: ["ignore", "pipe", "pipe"] });
    const config = path.join(target, ".git", "config");
    if (invalid === "symlink-modules") {
      fs.symlinkSync(path.join(temp, "missing"), path.join(target, ".git", "modules"));
    } else {
      fs.unlinkSync(config);
      if (invalid === "symlink-config") {
        const other = path.join(temp, "config");
        fs.writeFileSync(other, "[http]\nextraheader = Authorization: test-only-checkout-token\n");
        fs.symlinkSync(other, config);
      }
    }
    await expect(finish({ cleanCredentials: true })).rejects.toThrow("Unable to clean and verify runtime checkout credentials; refusing agent execution");
    expect(fs.existsSync(path.join(workspace, ".gh-aw/checkouts/0/action.yml"))).toBe(true);
  });

  it("preserves persisted credentials when finalizing a safe-output checkout", async () => {
    vi.stubEnv("GH_AW_DYNAMIC_REPOSITORIES", "org/a");
    vi.stubEnv("GH_AW_DYNAMIC_PERSIST", "true");
    const [entry] = await prepare(runtime);
    const target = path.join(workspace, entry.path);
    execFileSync("git", ["init", "--quiet", target], { stdio: ["ignore", "pipe", "pipe"] });
    execFileSync("git", ["-C", target, "config", "http.extraheader", "Authorization: test-only-checkout-token"]);
    await finish({ cleanCredentials: false });
    expect(fs.readFileSync(path.join(target, ".git/config"), "utf8")).toContain("test-only-checkout-token");
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
