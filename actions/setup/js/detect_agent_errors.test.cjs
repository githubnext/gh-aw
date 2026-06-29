import { describe, it, expect } from "vitest";

const {
  detectErrors,
  isCAPIQuotaExceededError,
  INFERENCE_ACCESS_ERROR_PATTERN,
  MCP_POLICY_BLOCKED_PATTERN,
  AGENTIC_ENGINE_TIMEOUT_PATTERN,
  MODEL_NOT_SUPPORTED_PATTERN,
  HTTP_400_RESPONSE_ERROR_PATTERN,
  CAPI_QUOTA_EXCEEDED_PATTERN,
} = require("./detect_agent_errors.cjs");

describe("detect_agent_errors.cjs", () => {
  describe("INFERENCE_ACCESS_ERROR_PATTERN", () => {
    it("matches 'Access denied by policy settings'", () => {
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test("Access denied by policy settings")).toBe(true);
    });

    it("matches 'invalid access to inference'", () => {
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test("invalid access to inference")).toBe(true);
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some output\nError: Access denied by policy settings\nMore output";
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test(log)).toBe(true);
    });

    it("does not match unrelated errors", () => {
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test("MCP server connection failed")).toBe(false);
      expect(INFERENCE_ACCESS_ERROR_PATTERN.test("")).toBe(false);
    });
  });

  describe("MCP_POLICY_BLOCKED_PATTERN", () => {
    it("matches the exact error from the issue report", () => {
      const errorOutput = "! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'";
      expect(MCP_POLICY_BLOCKED_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches with different server names", () => {
      expect(MCP_POLICY_BLOCKED_PATTERN.test("! 1 MCP servers were blocked by policy: 'github'")).toBe(true);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("MCP servers were blocked by policy: 'custom-server'")).toBe(true);
    });

    it("does not match unrelated errors", () => {
      expect(MCP_POLICY_BLOCKED_PATTERN.test("Error: MCP server connection failed")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("MCP server timeout")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("")).toBe(false);
    });
  });

  describe("AGENTIC_ENGINE_TIMEOUT_PATTERN", () => {
    it("matches copilot-harness process closed with SIGTERM", () => {
      const log = "[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s stdout=1234B stderr=567B hasOutput=true";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches copilot-harness process exit with SIGTERM", () => {
      const log = "[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process exit event exitCode=null signal=SIGTERM";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches SIGKILL from any engine", () => {
      const log = "process closed exitCode=1 signal=SIGKILL duration=20m 0s";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches SIGINT from any engine", () => {
      const log = "process closed exitCode=1 signal=SIGINT duration=20m 0s";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some agent output\n✓ All tests pass\n[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s\nMore output";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches signal from non-copilot engine logs", () => {
      const log = "Claude CLI terminated with signal=SIGTERM after timeout";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("does not match regular exit without signal", () => {
      const log = "[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 duration=5m 3s stdout=1234B stderr=567B hasOutput=true";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("MCP server timeout")).toBe(false);
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("")).toBe(false);
    });
  });

  describe("MODEL_NOT_SUPPORTED_PATTERN", () => {
    it("matches the exact error from the issue report", () => {
      const errorOutput = "Execution failed: CAPIError: 400 The requested model is not supported.";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some output\nExecution failed: CAPIError: 400 The requested model is not supported.\nMore output";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(log)).toBe(true);
    });

    it("matches invalid/unknown model name variants", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("invalid model name 'claude-sonnet-999'")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("unknown model gpt-unknown")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model 'gpt-foo' not found")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model 'claude-ultra' does not exist")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model claude-fake is not supported")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model gpt-unknown is not available")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model gemini-v99 is unavailable")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("model 'claude-3-5-sonnet@20241022' not found")).toBe(true);
    });

    it("matches AIC api-proxy 404 standalone 'Model not found' shape", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("404 Not Found: Model not found")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("ResponseError: 404 Not Found: Model not found")).toBe(true);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Error: 404 Model not found")).toBe(true);
    });

    it("matches 404 model-not-found when embedded in larger log output", () => {
      const log = "Some output\n[codex-harness] attempt 2: exitCode=1 isInvalidModelError=false\nError 404 Not Found: Model not found\nall 3 retries exhausted — giving up";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(log)).toBe(true);
    });

    it("does not match unrelated invalid/unknown model wording", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Error: invalid model response format")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Error: invalid model schema definition")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("unknown model behavior detected")).toBe(false);
    });

    it("does not match other CAPIError 400 errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("CAPIError: 400 400 Bad Request")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("MCP servers were blocked by policy: 'github'")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("")).toBe(false);
    });
  });

  describe("CAPI_QUOTA_EXCEEDED_PATTERN / isCAPIQuotaExceededError", () => {
    it("matches the exact observed error message", () => {
      expect(CAPI_QUOTA_EXCEEDED_PATTERN.test("CAPIError: 429 429 quota exceeded")).toBe(true);
      expect(isCAPIQuotaExceededError("CAPIError: 429 429 quota exceeded")).toBe(true);
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some agent output\nExecution failed: CAPIError: 429 429 quota exceeded\nMore output";
      expect(isCAPIQuotaExceededError(log)).toBe(true);
    });

    it("matches with varying whitespace around status codes", () => {
      expect(isCAPIQuotaExceededError("CAPIError:429 429 quota exceeded")).toBe(true);
      expect(isCAPIQuotaExceededError("CAPIError:  429  429  quota exceeded")).toBe(true);
    });

    it("matches case-insensitively", () => {
      expect(isCAPIQuotaExceededError("CAPIError: 429 429 QUOTA EXCEEDED")).toBe(true);
    });

    it("matches Copilot/CAPI Too Many Requests output", () => {
      expect(isCAPIQuotaExceededError("CAPIError: 429 Too Many Requests")).toBe(true);
      expect(isCAPIQuotaExceededError("Last error: CAPIError: Too Many Requests")).toBe(true);
    });

    it("does not match CAPIError 400", () => {
      expect(isCAPIQuotaExceededError("CAPIError: 400 Bad Request")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(isCAPIQuotaExceededError("Access denied by policy settings")).toBe(false);
      expect(isCAPIQuotaExceededError("MCP servers were blocked by policy: 'github'")).toBe(false);
      expect(isCAPIQuotaExceededError("")).toBe(false);
    });
  });

  describe("HTTP_400_RESPONSE_ERROR_PATTERN", () => {
    it("matches the generic HTTP 400 Bad Request response shape", () => {
      expect(HTTP_400_RESPONSE_ERROR_PATTERN.test("Response status code does not indicate success: 400 (Bad Request)")).toBe(true);
    });

    it("matches the HTTP 400 response shape without the (Bad Request) suffix", () => {
      expect(HTTP_400_RESPONSE_ERROR_PATTERN.test("Response status code does not indicate success: 400")).toBe(true);
    });

    it("does not match unrelated 400 text", () => {
      expect(HTTP_400_RESPONSE_ERROR_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(HTTP_400_RESPONSE_ERROR_PATTERN.test("Error: 400 Bad Request")).toBe(false);
    });
  });

  describe("detectErrors", () => {
    it("returns all false for empty log", () => {
      const result = detectErrors("");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects inference access error only", () => {
      const result = detectErrors("Error: Access denied by policy settings");
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects MCP policy error only", () => {
      const result = detectErrors("! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(true);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects engine timeout only", () => {
      const result = detectErrors("[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(true);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects model not supported error only", () => {
      const result = detectErrors("Execution failed: CAPIError: 400 The requested model is not supported.");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(true);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects invalid model name errors", () => {
      const result = detectErrors("Error: invalid model name 'claude-sonnet-999'");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(true);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects CAPI quota exceeded error only", () => {
      const result = detectErrors("Execution failed: CAPIError: 429 429 quota exceeded");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(true);
    });

    it("detects HTTP 400 response error only", () => {
      const result = detectErrors("Response status code does not indicate success: 400 (Bad Request)");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(true);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects both errors in the same log", () => {
      const log = "Access denied by policy settings\nMCP servers were blocked by policy: 'github'";
      const result = detectErrors(log);
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(true);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("detects timeout alongside other errors", () => {
      const log = "Access denied by policy settings\n[copilot-harness] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 0s";
      const result = detectErrors(log);
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(true);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });

    it("returns false for unrelated log content", () => {
      const result = detectErrors("CAPIError: 400 Bad Request\nSome normal output");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
      expect(result.modelNotSupportedError).toBe(false);
      expect(result.http400ResponseError).toBe(false);
      expect(result.capiQuotaExceededError).toBe(false);
    });
  });
});
