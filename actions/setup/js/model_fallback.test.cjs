import { describe, expect, it } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { MODEL_FALLBACK_ENV_VAR, resolveModelWithFallback, applyModelFallback, injectModelFlagAfterExec } = require("./model_fallback.cjs");

describe("model_fallback.cjs", () => {
  it("prefers the primary model env var when it is non-empty", () => {
    expect(resolveModelWithFallback({ COPILOT_MODEL: "gpt-5", [MODEL_FALLBACK_ENV_VAR]: "claude-sonnet-4.6" }, "COPILOT_MODEL")).toBe("gpt-5");
  });

  it("uses the fallback model when the primary resolves empty", () => {
    const env = { COPILOT_MODEL: "", [MODEL_FALLBACK_ENV_VAR]: "claude-sonnet-4.6" };
    expect(applyModelFallback(env, "COPILOT_MODEL")).toBe("claude-sonnet-4.6");
    expect(env.COPILOT_MODEL).toBe("claude-sonnet-4.6");
  });

  it("injects --model immediately after exec when needed", () => {
    expect(injectModelFlagAfterExec(["exec", "--json", "prompt"], "gpt-5")).toEqual(["exec", "--model", "gpt-5", "--json", "prompt"]);
  });

  it("does not inject --model when already present", () => {
    expect(injectModelFlagAfterExec(["exec", "--model", "gpt-5", "--json"], "claude-sonnet-4.6")).toEqual(["exec", "--model", "gpt-5", "--json"]);
  });
});
