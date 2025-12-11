import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

let generateCampaignId;
let normalizeProjectName;

beforeEach(async () => {
  const mod = await import("./project_helpers.cjs");
  const exports = mod.default || mod;
  generateCampaignId = exports.generateCampaignId;
  normalizeProjectName = exports.normalizeProjectName;
});

afterEach(() => {
  vi.useRealTimers();
});

describe("generateCampaignId", () => {
  it("builds a slug with a timestamp suffix", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Bug Bash Q1 2025");
    expect(id).toBe("bug-bash-q1-2025-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("handles project names with special characters", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Project! @#$ %^& *()");
    expect(id).toBe("project-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("removes leading and trailing dashes", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("---Project---");
    expect(id).toBe("project-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("truncates long project names to 30 characters", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const longName = "This is a very long project name that exceeds thirty characters";
    const id = generateCampaignId(longName);
    expect(id).toBe("this-is-a-very-long-project-na-m4syw5xc");
    expect(id.length).toBeLessThanOrEqual(39); // 30 + dash + 8 char timestamp
    nowSpy.mockRestore();
  });

  it("handles empty strings", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("");
    expect(id).toBe("-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("handles strings with only special characters", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("!@#$%^&*()");
    expect(id).toBe("-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("preserves numbers in project names", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Project 2025 Q1");
    expect(id).toBe("project-2025-q1-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("converts uppercase to lowercase", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("PROJECT NAME");
    expect(id).toBe("project-name-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("collapses multiple spaces/special chars into single dash", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Project    Name!!!");
    expect(id).toBe("project-name-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("generates unique timestamps for different calls", () => {
    const id1 = generateCampaignId("Test Project");
    const id2 = generateCampaignId("Test Project");

    // Extract timestamps (last 8 characters)
    const timestamp1 = id1.slice(-8);
    const timestamp2 = id2.slice(-8);

    // Timestamps might be the same if calls are very close, but should be valid base36
    expect(/^[a-z0-9]{8}$/.test(timestamp1)).toBe(true);
    expect(/^[a-z0-9]{8}$/.test(timestamp2)).toBe(true);
  });

  it("handles unicode characters", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("Projet été 2025");
    // Non-ASCII characters should be removed
    expect(id).toMatch(/^projet-t-2025-m4syw5xc$/);
    nowSpy.mockRestore();
  });

  it("handles project names with underscores", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("project_name_test");
    expect(id).toBe("project-name-test-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("handles project names with dots", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("project.name.test");
    expect(id).toBe("project-name-test-m4syw5xc");
    nowSpy.mockRestore();
  });

  it("handles mixed case and special characters", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("My_Project-2025.Q1");
    expect(id).toBe("my-project-2025-q1-m4syw5xc");
    nowSpy.mockRestore();
  });
});

describe("normalizeProjectName", () => {
  it("trims whitespace from project name", () => {
    expect(normalizeProjectName("  Test Project  ")).toBe("Test Project");
  });

  it("trims leading whitespace", () => {
    expect(normalizeProjectName("  Test Project")).toBe("Test Project");
  });

  it("trims trailing whitespace", () => {
    expect(normalizeProjectName("Test Project  ")).toBe("Test Project");
  });

  it("preserves internal whitespace", () => {
    expect(normalizeProjectName("Test   Project   Name")).toBe("Test   Project   Name");
  });

  it("handles tabs and newlines", () => {
    expect(normalizeProjectName("\t\nTest Project\n\t")).toBe("Test Project");
  });

  it("returns empty string if input is empty after trim", () => {
    expect(normalizeProjectName("   ")).toBe("");
  });

  it("preserves valid project names without whitespace", () => {
    expect(normalizeProjectName("TestProject")).toBe("TestProject");
  });

  it("throws error for null input", () => {
    expect(() => normalizeProjectName(null)).toThrow(/Invalid project name/);
  });

  it("throws error for undefined input", () => {
    expect(() => normalizeProjectName(undefined)).toThrow(/Invalid project name/);
  });

  it("throws error for non-string input", () => {
    expect(() => normalizeProjectName(123)).toThrow(/Invalid project name/);
  });

  it("throws error for object input", () => {
    expect(() => normalizeProjectName({})).toThrow(/Invalid project name/);
  });

  it("throws error for array input", () => {
    expect(() => normalizeProjectName([])).toThrow(/Invalid project name/);
  });

  it("handles project names with special characters (no normalization beyond trim)", () => {
    expect(normalizeProjectName("  Project-2025!  ")).toBe("Project-2025!");
  });

  it("handles very long project names", () => {
    const longName = "A".repeat(1000);
    expect(normalizeProjectName(longName)).toBe(longName);
  });
});
