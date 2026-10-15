// @ts-check
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { expectedGitHubHost, isTrustedGitHubApiUrl } = require("./work_queue_host.cjs");
const { resolveRepositoryTarget } = require("./work_queue_effect_resource.cjs");
let originalEnv;

beforeEach(() => {
  originalEnv = { ...process.env };
});

afterEach(() => {
  process.env = originalEnv;
});

describe("work queue GitHub host configuration", () => {
  it("defaults to GitHub.com and accepts a strict enterprise server URL", () => {
    delete process.env.GITHUB_SERVER_URL;
    expect(expectedGitHubHost()).toBe("github.com");
    process.env.GITHUB_SERVER_URL = "https://GitHub.Contoso.example/";
    expect(expectedGitHubHost()).toBe("github.contoso.example");
  });

  it.each(["http://github.com", "https://github.com@evil.example", "https://github.com:8443", "https://github.com:443", "https://github.com/path", "https://github.com?query=1", "https://github.com?", "https://github.com#fragment"])(
    "rejects untrusted server URL %s",
    serverUrl => {
      process.env.GITHUB_SERVER_URL = serverUrl;
      expect(() => expectedGitHubHost()).toThrow("Work queue requires a valid GITHUB_SERVER_URL");
    }
  );

  it("accepts only API endpoints on the configured server host", async () => {
    process.env.GITHUB_SERVER_URL = "https://contoso.example";
    process.env.GITHUB_API_URL = "https://api.contoso.example/api/v3";
    expect(isTrustedGitHubApiUrl()).toBe(true);
    expect(await resolveRepositoryTarget({}, { repository: "owner/repo" })).toMatchObject({ host: "contoso.example" });

    process.env.GITHUB_API_URL = "https://contoso.example/api/v3";
    expect(isTrustedGitHubApiUrl()).toBe(true);
    process.env.GITHUB_API_URL = "https://api.evil.example";
    expect(isTrustedGitHubApiUrl()).toBe(false);
    await expect(resolveRepositoryTarget({}, { repository: "owner/repo" })).rejects.toThrow("does not match the configured GitHub server");
  });

  it("rejects malformed API URLs", () => {
    process.env.GITHUB_API_URL = "not a URL";
    expect(() => isTrustedGitHubApiUrl()).toThrow("Work queue requires a valid GITHUB_API_URL");
    for (const apiUrl of ["https://@api.contoso.example", "https://api.contoso.example:443", "https://api.contoso.example?"]) {
      process.env.GITHUB_API_URL = apiUrl;
      expect(() => isTrustedGitHubApiUrl()).toThrow("Work queue requires a valid GITHUB_API_URL");
    }
  });
});
