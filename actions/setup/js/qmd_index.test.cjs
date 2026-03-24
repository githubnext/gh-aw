// @ts-check
/// <reference types="vitest/globals" />
import { describe, it, expect, beforeEach, afterEach, beforeAll, afterAll, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
import { createRequire } from "module";

// --- Fake @tobilu/qmd SDK setup -----------------------------------------------
//
// qmd_index.cjs dynamically imports the SDK via:
//   await import(pathToFileURL(path.join(__dirname, "node_modules/@tobilu/qmd/dist/index.js")))
//
// When the real package is not installed we create a minimal fake ESM module at
// that path so the dynamic import succeeds.  The fake createStore() returns
// globalThis.__qmdMockStore__, which is set fresh in beforeEach.
//
// Node's ES-module cache means the import() is only evaluated once across all
// tests.  Updating globalThis.__qmdMockStore__ between tests is therefore the
// mechanism for giving each test a fresh mock store.
//
// If @tobilu/qmd is already installed (e.g. in a dedicated CI integration job)
// the fake is not created; the real SDK's createStore() would be called, but
// because this scenario is only encountered in a CI job that specifically
// installs the package, both the unit tests (fake SDK) and the integration CI
// job (real SDK with a minimal fixture) are covered.

const SCRIPT_DIR = import.meta.dirname;
const SDK_DIST_DIR = path.join(SCRIPT_DIR, "node_modules", "@tobilu", "qmd", "dist");
const SDK_PATH = path.join(SDK_DIST_DIR, "index.js");
const SDK_PKG_PATH = path.join(SCRIPT_DIR, "node_modules", "@tobilu", "qmd", "package.json");

const sdkAlreadyInstalled = fs.existsSync(SDK_PATH);

// Minimal ESM module that proxies through the per-test mock store global.
const FAKE_SDK_ESM = `export async function createStore() {
  return globalThis.__qmdMockStore__;
}
`;
const FAKE_SDK_PKG = JSON.stringify({ type: "module", main: "dist/index.js" });

// --- Load module under test ---------------------------------------------------
//
// Load once; globals (core, github) and process.env are read at call time so
// changing them in beforeEach / afterEach affects each test independently.

const _require = createRequire(import.meta.url);
const { main } = _require("./qmd_index.cjs");

// --- Helpers ------------------------------------------------------------------

/** Creates a fresh mock store returned by the fake createStore(). */
function makeMockStore() {
  return {
    update: vi.fn().mockResolvedValue({ indexed: 2, updated: 0, unchanged: 0, removed: 0 }),
    embed: vi.fn().mockResolvedValue({ embedded: 2 }),
    close: vi.fn().mockResolvedValue(undefined),
  };
}

// --- Test suite ---------------------------------------------------------------

describe("qmd_index.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockStore;
  let tmpDir;

  // ── Global setup: create fake SDK if needed ───────────────────────────────
  beforeAll(() => {
    if (!sdkAlreadyInstalled) {
      // { recursive: true } creates all parent directories (including node_modules)
      // so this works even in a fresh clone before npm install.
      fs.mkdirSync(SDK_DIST_DIR, { recursive: true });
      fs.writeFileSync(SDK_PATH, FAKE_SDK_ESM, "utf8");
      fs.writeFileSync(SDK_PKG_PATH, FAKE_SDK_PKG, "utf8");
    }
  });

  afterAll(() => {
    if (!sdkAlreadyInstalled) {
      const tobiluScope = path.join(SCRIPT_DIR, "node_modules", "@tobilu");
      if (fs.existsSync(tobiluScope)) {
        fs.rmSync(tobiluScope, { recursive: true, force: true });
      }
    }
  });

  // ── Per-test setup ────────────────────────────────────────────────────────
  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "qmd-test-"));

    mockStore = makeMockStore();
    globalThis.__qmdMockStore__ = mockStore;

    mockCore = {
      info: vi.fn(),
      debug: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    mockGithub = {
      rest: {
        issues: { listForRepo: vi.fn() },
        search: {
          code: vi.fn().mockResolvedValue({ data: { items: [] } }),
        },
        repos: { getContent: vi.fn() },
      },
      paginate: vi.fn().mockResolvedValue([]),
    };

    global.core = mockCore;
    global.github = mockGithub;
    delete process.env.QMD_CONFIG_JSON;
    delete process.env.GITHUB_REPOSITORY;
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
    // Clean up qmd search dirs written to /tmp/gh-aw/qmd-search-N by the script.
    // These paths are hardcoded in qmd_index.cjs (Linux-specific, mirrors the
    // GitHub Actions runner environment).  We clean indices 0-9 which covers
    // all configs tested here (none use more than 2 search entries).
    for (let i = 0; i < 10; i++) {
      const d = `/tmp/gh-aw/qmd-search-${i}`;
      if (fs.existsSync(d)) fs.rmSync(d, { recursive: true, force: true });
    }
    delete globalThis.__qmdMockStore__;
    delete global.core;
    delete global.github;
    vi.restoreAllMocks();
  });

  // ── Helper ────────────────────────────────────────────────────────────────
  /**
   * Sets QMD_CONFIG_JSON and invokes main().
   * @param {object | undefined} config  Pass undefined to leave the env var unset.
   */
  async function runMain(config) {
    if (config !== undefined) {
      process.env.QMD_CONFIG_JSON = JSON.stringify(config);
    }
    await main();
  }

  // ── Error path: missing config ─────────────────────────────────────────────
  it("fails when QMD_CONFIG_JSON is not set", async () => {
    await runMain(undefined);
    expect(mockCore.setFailed).toHaveBeenCalledWith("QMD_CONFIG_JSON environment variable not set");
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Error path: SDK not installed ─────────────────────────────────────────
  it("fails when @tobilu/qmd SDK is not found", async () => {
    const realExistsSync = fs.existsSync.bind(fs);
    vi.spyOn(fs, "existsSync").mockImplementation(p => {
      // Use exact path comparison (same value the script computes) rather than
      // substring search to avoid accidentally suppressing unrelated lookups.
      if (path.normalize(String(p)) === path.normalize(SDK_PATH)) return false;
      return realExistsSync(p);
    });

    await runMain({ dbPath: path.join(tmpDir, "index") });

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("@tobilu/qmd not found at"));
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Checkout collection: basic usage ──────────────────────────────────────
  it("builds index from a checkout collection", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);
    fs.writeFileSync(path.join(docsDir, "readme.md"), "# README\nHello world");
    fs.writeFileSync(path.join(docsDir, "guide.md"), "# Guide\nFoo bar");

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir, patterns: ["**/*.md"], context: "Project docs" }],
    });

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
    expect(mockStore.embed).toHaveBeenCalledOnce();
    expect(mockStore.close).toHaveBeenCalledOnce();
  });

  // ── Checkout collection: env-var expansion ────────────────────────────────
  it("resolves ${ENV_VAR} placeholders in checkout paths", async () => {
    const workspaceDir = path.join(tmpDir, "workspace");
    fs.mkdirSync(workspaceDir);
    process.env.GITHUB_WORKSPACE = workspaceDir;

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: "${GITHUB_WORKSPACE}", patterns: ["**/*.md"] }],
    });

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
  });

  // ── Checkout collection: default pattern ─────────────────────────────────
  it("uses **/*.md as the default pattern when none specified", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir }],
    });

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
  });

  // ── Issues search: valid repo ─────────────────────────────────────────────
  it("fetches issues and saves them as markdown files", async () => {
    process.env.GITHUB_REPOSITORY = "owner/repo";
    mockGithub.paginate.mockResolvedValue([
      { number: 1, title: "First issue", body: "Body one" },
      { number: 2, title: "Second issue", body: "Body two" },
    ]);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "issues", type: "issues", max: 10 }],
    });

    expect(mockGithub.paginate).toHaveBeenCalledOnce();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();

    // The script writes search results to /tmp/gh-aw/qmd-search-N (hardcoded in
    // qmd_index.cjs, Linux-specific, mirrors the GitHub Actions runner).
    const searchDir = "/tmp/gh-aw/qmd-search-0";
    if (fs.existsSync(searchDir)) {
      const files = fs.readdirSync(searchDir);
      expect(files).toContain("issue-1.md");
      expect(files).toContain("issue-2.md");
      const content = fs.readFileSync(path.join(searchDir, "issue-1.md"), "utf8");
      expect(content).toContain("## 1: First issue");
    }
  });

  // ── Issues search: explicit repo field ───────────────────────────────────
  it("uses explicit repo field instead of GITHUB_REPOSITORY for issues search", async () => {
    process.env.GITHUB_REPOSITORY = "default/repo";
    mockGithub.paginate.mockResolvedValue([]);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "issues", type: "issues", repo: "explicit/repo" }],
    });

    expect(mockGithub.paginate).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({ owner: "explicit", repo: "repo" }), expect.any(Function));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  // ── Issues search: invalid slug (no slash) ───────────────────────────────
  it("fails when issues search repo slug has no slash", async () => {
    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "issues", type: "issues", repo: "invalid-no-slash" }],
    });

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining('invalid repository slug "invalid-no-slash"'));
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Issues search: empty slug (GITHUB_REPOSITORY not set) ────────────────
  it("fails when issues search slug is empty (GITHUB_REPOSITORY unset)", async () => {
    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "issues", type: "issues" }],
    });

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("invalid repository slug"));
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Issues search: min count enforcement ─────────────────────────────────
  it("fails when issues search returns fewer results than min", async () => {
    process.env.GITHUB_REPOSITORY = "owner/repo";
    mockGithub.paginate.mockResolvedValue([{ number: 1, title: "Only one", body: "" }]);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "issues", type: "issues", min: 5 }],
    });

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("minimum is 5"));
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Code search: downloads files ──────────────────────────────────────────
  it("downloads code search results and registers them as a collection", async () => {
    mockGithub.rest.search.code.mockResolvedValue({
      data: {
        items: [
          { path: "docs/README.md", repository: { full_name: "owner/repo" } },
          { path: "docs/guide.md", repository: { full_name: "owner/repo" } },
        ],
      },
    });
    mockGithub.rest.repos.getContent.mockResolvedValue({
      data: { type: "file", content: Buffer.from("# Content").toString("base64") },
    });

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "api-docs", query: "repo:owner/repo language:Markdown path:docs/", max: 10 }],
    });

    expect(mockGithub.rest.search.code).toHaveBeenCalledWith(expect.objectContaining({ q: "repo:owner/repo language:Markdown path:docs/" }));
    expect(mockGithub.rest.repos.getContent).toHaveBeenCalledTimes(2);
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();

    // /tmp/gh-aw/qmd-search-0 is the hardcoded search dir in qmd_index.cjs.
    const searchDir = "/tmp/gh-aw/qmd-search-0";
    if (fs.existsSync(searchDir)) {
      const files = fs.readdirSync(searchDir);
      expect(files.some(f => f.includes("owner-repo-docs-README.md"))).toBe(true);
      expect(files.some(f => f.includes("owner-repo-docs-guide.md"))).toBe(true);
    }
  });

  // ── Code search: min count enforcement ───────────────────────────────────
  it("fails when code search returns fewer results than min", async () => {
    mockGithub.rest.search.code.mockResolvedValue({ data: { items: [] } });

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "docs", query: "repo:owner/repo", min: 3 }],
    });

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("minimum is 3"));
    expect(mockStore.update).not.toHaveBeenCalled();
  });

  // ── Code search: download error is a warning, not a failure ──────────────
  it("emits a warning (not failure) when getContent throws for a code search item", async () => {
    mockGithub.rest.search.code.mockResolvedValue({
      data: {
        items: [{ path: "README.md", repository: { full_name: "owner/repo" } }],
      },
    });
    mockGithub.rest.repos.getContent.mockRejectedValue(new Error("404 Not Found"));

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "docs", query: "repo:owner/repo" }],
    });

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not download owner/repo/README.md"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
  });

  // ── Code search: skip items with malformed full_name ─────────────────────
  it("skips code search items whose repository full_name has no slash", async () => {
    mockGithub.rest.search.code.mockResolvedValue({
      data: {
        items: [{ path: "file.md", repository: { full_name: "no-slash" } }],
      },
    });

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "docs", query: "test" }],
    });

    expect(mockGithub.rest.repos.getContent).not.toHaveBeenCalled();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
  });

  // ── Combined: checkouts + searches ───────────────────────────────────────
  it("combines checkout collections and search results into one index", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);
    fs.writeFileSync(path.join(docsDir, "readme.md"), "# README");

    process.env.GITHUB_REPOSITORY = "owner/repo";
    mockGithub.paginate.mockResolvedValue([{ number: 10, title: "Issue", body: "" }]);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir, patterns: ["**/*.md"] }],
      searches: [{ name: "issues", type: "issues", max: 50 }],
    });

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockStore.update).toHaveBeenCalledOnce();
    expect(mockStore.embed).toHaveBeenCalledOnce();
    expect(mockStore.close).toHaveBeenCalledOnce();
  });

  // ── finally: store.close() always called ─────────────────────────────────
  it("always calls store.close() even when store.update() throws", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);
    mockStore.update.mockRejectedValue(new Error("update failed"));

    await expect(
      runMain({
        dbPath: path.join(tmpDir, "index"),
        checkouts: [{ name: "docs", path: docsDir }],
      })
    ).rejects.toThrow("update failed");

    expect(mockStore.close).toHaveBeenCalledOnce();
  });

  // ── writeSummary: checkouts section ──────────────────────────────────────
  it("writes step summary with a collections table for checkouts", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir, patterns: ["**/*.md", "**/*.mdx"], context: "Project docs" }],
    });

    const summaryText = mockCore.summary.addRaw.mock.calls.flat().join("\n");
    expect(summaryText).toContain("### Collections");
    expect(summaryText).toContain("| docs | **/*.md, **/*.mdx | Project docs |");
    expect(mockCore.summary.write).toHaveBeenCalledOnce();
  });

  // ── writeSummary: searches section ───────────────────────────────────────
  it("writes step summary with a searches table", async () => {
    mockGithub.rest.search.code.mockResolvedValue({ data: { items: [] } });

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      searches: [{ name: "api-docs", query: "repo:owner/repo language:Markdown", min: 0, max: 20 }],
    });

    const summaryText = mockCore.summary.addRaw.mock.calls.flat().join("\n");
    expect(summaryText).toContain("### Searches");
    expect(summaryText).toContain("| api-docs | code | repo:owner/repo language:Markdown |");
  });

  // ── writeSummary: index stats section ────────────────────────────────────
  it("writes step summary with update and embed statistics", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);
    mockStore.update.mockResolvedValue({ indexed: 7, updated: 2, unchanged: 1, removed: 0 });
    mockStore.embed.mockResolvedValue({ embedded: 9 });

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir }],
    });

    const summaryText = mockCore.summary.addRaw.mock.calls.flat().join("\n");
    expect(summaryText).toContain("### Index stats");
    expect(summaryText).toContain("| Indexed | 7 |");
    expect(summaryText).toContain("| Embedded | 9 |");
  });

  // ── writeSummary: error handling ─────────────────────────────────────────
  it("emits a warning (not failure) when writing the step summary throws", async () => {
    const docsDir = path.join(tmpDir, "docs");
    fs.mkdirSync(docsDir);
    mockCore.summary.write.mockRejectedValue(new Error("step summary unavailable"));

    await runMain({
      dbPath: path.join(tmpDir, "index"),
      checkouts: [{ name: "docs", path: docsDir }],
    });

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not write step summary"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
