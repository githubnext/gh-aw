// @ts-check
import fs from "node:fs";
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";

describe("check_version_updates", () => {
  let mockCore;
  let mockFetch;
  let checkVersionUpdates;

  beforeEach(async () => {
    vi.useFakeTimers();

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    global.core = mockCore;

    mockFetch = vi.fn();
    vi.stubGlobal("fetch", mockFetch);

    delete process.env.GH_AW_COMPILED_VERSION;

    vi.resetModules();

    checkVersionUpdates = await import("./check_version_updates.cjs");
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  /**
   * Helper to mock a successful fetch response with the given body.
   * @param {string} body
   */
  function mockFetchSuccess(body) {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve(body),
    });
  }

  /**
   * Helper to mock a failed fetch request (network error).
   * @param {Error} err
   */
  function mockFetchError(err) {
    mockFetch.mockRejectedValue(err);
  }

  /**
   * Run main() and advance all pending timers to process retry delays.
   * @returns {Promise<void>}
   */
  async function runMain() {
    const promise = checkVersionUpdates.main();
    await vi.runAllTimersAsync();
    return promise;
  }

  function readSchema(path) {
    return JSON.parse(fs.readFileSync(path, "utf8"));
  }

  // ---------------------------------------------------------------------------
  // Skip cases — version not subject to check
  // ---------------------------------------------------------------------------

  it("should skip check when version is 'dev'", async () => {
    process.env.GH_AW_COMPILED_VERSION = "dev";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("dev"));
  });

  it("should skip check when version is empty", async () => {
    process.env.GH_AW_COMPILED_VERSION = "";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("should skip check when version has no 'v' prefix (not an official release)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "1.0.0";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when version is a non-semver string", async () => {
    process.env.GH_AW_COMPILED_VERSION = "latest";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when version has only two parts (v1.0 is invalid)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when version has non-numeric parts (e.g. v1.0.alpha)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.alpha";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when version has extra dot segments (v1.2.3.4 is not vMAJOR.MINOR.PATCH)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.3.4";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when base numeric identifiers have leading zeroes", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.02.0";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("should skip check when prerelease numeric identifiers have leading zeroes", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0-01";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not an official release version"));
  });

  it("fetches the compatibility matrix from the gh-aw repository", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "" }));
    await runMain();
    expect(mockFetch).toHaveBeenCalledWith("https://raw.githubusercontent.com/github/gh-aw/main/.github/aw/compat.json", expect.any(Object));
  });

  it("keeps blocked-version schema patterns in sync with the runtime parser", () => {
    const compatSchema = readSchema("../../../.github/aw/compat.schema.json");
    const releasesSchema = readSchema("../../../.github/aw/releases.schema.json");

    expect(compatSchema.properties.blockedVersions.items.pattern).toBe(checkVersionUpdates.VERSION_PATTERN_SOURCE);
    expect(releasesSchema.properties.blockedVersions.items.pattern).toBe(checkVersionUpdates.VERSION_PATTERN_SOURCE);
  });

  it("keeps stable policy schema patterns in sync with runtime policy validation", () => {
    const compatSchema = readSchema("../../../.github/aw/compat.schema.json");
    const releasesSchema = readSchema("../../../.github/aw/releases.schema.json");

    for (const schema of [compatSchema, releasesSchema]) {
      expect(schema.properties.minimumVersion.pattern).toBe(checkVersionUpdates.CONFIG_STABLE_VERSION_PATTERN_SOURCE);
      expect(schema.properties.minRecommendedVersion.pattern).toBe(checkVersionUpdates.CONFIG_STABLE_VERSION_PATTERN_SOURCE);
    }
  });

  it("keeps workflow blocked-version patterns in sync with the runtime parser", () => {
    const cgoWorkflow = fs.readFileSync("../../../.github/workflows/cgo.yml", "utf8");
    const yamlEscapedPattern = checkVersionUpdates.VERSION_PATTERN_SOURCE.replaceAll("\\", "\\\\");

    expect(cgoWorkflow.match(/const blockedVersionPatternSource = /g)).toHaveLength(2);
    expect(Array.from(cgoWorkflow.matchAll(/const blockedVersionPatternSource = '([^']+)'/g), match => match[1])).toEqual([yamlEscapedPattern, yamlEscapedPattern]);
  });

  // ---------------------------------------------------------------------------
  // Network / download failure cases (soft fail)
  // ---------------------------------------------------------------------------

  it("should skip check when all fetch attempts fail (ECONNREFUSED)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchError(new Error("ECONNREFUSED"));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should skip check when all fetch attempts fail (ECONNRESET)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchError(new Error("ECONNRESET"));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should retry and succeed when fetch fails transiently", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    // First call fails transiently; second call succeeds
    mockFetch.mockRejectedValueOnce(new Error("ECONNRESET")).mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: () => Promise.resolve(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" })),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should skip check when server returns 404", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetch.mockResolvedValue({ ok: false, status: 404, text: () => Promise.resolve("") });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should skip check when server returns 500 after exhausting retries", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    // 500 should trigger retries; after all retries exhausted, soft-fail
    mockFetch.mockResolvedValue({ ok: false, status: 500, text: () => Promise.resolve("") });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
    // Should have retried (default 3 retries = 4 total attempts: 1 initial + 3 retries)
    expect(mockFetch).toHaveBeenCalledTimes(4);
  });

  it("should retry on 500 and succeed if a later attempt succeeds", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    // First call returns 500, second succeeds
    mockFetch.mockResolvedValueOnce({ ok: false, status: 500, text: () => Promise.resolve("") }).mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: () => Promise.resolve(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" })),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should skip check when response is not valid JSON (soft fail)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve("not json {{{"),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should skip check when response is empty string (soft fail)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve(""),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should skip check when response is an HTML error page (soft fail)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve("<!DOCTYPE html><html><body>Error</body></html>"),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  // ---------------------------------------------------------------------------
  // Version check passes
  // ---------------------------------------------------------------------------

  it("should pass when version is not blocked and meets minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v0.9.0"], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should pass when version exactly equals minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should pass when blockedVersions is empty and no minimumVersion", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should pass when config has no blockedVersions field", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ minimumVersion: "v0.5.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass when config has no minimumVersion field", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v0.9.0"] }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should pass when config is an empty object", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({}));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  // ---------------------------------------------------------------------------
  // Blocked version cases
  // ---------------------------------------------------------------------------

  it("should fail when version is in blocked list (exact match)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.1.0", "v1.2.0"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v1.1.0"));
  });

  it("should fail when version is in blocked list (one of many)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v2.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0", "v2.0.0", "v3.0.0"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v2.0.0"));
  });

  it("should fail when a prerelease version is in the blocked list", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0-beta.1";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.2.0-beta.1"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v1.2.0-beta.1"));
  });

  it("should ignore an invalid prerelease version in the blocked list", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0-beta.1";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.2.0-..beta"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should ignore a blocked prerelease with leading-zero numeric identifiers", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0-beta.1";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.2.0-01"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should not treat different large numeric prerelease identifiers as equal", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0-alpha.9007199254740992";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.2.0-alpha.9007199254740993"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should not block a release version with a prerelease block entry", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0-beta.1"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should not block an earlier prerelease with a later prerelease block entry", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0-alpha.1";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0-alpha.2"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should compare numeric prerelease identifiers numerically, not lexically", async () => {
    expect(checkVersionUpdates.compareVersions("v1.0.0-alpha.2", "v1.0.0-alpha.10")).toBeLessThan(0);
  });

  it("should compare release versions greater than their prereleases", async () => {
    expect(checkVersionUpdates.compareVersions("v1.0.0", "v1.0.0-beta.1")).toBeGreaterThan(0);
  });

  it("should NOT block version when blocked list entry has no 'v' prefix (unknown format — ignore)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["1.0.0"], minimumVersion: "" }));
    // "1.0.0" in blocked list has no v prefix — it should be ignored, not matched
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should NOT block when blocked list contains unknown/garbage versions", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["latest", "stable", "1.0.0", "vfoo"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should correctly identify block when blocked list mixes valid and invalid entries", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["1.0.0", "latest", "v1.0.0", "v999.0.0"], minimumVersion: "" }));
    // Only "v1.0.0" is a valid entry and it matches
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });

  it("should write summary when version is blocked", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Minimum version cases
  // ---------------------------------------------------------------------------

  it("should fail when version is below minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.8.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v0.8.0"));
  });

  it("should fail when major version is older (v0.x.x vs min v1.0.0)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.99.99";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
  });

  it("should compare large base version components without precision loss", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v9007199254740992.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v9007199254740993.0.0" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
  });

  it("should fail when minor version is older (v1.0.5 vs min v1.1.0)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.5";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.1.0" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
  });

  it("should fail when patch version is older (v1.0.0 vs min v1.0.1)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.1" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
  });

  it("should skip minimum check when minimumVersion is empty string", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should skip minimum check when minimumVersion has no 'v' prefix (unknown format — ignore)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    // minimumVersion "2.0.0" has no v prefix — should be ignored
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "2.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should skip minimum check when minimumVersion is a garbage string", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "latest" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should skip minimum check when minimumVersion is a prerelease", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0-alpha.1" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should write summary when version is below minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.8.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  // ---------------------------------------------------------------------------
  // Config structure edge cases
  // ---------------------------------------------------------------------------

  it("should pass when blockedVersions is null (treated as missing)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: null, minimumVersion: "v0.5.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass when minimumVersion is null (treated as missing)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: null }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass when config has extra unknown fields", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(
      JSON.stringify({
        blockedVersions: [],
        minimumVersion: "v0.5.0",
        futureField: "some-value",
        anotherField: 42,
      })
    );
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass when config is a JSON array (not object) — treats as empty config", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify([]));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should pass when config body is JSON null — treats as empty config", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess("null");
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  // ---------------------------------------------------------------------------
  // minRecommendedVersion (soft check — warning only, no failure)
  // ---------------------------------------------------------------------------

  it("should warn when version is below minRecommendedVersion", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.9.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Recommended upgrade"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("v0.9.0"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("v1.0.0"));
  });

  it("should pass without warning when version equals minRecommendedVersion", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should pass without warning when version is above minRecommendedVersion", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip recommended check when minRecommendedVersion is empty string", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.5.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip recommended check when minRecommendedVersion has no 'v' prefix (unknown format — ignore)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.5.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "2.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip recommended check when minRecommendedVersion is a garbage string", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.5.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "latest" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip recommended check when minRecommendedVersion is a prerelease", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.5.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: "v1.0.0-alpha.1" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should skip recommended check when minRecommendedVersion is null (treated as missing)", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.5.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "", minRecommendedVersion: null }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("should fail hard (not just warn) when version is also below minimumVersion", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.8.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0", minRecommendedVersion: "v1.0.0" }));
    await runMain();
    // Should fail, not just warn
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("minimum supported version"));
  });
});
