// @ts-check

import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { reduceModelNameToIdentifier, formatModelEmojiAlias, formatModelEmojiAliasLegend } = require("./model_aliases.cjs");

describe("reduceModelNameToIdentifier", () => {
  it('returns "auto" unchanged (short name, 4 chars)', () => {
    expect(reduceModelNameToIdentifier("auto")).toBe("auto");
  });

  it("returns safe short model names (< 6 alphanumeric chars) unchanged", () => {
    expect(reduceModelNameToIdentifier("o1")).toBe("o1");
    expect(reduceModelNameToIdentifier("o3")).toBe("o3");
    expect(reduceModelNameToIdentifier("gpt")).toBe("gpt");
    expect(reduceModelNameToIdentifier("mini")).toBe("mini");
    expect(reduceModelNameToIdentifier("haiku")).toBe("haiku");
  });

  it("sanitizes short model names that contain non-alphanumeric characters", () => {
    // "a|b" has a pipe character that would split a Markdown table row — must be compacted
    expect(reduceModelNameToIdentifier("a|b")).toBe("abx00");
    // "gpt-5" has a hyphen, so it falls through to the GPT family shortcut
    expect(reduceModelNameToIdentifier("gpt-5")).toBe("gpt50");
  });

  it("processes names of exactly 6 characters through normalization (boundary)", () => {
    // 6 chars is not < 6, so the short-name guard does not apply; normalization runs
    expect(reduceModelNameToIdentifier("gpt-4o")).toBe("gpt40");
  });

  it("returns empty string for empty/null/undefined input", () => {
    expect(reduceModelNameToIdentifier("")).toBe("");
    expect(reduceModelNameToIdentifier(null)).toBe("");
    expect(reduceModelNameToIdentifier(undefined)).toBe("");
  });

  it("handles known Claude model families", () => {
    expect(reduceModelNameToIdentifier("claude-sonnet-4.5")).toBe("sonnet45");
    expect(reduceModelNameToIdentifier("claude-opus-5")).toBe("opus50");
    expect(reduceModelNameToIdentifier("claude-haiku-4.5")).toBe("haiku45");
  });

  it("handles GPT model families", () => {
    expect(reduceModelNameToIdentifier("gpt-5")).toBe("gpt50");
    expect(reduceModelNameToIdentifier("gpt-4o")).toBe("gpt40");
  });

  it("handles Gemini model families", () => {
    expect(reduceModelNameToIdentifier("gemini-3.1-pro")).toBe("gem31pro");
  });

  it("normalizes to lowercase before processing", () => {
    expect(reduceModelNameToIdentifier("AUTO")).toBe("auto");
    expect(reduceModelNameToIdentifier("Auto")).toBe("auto");
  });

  it("uses fallback identifier for unrecognized longer model names", () => {
    // "unknown-model" -> compact "unknownmodel" (12 chars) -> letterPart "unk", digitPart "00" -> "unk00"
    expect(reduceModelNameToIdentifier("unknown-model")).toBe("unk00");
  });
});

describe("formatModelEmojiAlias", () => {
  it('returns "auto" unchanged', () => {
    expect(formatModelEmojiAlias("auto")).toBe("auto");
  });
});

describe("formatModelEmojiAliasLegend", () => {
  it("produces legend entries for a list of models", () => {
    const result = formatModelEmojiAliasLegend(["auto", "claude-sonnet-4.5"]);
    expect(result).toContain("auto=auto");
    expect(result).toContain("sonnet45=claude-sonnet-4.5");
  });
});
