import { describe, it, expect } from "vitest";

const { detectErrors, INFERENCE_ACCESS_ERROR_PATTERN, MCP_POLICY_BLOCKED_PATTERN, AGENTIC_ENGINE_TIMEOUT_PATTERN } = require("./detect_copilot_errors.cjs");

describe("detect_copilot_errors.cjs", () => {
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
    it("matches copilot-driver process closed with SIGTERM", () => {
      const log = "[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s stdout=1234B stderr=567B hasOutput=true";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches copilot-driver process exit with SIGTERM", () => {
      const log = "[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process exit event exitCode=null signal=SIGTERM";
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
      const log = "Some agent output\n✓ All tests pass\n[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s\nMore output";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("matches signal from non-copilot engine logs", () => {
      const log = "Claude CLI terminated with signal=SIGTERM after timeout";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(true);
    });

    it("does not match regular exit without signal", () => {
      const log = "[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 duration=5m 3s stdout=1234B stderr=567B hasOutput=true";
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(log)).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("MCP server timeout")).toBe(false);
      expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test("")).toBe(false);
    });
  });

  describe("detectErrors", () => {
    it("returns all false for empty log", () => {
      const result = detectErrors("");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
    });

    it("detects inference access error only", () => {
      const result = detectErrors("Error: Access denied by policy settings");
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
    });

    it("detects MCP policy error only", () => {
      const result = detectErrors("! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(true);
      expect(result.agenticEngineTimeout).toBe(false);
    });

    it("detects engine timeout only", () => {
      const result = detectErrors("[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 12s");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(true);
    });

    it("detects both errors in the same log", () => {
      const log = "Access denied by policy settings\nMCP servers were blocked by policy: 'github'";
      const result = detectErrors(log);
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(true);
      expect(result.agenticEngineTimeout).toBe(false);
    });

    it("detects timeout alongside other errors", () => {
      const log = "Access denied by policy settings\n[copilot-driver] 2026-04-12T04:56:28.000Z attempt 1: process closed exitCode=1 signal=SIGTERM duration=20m 0s";
      const result = detectErrors(log);
      expect(result.inferenceAccessError).toBe(true);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(true);
    });

    it("returns false for unrelated log content", () => {
      const result = detectErrors("CAPIError: 400 Bad Request\nSome normal output");
      expect(result.inferenceAccessError).toBe(false);
      expect(result.mcpPolicyError).toBe(false);
      expect(result.agenticEngineTimeout).toBe(false);
    });
  });
});
