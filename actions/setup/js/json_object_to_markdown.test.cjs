import { describe, it, expect } from "vitest";

const { humanifyKey, jsonObjectToMarkdown } = await import("./json_object_to_markdown.cjs");

describe("json_object_to_markdown.cjs", () => {
  describe("humanifyKey", () => {
    it("should replace underscores with spaces", () => {
      expect(humanifyKey("engine_id")).toBe("engine id");
      expect(humanifyKey("firewall_enabled")).toBe("firewall enabled");
      expect(humanifyKey("awf_version")).toBe("awf version");
    });

    it("should replace hyphens with spaces", () => {
      expect(humanifyKey("run-id")).toBe("run id");
      expect(humanifyKey("awf-version")).toBe("awf version");
    });

    it("should leave keys without separators unchanged", () => {
      expect(humanifyKey("version")).toBe("version");
      expect(humanifyKey("model")).toBe("model");
    });
  });

  it("should render flat key-value pairs with humanified keys", () => {
    const obj = { engine_id: "copilot", version: "v1.0.0" };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **engine id**: copilot");
    expect(result).toContain("- **version**: v1.0.0");
  });

  it("should render boolean values as true/false strings", () => {
    const obj = { firewall_enabled: true, staged: false };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **firewall enabled**: true");
    expect(result).toContain("- **staged**: false");
  });

  it("should render null/undefined/empty string values as (none)", () => {
    const obj = { model: "", awf_version: null, agent_version: undefined };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **model**: (none)");
    expect(result).toContain("- **awf version**: (none)");
    expect(result).toContain("- **agent version**: (none)");
  });

  it("should render non-empty arrays as sub-bullet lists", () => {
    const obj = { allowed_domains: ["example.com", "github.com"] };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **allowed domains**:");
    expect(result).toContain("  - example.com");
    expect(result).toContain("  - github.com");
  });

  it("should render empty arrays as (none)", () => {
    const obj = { allowed_domains: [] };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **allowed domains**: (none)");
  });

  it("should render nested objects as indented sub-bullet lists", () => {
    const obj = { steps: { firewall: "iptables" } };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **steps**:");
    expect(result).toContain("  - **firewall**: iptables");
  });

  it("should return empty string for null or non-object input", () => {
    expect(jsonObjectToMarkdown(null)).toBe("");
    expect(jsonObjectToMarkdown(undefined)).toBe("");
    expect(jsonObjectToMarkdown([])).toBe("");
  });

  it("should handle numeric values", () => {
    const obj = { run_id: 12345, run_number: 7 };
    const result = jsonObjectToMarkdown(obj);
    expect(result).toContain("- **run id**: 12345");
    expect(result).toContain("- **run number**: 7");
  });
});
