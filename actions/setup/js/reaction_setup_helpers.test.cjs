// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  setFailed: vi.fn(),
};

global.core = mockCore;

describe("reaction_setup_helpers", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.resetModules();
    delete process.env.GH_AW_REACTION;
  });

  async function importHelpers() {
    return import("./reaction_setup_helpers.cjs?" + Date.now());
  }

  it("uses eyes as the default reaction", async () => {
    const { resolveReactionSetup } = await importHelpers();

    const result = resolveReactionSetup({
      eventName: "issues",
      repo: { owner: "testowner", repo: "testrepo" },
      payload: { issue: { number: 123 } },
    });

    expect(result?.reaction).toBe("eyes");
    expect(result?.invocationContext.eventName).toBe("issues");
  });

  it("uses GH_AW_REACTION when provided", async () => {
    process.env.GH_AW_REACTION = "rocket";
    const { resolveReactionSetup } = await importHelpers();

    const result = resolveReactionSetup({
      eventName: "issues",
      repo: { owner: "testowner", repo: "testrepo" },
      payload: { issue: { number: 123 } },
    });

    expect(result?.reaction).toBe("rocket");
  });

  it("fails and returns null for invalid reaction", async () => {
    process.env.GH_AW_REACTION = "invalid";
    const { resolveReactionSetup } = await importHelpers();

    const result = resolveReactionSetup({
      eventName: "issues",
      repo: { owner: "testowner", repo: "testrepo" },
      payload: { issue: { number: 123 } },
    });

    expect(result).toBeNull();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid reaction type: invalid"));
  });

  it("VALID_REACTIONS stays in sync with REACTION_MAP keys", async () => {
    const { VALID_REACTIONS } = await importHelpers();
    const { REACTION_MAP } = await import("./add_reaction.cjs?" + Date.now());

    expect([...VALID_REACTIONS].sort()).toEqual(Object.keys(REACTION_MAP).sort());
  });
});
