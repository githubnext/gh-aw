// @ts-check
import { describe, it, expect, afterEach, vi } from "vitest";
import { createRequire } from "module";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Set GH_AW_PROMPTS_DIR before loading any module that transitively reads prompts
// at initialization time (messages_header.cjs reads safe_outputs_disclosure_header.md).
process.env.GH_AW_PROMPTS_DIR = path.resolve(__dirname, "../md");

const req = createRequire(import.meta.url);

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};
global.core = mockCore;

const { formatFailedJobsList } = req("./report_failed_jobs.cjs");

afterEach(() => {
  vi.clearAllMocks();
});

describe("formatFailedJobsList", () => {
  it("formats a plain job name with a URL as a markdown link", () => {
    const result = formatFailedJobsList([{ name: "my-job", html_url: "https://github.com/owner/repo/actions/runs/1/jobs/2" }]);
    expect(result).toBe("- [`my-job`](https://github.com/owner/repo/actions/runs/1/jobs/2)");
  });

  it("formats a plain job name without a URL as a plain code span", () => {
    const result = formatFailedJobsList([{ name: "my-job", html_url: null }]);
    expect(result).toBe("- `my-job`");
  });

  it("sanitizes markdown/HTML special characters in job name", () => {
    const result = formatFailedJobsList([{ name: "<script>alert(1)</script>", html_url: null }]);
    expect(result).not.toContain("<script>");
    expect(result).not.toContain("</script>");
  });

  it("sanitizes job name containing HTML comment injection", () => {
    const result = formatFailedJobsList([{ name: "<!-- @exploituser injected payload -->", html_url: null }]);
    expect(result).not.toContain("@exploituser");
  });

  it("rejects html_url with javascript: scheme and renders name-only fallback", () => {
    const result = formatFailedJobsList([{ name: "my-job", html_url: "javascript:alert(1)" }]);
    expect(result).not.toContain("javascript:");
    expect(result).toBe("- `my-job`");
  });

  it("joins multiple jobs with newlines", () => {
    const result = formatFailedJobsList([
      { name: "job-a", html_url: null },
      { name: "job-b", html_url: null },
    ]);
    expect(result).toBe("- `job-a`\n- `job-b`");
  });

  it("returns empty string for empty jobs array", () => {
    expect(formatFailedJobsList([])).toBe("");
  });
});
