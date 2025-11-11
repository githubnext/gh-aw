import { describe, it, expect } from "vitest";

describe("generateFingerprintComment", () => {
  it("should return empty string when fingerprint is empty", async () => {
    const { generateFingerprintComment } = await import("./generate_fingerprint_comment.cjs");

    const result = generateFingerprintComment("");

    expect(result).toBe("");
  });

  it("should return HTML comment with fingerprint when provided", async () => {
    const { generateFingerprintComment } = await import("./generate_fingerprint_comment.cjs");

    const result = generateFingerprintComment("test-fp-12345");

    expect(result).toBe("\n\n<!-- fingerprint: test-fp-12345 -->");
  });

  it("should handle fingerprint with underscores", async () => {
    const { generateFingerprintComment } = await import("./generate_fingerprint_comment.cjs");

    const result = generateFingerprintComment("project_alpha_2024");

    expect(result).toBe("\n\n<!-- fingerprint: project_alpha_2024 -->");
  });

  it("should handle fingerprint with hyphens", async () => {
    const { generateFingerprintComment } = await import("./generate_fingerprint_comment.cjs");

    const result = generateFingerprintComment("project-alpha-2024");

    expect(result).toBe("\n\n<!-- fingerprint: project-alpha-2024 -->");
  });

  it("should handle fingerprint with mixed alphanumeric", async () => {
    const { generateFingerprintComment } = await import("./generate_fingerprint_comment.cjs");

    const result = generateFingerprintComment("Test123_Project-v2");

    expect(result).toBe("\n\n<!-- fingerprint: Test123_Project-v2 -->");
  });
});
