import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createSharedState } from "../src/types.js";
import { loadUserExtensions } from "../src/user-extensions.js";
import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";

describe("loadUserExtensions", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let stderrSpy: any;

  beforeEach(() => {
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    stderrSpy.mockRestore();
  });

  it("returns an empty array when no refs are provided", async () => {
    const result = await loadUserExtensions([], process.cwd(), false);
    expect(result).toEqual([]);
  });

  it("emits a warning and skips a failing extension when extensionsRequired is false", async () => {
    const result = await loadUserExtensions(["./nonexistent-extension.cjs"], process.cwd(), false);
    expect(result).toEqual([]);
    expect(stderrSpy).toHaveBeenCalled();
    const warnCall = stderrSpy.mock.calls.find((args: unknown[]) =>
      String(args[0]).includes("nonexistent-extension"),
    );
    expect(warnCall).toBeDefined();
  });

  it("throws when extensionsRequired is true and an extension fails to load", async () => {
    await expect(
      loadUserExtensions(["./nonexistent-extension.cjs"], process.cwd(), true),
    ).rejects.toThrow("nonexistent-extension");
  });
});

describe("createSharedState", () => {
  it("initialises state with correct defaults", () => {
    const state = createSharedState("claude-sonnet-4.6");
    expect(state.budgetAborted).toBe(false);
    expect(state.cumulativeTokens).toBe(0);
    expect(state.cumulativeCostUsd).toBe(0);
    expect(state.turnCount).toBe(0);
    expect(state.sessionStartMs).toBe(0);
    expect(state.model).toBe("claude-sonnet-4.6");
  });
});
