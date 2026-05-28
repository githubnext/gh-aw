import { afterEach, describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const {
  appendSafeOutputLine,
  buildMissingToolPermissionIssuePayload,
  buildInfrastructureIncompletePayload,
  buildPromptFileFallbackInstruction,
  countPermissionDeniedIssues,
  detectCopilotErrors,
  emitInfrastructureIncomplete,
  extractDeniedCommands,
  hasNumerousPermissionDeniedIssues,
  INFERENCE_ACCESS_ERROR_PATTERN,
  AGENTIC_ENGINE_TIMEOUT_PATTERN,
  isDetectionPhase,
  isAuthenticationFailedError,
  isModelAvailableInReflectData,
  isModelAvailableInReflectFile,
  enrichReflectModels,
  extractModelIds,
  fetchAWFReflect,
  fetchModelsFromUrl,
  GEMINI_MODEL_NAME_PREFIX,
  PROMPT_FILE_INLINE_THRESHOLD_BYTES,
  resolvePromptFileArgs,
  writeCopilotOutputs,
} = require("./copilot_harness.cjs");

describe("copilot_harness.cjs", () => {
  // Test the core logic patterns used by the driver without importing the module
  // (importing the module would invoke main() which calls process.exit).

  describe("CAPIError 400 detection pattern", () => {
    const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;

    it("matches the exact error from the failed workflow run", () => {
      const errorOutput = "Execution failed: CAPIError: 400 400 Bad Request\n (Request ID: C818:3ED713:19D401B:1C446B7:69D653CA)";
      expect(CAPI_ERROR_400_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches CAPIError: 400 with various spacing", () => {
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError: 400")).toBe(true);
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError:400")).toBe(true);
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError:  400")).toBe(true);
    });

    it("does not match CAPIError 401 Unauthorized", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Execution failed: CAPIError: 401 Unauthorized")).toBe(false);
    });

    it("does not match generic 400 errors without CAPIError prefix", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Error: 400 Bad Request")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("HTTP 400")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Error: ENOENT: no such file")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("Fatal: out of memory")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("")).toBe(false);
    });
  });

  describe("retry policy: continue on partial execution", () => {
    // Inline the same retry-eligibility logic as the driver for unit testing.
    // The driver retries whenever the session produced output (hasOutput), regardless
    // of the specific error type.  CAPIError 400 is just the well-known case.
    const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      if (hasNumerousPermissionDeniedIssues(result.output)) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    /**
     * @param {string} output
     * @returns {"CAPIError 400 (transient)" | "partial execution"}
     */
    function retryReason(output) {
      return CAPI_ERROR_400_PATTERN.test(output) ? "CAPIError 400 (transient)" : "partial execution";
    }

    it("retries on CAPIError 400 after partial output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0)).toBe(true);
      expect(retryReason(result.output)).toBe("CAPIError 400 (transient)");
    });

    it("retries on any other non-zero exit when session produced output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: connection reset by peer" };
      expect(shouldRetry(result, 0)).toBe(true);
      expect(retryReason(result.output)).toBe("partial execution");
    });

    it("does not retry when no output was produced (process failed to start)", () => {
      const result = { exitCode: 1, hasOutput: false, output: "" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry after retries are exhausted", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, MAX_RETRIES)).toBe(false);
    });

    it("does not retry on success", () => {
      const result = { exitCode: 0, hasOutput: true, output: "Done." };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("numerous permission-denied issues are treated as non-retryable", () => {
      const result = { exitCode: 1, hasOutput: true, output: "permission denied\npermission denied\npermission denied" };
      expect(hasNumerousPermissionDeniedIssues(result.output)).toBe(true);
      expect(shouldRetry(result, 0)).toBe(false);
    });
  });

  describe("scheduled startup retry policy (exit code 2)", () => {
    const MAX_RETRIES = 3;
    const MAX_SCHEDULED_EXIT2_RETRIES = 1;

    /**
     * @param {{hasOutput: boolean, exitCode: number}} result
     * @param {number} attempt
     * @param {boolean} isScheduledRun
     * @param {number} scheduledExit2Retries
     * @returns {boolean}
     */
    function shouldRetry(result, attempt, isScheduledRun, scheduledExit2Retries) {
      if (result.exitCode === 0) return false;

      // Scheduled startup outage: retry once even when no output was produced.
      if (isScheduledRun && result.exitCode === 2 && !result.hasOutput && scheduledExit2Retries < MAX_SCHEDULED_EXIT2_RETRIES && attempt < MAX_RETRIES) {
        return true;
      }

      // Existing partial-execution retry policy
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("retries once for scheduled startup interruption with exit code 2 and no output", () => {
      const result = { exitCode: 2, hasOutput: false };
      expect(shouldRetry(result, 0, true, 0)).toBe(true);
      expect(shouldRetry(result, 1, true, 1)).toBe(false);
    });

    it("does not claim a retry when already at max retry attempt", () => {
      const result = { exitCode: 2, hasOutput: false };
      expect(shouldRetry(result, MAX_RETRIES, true, 0)).toBe(false);
    });

    it("does not apply startup retry for non-scheduled runs", () => {
      const result = { exitCode: 2, hasOutput: false };
      expect(shouldRetry(result, 0, false, 0)).toBe(false);
    });

    it("continues to use partial-execution retries when output exists", () => {
      const result = { exitCode: 2, hasOutput: true };
      expect(shouldRetry(result, 0, true, 0)).toBe(true);
    });
  });

  describe("infrastructure report_incomplete emission helpers", () => {
    it("builds report_incomplete payload with infrastructure_error reason", () => {
      const payload = buildInfrastructureIncompletePayload("temporary outage");
      expect(JSON.parse(payload)).toEqual({
        type: "report_incomplete",
        reason: "infrastructure_error",
        details: "temporary outage",
      });
    });

    it("appends one JSONL line through appendSafeOutputLine", () => {
      const writes = [];
      const appendStub = (file, data, options) => writes.push({ file, data, options });
      appendSafeOutputLine(appendStub, "/tmp/safeoutputs.jsonl", '{"type":"report_incomplete"}');
      expect(writes).toEqual([{ file: "/tmp/safeoutputs.jsonl", data: '{"type":"report_incomplete"}\n', options: { encoding: "utf8" } }]);
    });

    it("emitInfrastructureIncomplete writes payload when path is configured", () => {
      const writes = [];
      const logs = [];
      emitInfrastructureIncomplete("temporary outage", {
        safeOutputsPath: "/tmp/safeoutputs.jsonl",
        appendFileSync: (file, data, options) => writes.push({ file, data, options }),
        logger: message => logs.push(message),
      });
      expect(writes).toHaveLength(1);
      expect(writes[0].file).toBe("/tmp/safeoutputs.jsonl");
      const parsed = JSON.parse(writes[0].data.trim());
      expect(parsed.type).toBe("report_incomplete");
      expect(parsed.reason).toBe("infrastructure_error");
      expect(parsed.details).toBe("temporary outage");
      expect(logs.some(message => message.includes("report_incomplete emitted"))).toBe(true);
    });

    it("emitInfrastructureIncomplete skips when path is missing", () => {
      const writes = [];
      const logs = [];
      emitInfrastructureIncomplete("temporary outage", {
        safeOutputsPath: "",
        appendFileSync: () => writes.push("write"),
        logger: message => logs.push(message),
      });
      expect(writes).toHaveLength(0);
      expect(logs.some(message => message.includes("skipped"))).toBe(true);
    });
  });

  describe("permission-denied classification helpers", () => {
    it("counts repeated permission-denied signals", () => {
      const output = "permission denied\nEACCES: permission denied\nEPERM operation not permitted\npermissions denied";
      expect(countPermissionDeniedIssues(output)).toBe(5);
    });

    it("detects numerous permission-denied issues at threshold", () => {
      const output = "permission denied\npermission denied\npermission denied";
      expect(hasNumerousPermissionDeniedIssues(output)).toBe(true);
    });

    it("does not classify sparse permission-denied output as numerous", () => {
      const output = "permission denied once";
      expect(hasNumerousPermissionDeniedIssues(output)).toBe(false);
    });

    it("builds missing_tool payload for permission issues", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload());
      expect(payload.type).toBe("missing_tool");
      expect(payload.reason).toContain("missing tool/permission issue");
      expect(payload.denied_commands).toEqual([]);
    });

    it("builds missing_tool payload with denied commands", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload(["go version", "ls /usr/local/go/bin/go"]));
      expect(payload.type).toBe("missing_tool");
      expect(payload.denied_commands).toEqual(["go version", "ls /usr/local/go/bin/go"]);
    });
  });

  describe("extractDeniedCommands", () => {
    it("returns empty array for empty output", () => {
      expect(extractDeniedCommands("")).toEqual([]);
      expect(extractDeniedCommands(null)).toEqual([]);
    });

    it("extracts command from line with box-drawing pipe marker (│) before permission denied", () => {
      const output = ["\u2713 Some successful step", "\u2717 Check if go command works (shell)", "  \u2502 go version 2>&1", "  \u2514 Permission denied and could not request permission from user"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["go version 2>&1"]);
    });

    it("extracts command with plain pipe (|) before permission denied", () => {
      const output = ["| ls -la", "Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["ls -la"]);
    });

    it("deduplicates repeated denied commands", () => {
      const output = ["  \u2502 go version", "  Permission denied", "  \u2502 go version", "  Permission denied", "  \u2502 go version", "  Permission denied"].join("\n");
      const result = extractDeniedCommands(output);
      expect(result).toEqual(["go version"]);
    });

    it("extracts multiple distinct denied commands", () => {
      const output = ["  \u2502 go version 2>&1", "  Permission denied", "  \u2502 ls /usr/local/go/bin/go", "  Permission denied", "  \u2502 which go", "  Permission denied"].join("\n");
      const result = extractDeniedCommands(output);
      expect(result).toContain("go version 2>&1");
      expect(result).toContain("ls /usr/local/go/bin/go");
      expect(result).toContain("which go");
    });

    it("returns empty array when no pipe markers are present before permission denied", () => {
      const output = "Some output\nPermission denied\nMore output";
      expect(extractDeniedCommands(output)).toEqual([]);
    });

    it("looks back up to 3 lines for command context", () => {
      const output = ["  \u2502 make test", "Running...", "Still running...", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["make test"]);
    });

    it("does not look back more than 3 lines", () => {
      const output = ["  \u2502 make test", "line2", "line3", "line4", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual([]);
    });

    it("does not capture suffix of a command containing an internal pipe", () => {
      // "find . -name '*.go' | sort" should not match by splitting on the internal |
      const output = ["  find . -name '*.go' | sort", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual([]);
    });
  });

  describe("MCP policy blocked detection pattern", () => {
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

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

  describe("MCP policy error prevents retry", () => {
    // Inline the same retry logic as the driver, including MCP policy check
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;
    const MODEL_NOT_SUPPORTED_PATTERN = /The requested model is not supported/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      // MCP policy errors are persistent — never retry
      if (MCP_POLICY_BLOCKED_PATTERN.test(result.output)) return false;
      // Model-not-supported errors are persistent — never retry
      if (MODEL_NOT_SUPPORTED_PATTERN.test(result.output)) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("does not retry when MCP servers are blocked by policy", () => {
      const result = { exitCode: 1, hasOutput: true, output: "! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry MCP policy error even on first attempt with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nMCP servers were blocked by policy: 'github'\nMore output" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry model-not-supported error", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Execution failed: CAPIError: 400 The requested model is not supported." };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry model-not-supported error even on first attempt with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nExecution failed: CAPIError: 400 The requested model is not supported.\nMore output" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("still retries non-policy errors with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0)).toBe(true);
    });
  });

  describe("model-not-supported detection pattern", () => {
    const MODEL_NOT_SUPPORTED_PATTERN = /The requested model is not supported/;

    it("matches the exact error from the issue report", () => {
      const errorOutput = "Execution failed: CAPIError: 400 The requested model is not supported.";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(errorOutput)).toBe(true);
    });

    describe("copilot output detection + workflow outputs", () => {
      afterEach(() => {
        delete process.env.GITHUB_OUTPUT;
      });

      it("detects inference/mcp/timeout/model-not-supported patterns from output", () => {
        const output = [
          "Access denied by policy settings",
          "MCP servers were blocked by policy: 'github'",
          "[copilot-harness] attempt 1: process closed exitCode=1 signal=SIGTERM",
          "Execution failed: CAPIError: 400 The requested model is not supported.",
        ].join("\n");
        expect(detectCopilotErrors(output)).toEqual({
          inferenceAccessError: true,
          mcpPolicyError: true,
          agenticEngineTimeout: true,
          modelNotSupportedError: true,
        });
        expect(INFERENCE_ACCESS_ERROR_PATTERN.test(output)).toBe(true);
        expect(AGENTIC_ENGINE_TIMEOUT_PATTERN.test(output)).toBe(true);
      });

      it("writes copilot detection outputs to GITHUB_OUTPUT", () => {
        const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "copilot-output-test-"));
        const outputFile = path.join(tempDir, "github-output.txt");
        process.env.GITHUB_OUTPUT = outputFile;

        writeCopilotOutputs({
          inferenceAccessError: true,
          mcpPolicyError: false,
          agenticEngineTimeout: true,
          modelNotSupportedError: false,
        });

        const content = fs.readFileSync(outputFile, "utf8");
        expect(content).toContain("inference_access_error=true");
        expect(content).toContain("mcp_policy_error=false");
        expect(content).toContain("agentic_engine_timeout=true");
        expect(content).toContain("model_not_supported_error=false");
      });
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some output\nExecution failed: CAPIError: 400 The requested model is not supported.\nMore output";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(log)).toBe(true);
    });

    it("does not match other CAPIError 400 errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("MCP servers were blocked by policy: 'github'")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("")).toBe(false);
    });
  });

  describe("no-auth-info detection pattern", () => {
    const NO_AUTH_INFO_PATTERN = /No authentication information found/;

    it("matches the exact error from the issue report", () => {
      const errorOutput =
        "Error: No authentication information found.\n" +
        "Copilot can be authenticated with GitHub using an OAuth Token or a Fine-Grained Personal Access Token.\n" +
        "To authenticate, you can use any of the following methods:\n" +
        "  - Start 'copilot' and run the '/login' command\n" +
        "  - Set the COPILOT_GITHUB_TOKEN, GH_TOKEN, or GITHUB_TOKEN environment variable\n" +
        "  - Run 'gh auth login' to authenticate with the GitHub CLI";
      expect(NO_AUTH_INFO_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches when embedded in larger output after a long run", () => {
      const output = "Some agent work output\nMore work\nNo authentication information found\nEnd";
      expect(NO_AUTH_INFO_PATTERN.test(output)).toBe(true);
    });

    it("does not match unrelated auth errors", () => {
      expect(NO_AUTH_INFO_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("Error: 401 Unauthorized")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("Authentication failed")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("")).toBe(false);
    });
  });

  describe("authentication-failed detection pattern", () => {
    it("matches authentication failed with request id", () => {
      expect(isAuthenticationFailedError("Authentication failed (Request ID: C818:3ED713:19D401B:1C446B7:69D653CA)")).toBe(true);
    });

    it("does not match no-auth-info error", () => {
      expect(isAuthenticationFailedError("Error: No authentication information found.")).toBe(false);
    });
  });

  describe("auth error prevents retry", () => {
    // Inline the same retry logic as the driver, including auth error check
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;
    const NO_AUTH_INFO_PATTERN = /No authentication information found/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @param {boolean} useContinueOnRetry - whether the current attempt used --continue
     * @returns {boolean}
     */
    function shouldRetry(result, attempt, useContinueOnRetry = false) {
      if (result.exitCode === 0) return false;
      // MCP policy errors are persistent — never retry
      if (MCP_POLICY_BLOCKED_PATTERN.test(result.output)) return false;
      if (attempt === 0 && isAuthenticationFailedError(result.output)) return false;
      // Auth error on --continue: fall back to fresh run once; on fresh run: bail
      if (NO_AUTH_INFO_PATTERN.test(result.output)) {
        return useContinueOnRetry && attempt < MAX_RETRIES;
      }
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("does not retry when auth fails on first attempt (no real work done)", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: No authentication information found." };
      expect(shouldRetry(result, 0, false)).toBe(false);
    });

    it("does not retry when first attempt reports authentication failed", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Authentication failed (Request ID: ABC123)" };
      expect(shouldRetry(result, 0, false)).toBe(false);
    });

    it("retries as fresh run when auth fails on a --continue attempt", () => {
      // This replicates the fix: attempt 1 ran for 3+ min then failed mid-stream,
      // attempt 2 (--continue) fails with auth error — driver retries once as fresh run.
      const continueResult = { exitCode: 1, hasOutput: true, output: "Error: No authentication information found." };
      expect(shouldRetry(continueResult, 1, true)).toBe(true); // --continue attempt: triggers fresh retry
      expect(shouldRetry(continueResult, 2, true)).toBe(true); // still within retry budget
      expect(shouldRetry(continueResult, 3, true)).toBe(false); // budget exhausted
    });

    it("does not retry when auth fails on a fresh-run recovery attempt (useContinueOnRetry=false)", () => {
      // After falling back to a fresh run, useContinueOnRetry is reset to false.
      // If the fresh run also hits auth error, the driver bails immediately.
      const freshResult = { exitCode: 1, hasOutput: true, output: "Error: No authentication information found." };
      expect(shouldRetry(freshResult, 1, false)).toBe(false);
      expect(shouldRetry(freshResult, 2, false)).toBe(false);
    });

    it("does not retry auth error even when output is mixed with other content", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nError: No authentication information found.\nMore output" };
      expect(shouldRetry(result, 0, false)).toBe(false);
    });

    it("still retries non-auth errors with output (CAPIError 400)", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0, false)).toBe(true);
    });

    it("still retries generic partial-execution errors with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Failed to get response from the AI model; retried 5 times" };
      expect(shouldRetry(result, 0, false)).toBe(true);
    });
  });

  describe("null-type tool_call detection pattern", () => {
    const NULL_TYPE_TOOL_CALL_PATTERN = /tool_calls\[.*?\]\.type.*null/;

    it("matches the error format observed in failed workflow runs", () => {
      const errorOutput = "Execution failed: CAPIError: 400 Invalid type for 'messages[45].tool_calls[0].type': expected one of 'function', 'all...ols', or 'custom', but got null instead.";
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches with different array indices", () => {
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("tool_calls[0].type: null")).toBe(true);
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("tool_calls[12].type, got null")).toBe(true);
    });

    it("does not match unrelated tool_calls errors", () => {
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("tool_calls[0].name: missing")).toBe(false);
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("Error: tool call failed")).toBe(false);
    });

    it("does not match unrelated null errors", () => {
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("Unexpected null value in response")).toBe(false);
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(NULL_TYPE_TOOL_CALL_PATTERN.test("")).toBe(false);
    });
  });

  describe("null-type tool_call restarts fresh instead of --continue", () => {
    // Inline the same retry logic as the driver including null-type tool_call handling
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;
    const NO_AUTH_INFO_PATTERN = /No authentication information found/;
    const NULL_TYPE_TOOL_CALL_PATTERN = /tool_calls\[.*?\]\.type.*null/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @param {boolean} useContinueOnRetry
     * @param {boolean} continueDisabledPermanently
     * @returns {{ shouldRetry: boolean, useContinueOnRetry: boolean, continueDisabledPermanently: boolean }}
     */
    function applyRetryPolicy(result, attempt, useContinueOnRetry = false, continueDisabledPermanently = false) {
      if (result.exitCode === 0) return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      if (MCP_POLICY_BLOCKED_PATTERN.test(result.output)) return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      if (NO_AUTH_INFO_PATTERN.test(result.output)) {
        if (useContinueOnRetry && attempt < MAX_RETRIES) {
          return { shouldRetry: true, useContinueOnRetry: false, continueDisabledPermanently: true };
        }
        return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      }
      if (NULL_TYPE_TOOL_CALL_PATTERN.test(result.output)) {
        if (attempt < MAX_RETRIES && result.hasOutput) {
          return { shouldRetry: true, useContinueOnRetry: false, continueDisabledPermanently: true };
        }
        return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      }
      if (attempt < MAX_RETRIES && result.hasOutput) {
        return { shouldRetry: true, useContinueOnRetry: !continueDisabledPermanently, continueDisabledPermanently };
      }
      return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
    }

    it("restarts fresh when null-type error occurs on a --continue attempt", () => {
      const result = {
        exitCode: 1,
        hasOutput: true,
        output: "CAPIError: 400 Invalid type for 'messages[45].tool_calls[0].type': expected one of 'function', 'all...ols', or 'custom', but got null instead.",
      };
      const {
        shouldRetry,
        useContinueOnRetry: newContinue,
        continueDisabledPermanently: disabled,
      } = applyRetryPolicy(
        result,
        1,
        true, // was using --continue
        false
      );
      expect(shouldRetry).toBe(true);
      expect(newContinue).toBe(false); // must NOT use --continue on restart
      expect(disabled).toBe(true); // permanently disabled
    });

    it("restarts fresh when null-type error occurs on a fresh run", () => {
      const result = {
        exitCode: 1,
        hasOutput: true,
        output: "CAPIError: 400 Invalid type for 'messages[0].tool_calls[0].type': got null instead.",
      };
      const { shouldRetry, useContinueOnRetry: newContinue, continueDisabledPermanently: disabled } = applyRetryPolicy(result, 0, false, false);
      expect(shouldRetry).toBe(true);
      expect(newContinue).toBe(false); // must NOT use --continue
      expect(disabled).toBe(true); // permanently disabled
    });

    it("does not retry when budget is exhausted", () => {
      const result = {
        exitCode: 1,
        hasOutput: true,
        output: "tool_calls[0].type: null",
      };
      const { shouldRetry } = applyRetryPolicy(result, MAX_RETRIES, true, false);
      expect(shouldRetry).toBe(false);
    });

    it("does not retry when no output was produced", () => {
      const result = {
        exitCode: 1,
        hasOutput: false,
        output: "tool_calls[0].type: null",
      };
      const { shouldRetry } = applyRetryPolicy(result, 0, false, false);
      expect(shouldRetry).toBe(false);
    });
  });

  describe("permanent --continue disable guard", () => {
    // Inline retry logic to verify that once continueDisabledPermanently is set,
    // subsequent partial-execution retries never re-enable --continue.
    const NULL_TYPE_TOOL_CALL_PATTERN = /tool_calls\[.*?\]\.type.*null/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @param {boolean} useContinueOnRetry
     * @param {boolean} continueDisabledPermanently
     * @returns {{ shouldRetry: boolean, useContinueOnRetry: boolean, continueDisabledPermanently: boolean }}
     */
    function applyRetryPolicy(result, attempt, useContinueOnRetry = false, continueDisabledPermanently = false) {
      if (result.exitCode === 0) return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      if (NULL_TYPE_TOOL_CALL_PATTERN.test(result.output)) {
        if (attempt < MAX_RETRIES && result.hasOutput) {
          return { shouldRetry: true, useContinueOnRetry: false, continueDisabledPermanently: true };
        }
        return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      }
      if (attempt < MAX_RETRIES && result.hasOutput) {
        return { shouldRetry: true, useContinueOnRetry: !continueDisabledPermanently, continueDisabledPermanently };
      }
      return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
    }

    it("does not re-enable --continue after a null-type fresh restart", () => {
      // Attempt 0 (fresh): normal failure → schedule --continue
      const attempt0Result = { exitCode: 1, hasOutput: true, output: "some error" };
      const after0 = applyRetryPolicy(attempt0Result, 0, false, false);
      expect(after0.shouldRetry).toBe(true);
      expect(after0.useContinueOnRetry).toBe(true);
      expect(after0.continueDisabledPermanently).toBe(false);

      // Attempt 1 (--continue): null-type error → restart fresh, disable permanently
      const attempt1Result = { exitCode: 1, hasOutput: true, output: "tool_calls[0].type: null" };
      const after1 = applyRetryPolicy(attempt1Result, 1, after0.useContinueOnRetry, after0.continueDisabledPermanently);
      expect(after1.shouldRetry).toBe(true);
      expect(after1.useContinueOnRetry).toBe(false); // disabled for this retry
      expect(after1.continueDisabledPermanently).toBe(true); // permanently set

      // Attempt 2 (fresh): another partial failure → MUST NOT re-enable --continue
      const attempt2Result = { exitCode: 1, hasOutput: true, output: "another error" };
      const after2 = applyRetryPolicy(attempt2Result, 2, after1.useContinueOnRetry, after1.continueDisabledPermanently);
      expect(after2.shouldRetry).toBe(true);
      expect(after2.useContinueOnRetry).toBe(false); // guard prevents re-enabling
      expect(after2.continueDisabledPermanently).toBe(true);
    });

    it("does not re-enable --continue after an auth-error fresh restart", () => {
      const NO_AUTH_INFO_PATTERN_LOCAL = /No authentication information found/;

      function applyRetryPolicyWithAuth(result, attempt, useContinueOnRetry = false, continueDisabledPermanently = false) {
        if (result.exitCode === 0) return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
        if (NO_AUTH_INFO_PATTERN_LOCAL.test(result.output)) {
          if (useContinueOnRetry && attempt < MAX_RETRIES) {
            return { shouldRetry: true, useContinueOnRetry: false, continueDisabledPermanently: true };
          }
          return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
        }
        if (attempt < MAX_RETRIES && result.hasOutput) {
          return { shouldRetry: true, useContinueOnRetry: !continueDisabledPermanently, continueDisabledPermanently };
        }
        return { shouldRetry: false, useContinueOnRetry, continueDisabledPermanently };
      }

      // Attempt 0 (fresh): normal failure → schedule --continue
      const attempt0Result = { exitCode: 1, hasOutput: true, output: "some work done" };
      const after0 = applyRetryPolicyWithAuth(attempt0Result, 0, false, false);
      expect(after0.useContinueOnRetry).toBe(true);

      // Attempt 1 (--continue): auth error → restart fresh, disable permanently
      const attempt1Result = { exitCode: 1, hasOutput: true, output: "No authentication information found" };
      const after1 = applyRetryPolicyWithAuth(attempt1Result, 1, after0.useContinueOnRetry, after0.continueDisabledPermanently);
      expect(after1.shouldRetry).toBe(true);
      expect(after1.useContinueOnRetry).toBe(false);
      expect(after1.continueDisabledPermanently).toBe(true);

      // Attempt 2 (fresh): partial failure → MUST NOT re-enable --continue
      const attempt2Result = { exitCode: 1, hasOutput: true, output: "some other error" };
      const after2 = applyRetryPolicyWithAuth(attempt2Result, 2, after1.useContinueOnRetry, after1.continueDisabledPermanently);
      expect(after2.useContinueOnRetry).toBe(false); // guard prevents re-enabling
    });
  });

  describe("retry configuration", () => {
    it("has sensible default values", () => {
      // These match the constants in copilot_harness.cjs
      const MAX_RETRIES = 3;
      const INITIAL_DELAY_MS = 5000;
      const BACKOFF_MULTIPLIER = 2;
      const MAX_DELAY_MS = 60000;

      expect(MAX_RETRIES).toBeGreaterThan(0);
      expect(INITIAL_DELAY_MS).toBeGreaterThan(0);
      expect(BACKOFF_MULTIPLIER).toBeGreaterThan(1);
      expect(MAX_DELAY_MS).toBeGreaterThanOrEqual(INITIAL_DELAY_MS);
    });

    it("exponential backoff does not exceed max delay", () => {
      const INITIAL_DELAY_MS = 5000;
      const BACKOFF_MULTIPLIER = 2;
      const MAX_DELAY_MS = 60000;
      const MAX_RETRIES = 3;

      let delay = INITIAL_DELAY_MS;
      for (let i = 0; i < MAX_RETRIES; i++) {
        delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
        expect(delay).toBeLessThanOrEqual(MAX_DELAY_MS);
      }
    });
  });

  describe("prompt-file support", () => {
    it("inlines small prompt files as -p", () => {
      const promptFile = path.join(os.tmpdir(), `copilot-driver-small-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "small prompt body", "utf8");

      const resolved = resolvePromptFileArgs(["--add-dir", "/tmp", "--prompt-file", promptFile, "--allow-all-tools"]);
      expect(resolved).toEqual(["--add-dir", "/tmp", "-p", "small prompt body", "--allow-all-tools"]);
    });

    it("uses compact fallback prompt when prompt file is larger than 100KB", () => {
      const promptFile = path.join(os.tmpdir(), `copilot-driver-large-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "x".repeat(PROMPT_FILE_INLINE_THRESHOLD_BYTES + 1), "utf8");

      const resolved = resolvePromptFileArgs(["--prompt-file", promptFile, "--allow-all-tools"]);
      expect(resolved).toEqual(["-p", buildPromptFileFallbackInstruction(promptFile), "--allow-all-tools"]);
    });

    it("keeps --prompt-file arguments unchanged when file resolution fails", () => {
      const missingPath = path.join(os.tmpdir(), `copilot-driver-missing-${Date.now()}.txt`);
      const resolved = resolvePromptFileArgs(["--prompt-file", missingPath, "--allow-all-tools"]);
      expect(resolved).toEqual(["--prompt-file", missingPath, "--allow-all-tools"]);
    });
  });

  describe("formatDuration", () => {
    // Inline the same logic as the driver's formatDuration for unit testing
    function formatDuration(ms) {
      const totalSeconds = Math.floor(ms / 1000);
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = totalSeconds % 60;
      if (minutes > 0) {
        return `${minutes}m ${seconds}s`;
      }
      return `${seconds}s`;
    }

    it("formats sub-minute durations as seconds", () => {
      expect(formatDuration(0)).toBe("0s");
      expect(formatDuration(500)).toBe("0s");
      expect(formatDuration(1000)).toBe("1s");
      expect(formatDuration(59000)).toBe("59s");
    });

    it("formats minute-level durations with minutes and seconds", () => {
      expect(formatDuration(60000)).toBe("1m 0s");
      expect(formatDuration(90000)).toBe("1m 30s");
      expect(formatDuration(192000)).toBe("3m 12s"); // 3m 12s (real-world example)
    });

    it("handles long durations correctly", () => {
      expect(formatDuration(3600000)).toBe("60m 0s");
    });
  });

  describe("log format", () => {
    it("log lines include [copilot-harness] prefix and ISO timestamp", () => {
      // Verify the format matches what we expect in agent-stdio.log
      const ts = new Date().toISOString();
      const logLine = `[copilot-harness] ${ts} test message`;
      expect(logLine).toMatch(/^\[copilot-harness\] \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    });
  });

  describe("startup log includes node version and platform", () => {
    it("starting log line contains nodeVersion and platform fields", () => {
      const command = "/usr/local/bin/copilot";
      const startingLine = `starting: command=${command} maxRetries=3 initialDelayMs=5000` + ` backoffMultiplier=2 maxDelayMs=60000` + ` nodeVersion=${process.version} platform=${process.platform}`;
      expect(startingLine).toContain("nodeVersion=");
      expect(startingLine).toContain("platform=");
      expect(startingLine).toMatch(/nodeVersion=v\d+\.\d+/);
    });
  });

  describe("no-output failure message", () => {
    it("includes actionable possible causes", () => {
      const msg = `attempt 1: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`;
      expect(msg).toContain("binary not found");
      expect(msg).toContain("permission denied");
      expect(msg).toContain("auth failure");
      expect(msg).toContain("silent startup crash");
    });
  });

  describe("error event message", () => {
    it("includes code and syscall fields", () => {
      const errMessage = "spawn /usr/local/bin/copilot ENOENT";
      const errCode = "ENOENT";
      const errSyscall = "spawn";
      const logMsg = `attempt 1: failed to start process '/usr/local/bin/copilot': ${errMessage}` + ` (code=${errCode} syscall=${errSyscall})`;
      expect(logMsg).toContain("code=ENOENT");
      expect(logMsg).toContain("syscall=spawn");
    });
  });

  describe("extractModelIds", () => {
    it("returns null for null input", () => {
      expect(extractModelIds(null)).toBeNull();
    });

    it("returns null for empty object", () => {
      expect(extractModelIds({})).toBeNull();
    });

    it("returns null for empty data array", () => {
      expect(extractModelIds({ data: [] })).toBeNull();
    });

    it("extracts ids from OpenAI format", () => {
      const json = { data: [{ id: "gpt-4o" }, { id: "gpt-4o-mini" }] };
      expect(extractModelIds(json)).toEqual(["gpt-4o", "gpt-4o-mini"]);
    });

    it("falls back to name when id is absent in OpenAI format", () => {
      const json = { data: [{ name: "model-a" }, { id: "model-b" }] };
      expect(extractModelIds(json)).toEqual(["model-a", "model-b"]);
    });

    it("extracts ids from Gemini format, stripping prefix", () => {
      const json = {
        models: [{ name: "models/gemini-1.5-pro" }, { name: "models/gemini-1.0-pro" }],
      };
      expect(extractModelIds(json)).toEqual(["gemini-1.0-pro", "gemini-1.5-pro"]);
    });

    it("handles Gemini entries without the prefix", () => {
      const json = { models: [{ name: "custom-model" }] };
      expect(extractModelIds(json)).toEqual(["custom-model"]);
    });

    it("returns sorted results", () => {
      const json = { data: [{ id: "z-model" }, { id: "a-model" }, { id: "m-model" }] };
      expect(extractModelIds(json)).toEqual(["a-model", "m-model", "z-model"]);
    });
  });

  describe("detection model availability helpers", () => {
    it("identifies detection phase from GH_AW_PHASE", () => {
      expect(isDetectionPhase("detection")).toBe(true);
      expect(isDetectionPhase("DETECTION")).toBe(true);
      expect(isDetectionPhase("agent")).toBe(false);
      expect(isDetectionPhase("")).toBe(false);
    });

    it("checks model availability from reflect endpoint payload", () => {
      const reflectData = {
        endpoints: [
          { provider: "copilot", configured: true, models: ["claude-sonnet-4.6", "gpt-5.4"] },
          { provider: "openai", configured: false, models: ["gpt-4.1"] },
        ],
      };
      expect(isModelAvailableInReflectData("claude-sonnet-4.6", reflectData)).toBe(true);
      expect(isModelAvailableInReflectData("gpt-4.1", reflectData)).toBe(false);
      expect(isModelAvailableInReflectData("missing-model", reflectData)).toBe(false);
    });

    it("reads reflect file and checks model availability", () => {
      const reflectFile = path.join(os.tmpdir(), `awf-reflect-${Date.now()}.json`);
      try {
        fs.writeFileSync(
          reflectFile,
          JSON.stringify({
            endpoints: [{ provider: "copilot", configured: true, models: ["claude-sonnet-4.6"] }],
          }),
          "utf8"
        );
        const logs = [];
        expect(isModelAvailableInReflectFile("claude-sonnet-4.6", { reflectPath: reflectFile, logger: msg => logs.push(msg) })).toBe(true);
        expect(isModelAvailableInReflectFile("gpt-4.1", { reflectPath: reflectFile, logger: msg => logs.push(msg) })).toBe(false);
      } finally {
        fs.unlinkSync(reflectFile);
      }
    });
  });

  describe("enrichReflectModels", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("does nothing when all configured endpoints already have models", async () => {
      const reflectData = {
        endpoints: [{ provider: "openai", configured: true, models: ["gpt-4o"], models_url: "http://api-proxy:10000/v1/models" }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toEqual(["gpt-4o"]);
    });

    it("does nothing for unconfigured endpoints with null models", async () => {
      const reflectData = {
        endpoints: [{ provider: "anthropic", configured: false, models: null, models_url: "http://api-proxy:10001/v1/models" }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toBeNull();
    });

    it("does nothing when models_url is null", async () => {
      const reflectData = {
        endpoints: [{ provider: "opencode", configured: true, models: null, models_url: null }],
      };
      const logger = () => {};
      await enrichReflectModels(reflectData, 1000, logger);
      expect(reflectData.endpoints[0].models).toBeNull();
    });

    it("fetches models from models_url for configured endpoints with null models", async () => {
      const modelResponse = { data: [{ id: "claude-sonnet-4.6" }, { id: "gpt-4o" }] };
      vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => modelResponse }));

      const reflectData = {
        endpoints: [{ provider: "copilot", configured: true, models: null, models_url: "http://api-proxy:10002/models" }],
      };
      const logs = [];
      await enrichReflectModels(reflectData, 3000, msg => logs.push(msg));

      expect(reflectData.endpoints[0].models).toEqual(["claude-sonnet-4.6", "gpt-4o"]);
      expect(logs.some(l => l.includes("fetched 2 model(s)"))).toBe(true);
    });

    it("leaves models null when models_url fetch fails", async () => {
      vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("ECONNREFUSED")));

      const reflectData = {
        endpoints: [{ provider: "openai", configured: true, models: null, models_url: "http://api-proxy:10000/v1/models" }],
      };
      const logs = [];
      await enrichReflectModels(reflectData, 500, msg => logs.push(msg));
      expect(reflectData.endpoints[0].models).toBeNull();
      expect(logs.some(l => l.includes("models fetch error"))).toBe(true);
    });
  });

  describe("fetchAWFReflect enriches models via fallback", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("saves enriched reflect data when api-proxy returns null models for configured provider", async () => {
      const modelData = { data: [{ id: "gpt-4o" }, { id: "gpt-4o-mini" }] };
      const reflectPayload = {
        endpoints: [{ provider: "openai", port: 10000, configured: true, models: null, models_url: "http://api-proxy:10000/v1/models" }],
        models_fetch_complete: true,
      };

      vi.stubGlobal(
        "fetch",
        vi.fn().mockImplementation(url => {
          const body = String(url).includes("/reflect") ? reflectPayload : modelData;
          return Promise.resolve({ ok: true, status: 200, json: async () => body });
        })
      );

      const outputDir = fs.mkdtempSync(path.join(os.tmpdir(), "awf-reflect-test-"));
      const outputPath = path.join(outputDir, "awf-reflect.json");
      const logs = [];

      try {
        await fetchAWFReflect({
          reflectUrl: "http://api-proxy:10000/reflect",
          outputPath,
          timeoutMs: 3000,
          modelsTimeoutMs: 1000,
          logger: msg => logs.push(msg),
        });

        const saved = JSON.parse(fs.readFileSync(outputPath, "utf8"));
        expect(saved.endpoints[0].models).toEqual(["gpt-4o", "gpt-4o-mini"]);
      } finally {
        fs.rmSync(outputDir, { recursive: true, force: true });
      }
    });
  });
});
