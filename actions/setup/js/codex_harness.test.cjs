import { describe, it, expect } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const {
  resolveCodexPromptFileArgs,
  injectJsonFlag,
  isRateLimitError,
  isMaxEffectiveTokensExceededError,
  isAuthenticationFailedError,
  isMissingApiKeyError,
  isServerError,
  countPermissionDeniedIssues,
  hasNumerousPermissionDeniedIssues,
  extractDeniedCommands,
  buildMissingToolPermissionIssuePayload,
  buildCodexChildEnv,
  extractPortFromURL,
  extractOpenAIProxyBaseURLFromToml,
  getConfiguredOpenAIPortFromReflect,
  validateCodexOpenAIBaseURLFromReflect,
} = require("./codex_harness.cjs");

describe("codex_harness.cjs", () => {
  describe("resolveCodexPromptFileArgs", () => {
    it("replaces --prompt-file with the file's content as the last positional arg", () => {
      const promptFile = path.join(os.tmpdir(), `codex-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "fix the bug", "utf8");
      try {
        const result = resolveCodexPromptFileArgs(["exec", "--dangerously-bypass-approvals-and-sandbox", "--prompt-file", promptFile]);
        expect(result).toEqual(["exec", "--dangerously-bypass-approvals-and-sandbox", "fix the bug"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("appends prompt content as the last arg when only --prompt-file is provided", () => {
      const promptFile = path.join(os.tmpdir(), `codex-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "my task", "utf8");
      try {
        const result = resolveCodexPromptFileArgs(["--prompt-file", promptFile]);
        expect(result).toEqual(["my task"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("passes through args that have no --prompt-file", () => {
      const result = resolveCodexPromptFileArgs(["exec", "--dangerously-bypass-approvals-and-sandbox"]);
      expect(result).toEqual(["exec", "--dangerously-bypass-approvals-and-sandbox"]);
    });

    it("preserves args when --prompt-file is provided without a path", () => {
      const result = resolveCodexPromptFileArgs(["exec", "--prompt-file"]);
      // When no path follows --prompt-file, it is preserved as-is
      expect(result).toEqual(["exec", "--prompt-file"]);
    });

    it("throws when the prompt file does not exist", () => {
      const missingFile = path.join(os.tmpdir(), `codex-harness-missing-${Date.now()}.txt`);
      expect(() => resolveCodexPromptFileArgs(["--prompt-file", missingFile])).toThrow(`--prompt-file '${missingFile}' is not readable`);
    });

    it("throws when the prompt file cannot be read (directory)", () => {
      const dir = fs.mkdtempSync(path.join(os.tmpdir(), "codex-harness-dir-"));
      try {
        expect(() => resolveCodexPromptFileArgs(["--prompt-file", dir])).toThrow(`--prompt-file '${dir}' is not readable`);
      } finally {
        fs.rmdirSync(dir);
      }
    });
  });

  describe("isRateLimitError", () => {
    it("returns true for rate_limit_exceeded error", () => {
      expect(isRateLimitError("Error: rate_limit_exceeded")).toBe(true);
    });

    describe("isMaxEffectiveTokensExceededError", () => {
      it("returns true for AWF effective-token hard rails", () => {
        expect(isMaxEffectiveTokensExceededError("CAPIError: 429 Maximum effective tokens exceeded (25296477.30 / 25000000).")).toBe(true);
      });

      it("returns false for ordinary rate-limit output", () => {
        expect(isMaxEffectiveTokensExceededError("RateLimitError: You exceeded your current quota")).toBe(false);
      });
    });

    it("returns true for 429 Too Many Requests", () => {
      expect(isRateLimitError("429 Too Many Requests")).toBe(true);
    });

    it("returns true for RateLimitError", () => {
      expect(isRateLimitError("RateLimitError: You exceeded your current quota")).toBe(true);
    });

    it("returns false for unrelated errors", () => {
      expect(isRateLimitError("Error: ENOENT: no such file")).toBe(false);
      expect(isRateLimitError("Fatal: out of memory")).toBe(false);
      expect(isRateLimitError("")).toBe(false);
    });

    it("returns false for a 500 server error", () => {
      expect(isRateLimitError("500 Internal Server Error")).toBe(false);
    });
  });

  describe("isAuthenticationFailedError", () => {
    it("returns true for authentication failed with request id", () => {
      expect(isAuthenticationFailedError("Authentication failed (Request ID: C818:3ED713:19D401B:1C446B7:69D653CA)")).toBe(true);
    });

    it("returns false for non-authentication-failed output", () => {
      expect(isAuthenticationFailedError("No authentication information found")).toBe(false);
      expect(isAuthenticationFailedError("rate_limit_exceeded")).toBe(false);
    });
  });

  describe("isMissingApiKeyError", () => {
    it("returns true for missing OPENAI_API_KEY with backtick delimiters", () => {
      expect(isMissingApiKeyError("ERROR: Missing environment variable: `OPENAI_API_KEY`")).toBe(true);
    });

    it("returns true for missing CODEX_API_KEY with backtick delimiters", () => {
      expect(isMissingApiKeyError("ERROR: Missing environment variable: `CODEX_API_KEY`")).toBe(true);
    });

    it("returns true for missing OPENAI_API_KEY without backtick delimiters", () => {
      expect(isMissingApiKeyError("Missing environment variable: OPENAI_API_KEY")).toBe(true);
    });

    it("returns true when the error appears within a larger output block", () => {
      const output = "Starting codex...\nERROR: Missing environment variable: `OPENAI_API_KEY`\nExiting.";
      expect(isMissingApiKeyError(output)).toBe(true);
    });

    it("returns false for unrelated errors", () => {
      expect(isMissingApiKeyError("Authentication failed")).toBe(false);
      expect(isMissingApiKeyError("rate_limit_exceeded")).toBe(false);
      expect(isMissingApiKeyError("Missing environment variable: HOME")).toBe(false);
      expect(isMissingApiKeyError("")).toBe(false);
    });
  });

  describe("injectJsonFlag", () => {
    it("injects --json after exec when not already present", () => {
      expect(injectJsonFlag(["exec", "--dangerously-bypass-approvals-and-sandbox", "do the thing"])).toEqual(["exec", "--json", "--dangerously-bypass-approvals-and-sandbox", "do the thing"]);
    });

    it("does not inject --json when already present", () => {
      expect(injectJsonFlag(["exec", "--json", "--skip-git-repo-check", "do the thing"])).toEqual(["exec", "--json", "--skip-git-repo-check", "do the thing"]);
    });

    it("does not inject --json for non-exec subcommands", () => {
      expect(injectJsonFlag(["resume", "--last", "fix it"])).toEqual(["resume", "--last", "fix it"]);
    });

    it("returns empty array unchanged", () => {
      expect(injectJsonFlag([])).toEqual([]);
    });
  });

  describe("buildCodexChildEnv", () => {
    it("preserves captured keys even when base environment is missing them", () => {
      const result = buildCodexChildEnv({ PATH: "/usr/bin" }, "codex-key", "openai-key");
      expect(result.CODEX_API_KEY).toBe("codex-key");
      expect(result.OPENAI_API_KEY).toBe("openai-key");
      expect(result.PATH).toBe("/usr/bin");
    });

    it("does not add unset keys", () => {
      const result = buildCodexChildEnv({ PATH: "/usr/bin" }, undefined, undefined);
      expect(result.CODEX_API_KEY).toBeUndefined();
      expect(result.OPENAI_API_KEY).toBeUndefined();
    });
  });

  describe("OpenAI base URL validation", () => {
    it("extracts port from URL", () => {
      expect(extractPortFromURL("http://172.30.0.30:10000")).toBe(10000);
      expect(extractPortFromURL("https://example.com")).toBeNull();
      expect(extractPortFromURL("not-a-url")).toBeNull();
    });

    it("extracts openai-proxy base_url from TOML", () => {
      const toml = `
[history]
persistence = "none"
[model_providers.openai-proxy]
name = "OpenAI AWF proxy"
base_url = "http://172.30.0.30:10000"
env_key = "OPENAI_API_KEY"
`;
      expect(extractOpenAIProxyBaseURLFromToml(toml)).toBe("http://172.30.0.30:10000");
    });

    it("extracts configured OpenAI port from reflect payload", () => {
      const reflect = {
        endpoints: [
          { provider: "anthropic", port: 10001, configured: true },
          { provider: "openai", port: 10000, configured: true },
        ],
      };
      expect(getConfiguredOpenAIPortFromReflect(reflect)).toBe(10000);
    });

    it("returns null for malformed reflect endpoint ports", () => {
      const reflect = {
        endpoints: [{ provider: "openai", port: "not-a-number", configured: true }],
      };
      expect(getConfiguredOpenAIPortFromReflect(reflect)).toBeNull();
    });

    it("fails validation when config and reflect OpenAI ports mismatch", () => {
      const toml = `[model_providers.openai-proxy]\nbase_url = "http://172.30.0.30:10001"\n`;
      const reflect = JSON.stringify({
        endpoints: [
          { provider: "openai", port: 10000, configured: true },
          { provider: "anthropic", port: 10001, configured: true },
        ],
      });
      const files = {
        "/tmp/codex-config.toml": toml,
        "/tmp/awf-reflect.json": reflect,
      };
      const readFileSync = filePath => files[filePath];
      const result = validateCodexOpenAIBaseURLFromReflect({
        codexConfigPath: "/tmp/codex-config.toml",
        reflectPath: "/tmp/awf-reflect.json",
        readFileSync,
      });
      expect(result.ok).toBe(false);
      expect(result.reason).toContain("mismatch");
    });

    it("passes validation when ports match", () => {
      const toml = `[model_providers.openai-proxy]\nbase_url = "http://172.30.0.30:10000"\n`;
      const reflect = JSON.stringify({
        endpoints: [{ provider: "openai", port: 10000, configured: true }],
      });
      const files = {
        "/tmp/codex-config.toml": toml,
        "/tmp/awf-reflect.json": reflect,
      };
      const readFileSync = filePath => files[filePath];
      const result = validateCodexOpenAIBaseURLFromReflect({
        codexConfigPath: "/tmp/codex-config.toml",
        reflectPath: "/tmp/awf-reflect.json",
        readFileSync,
      });
      expect(result.ok).toBe(true);
    });

    it("passes through when TOML lacks openai-proxy section", () => {
      const files = {
        "/tmp/codex-config.toml": `[history]\npersistence = "none"\n`,
        "/tmp/awf-reflect.json": JSON.stringify({ endpoints: [{ provider: "openai", port: 10000, configured: true }] }),
      };
      const readFileSync = filePath => files[filePath];
      const result = validateCodexOpenAIBaseURLFromReflect({
        codexConfigPath: "/tmp/codex-config.toml",
        reflectPath: "/tmp/awf-reflect.json",
        readFileSync,
      });
      expect(result.ok).toBe(true);
    });
  });

  describe("isServerError", () => {
    it("returns true for InternalServerError", () => {
      expect(isServerError("InternalServerError: The server had an error processing your request")).toBe(true);
    });

    it("returns true for ServiceUnavailableError", () => {
      expect(isServerError("ServiceUnavailableError: The server is temporarily unable to service your request")).toBe(true);
    });

    it("returns true for 500 Internal Server Error", () => {
      expect(isServerError("500 Internal Server Error")).toBe(true);
    });

    it("returns true for 503 Service Unavailable", () => {
      expect(isServerError("503 Service Unavailable")).toBe(true);
    });

    it("returns false for rate limit errors", () => {
      expect(isServerError("rate_limit_exceeded")).toBe(false);
      expect(isServerError("429 Too Many Requests")).toBe(false);
    });

    it("returns false for unrelated errors", () => {
      expect(isServerError("Error: ENOENT: no such file")).toBe(false);
      expect(isServerError("")).toBe(false);
    });
  });

  describe("permission-denied classification helpers", () => {
    it("counts repeated permission-denied signals", () => {
      const output = "permission denied\npermissions denied\nEACCES: permission denied";
      expect(countPermissionDeniedIssues(output)).toBe(4);
    });

    it("detects numerous permission-denied issues at threshold", () => {
      const output = "permission denied\npermission denied\npermission denied";
      expect(hasNumerousPermissionDeniedIssues(output)).toBe(true);
    });

    it("does not classify sparse permission-denied output as numerous", () => {
      expect(hasNumerousPermissionDeniedIssues("permission denied")).toBe(false);
    });

    it("builds missing_tool payload for permission issues", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload());
      expect(payload.type).toBe("missing_tool");
      expect(payload.reason).toContain("missing tool/permission issue");
      expect(payload.denied_commands).toEqual([]);
    });

    it("builds missing_tool payload with denied commands", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload(["go version", "ls /usr/local/go"]));
      expect(payload.type).toBe("missing_tool");
      expect(payload.denied_commands).toEqual(["go version", "ls /usr/local/go"]);
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

  describe("retry policy: fresh run on partial execution", () => {
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      const RATE_LIMIT_ERROR_PATTERN = /rate_limit_exceeded|429 Too Many Requests|RateLimitError/i;
      const SERVER_ERROR_PATTERN = /InternalServerError|ServiceUnavailableError|500 Internal Server Error|503 Service Unavailable/i;
      if (attempt === 0 && isAuthenticationFailedError(result.output)) return false;
      if (isMissingApiKeyError(result.output)) return false;
      if (hasNumerousPermissionDeniedIssues(result.output)) return false;
      if (isMaxEffectiveTokensExceededError(result.output)) return false;
      const isTransient = RATE_LIMIT_ERROR_PATTERN.test(result.output) || SERVER_ERROR_PATTERN.test(result.output);
      return attempt < MAX_RETRIES && (result.hasOutput || isTransient);
    }

    it("retries on rate limit error even without output", () => {
      const result = { exitCode: 1, hasOutput: false, output: "rate_limit_exceeded" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("retries on server error even without output", () => {
      const result = { exitCode: 1, hasOutput: false, output: "InternalServerError" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("retries on any other non-zero exit when session produced output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: connection reset" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("does not retry when first attempt fails authentication", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Authentication failed (Request ID: ABC123)" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry when missing API key is detected (any attempt)", () => {
      const result = { exitCode: 1, hasOutput: false, output: "ERROR: Missing environment variable: `OPENAI_API_KEY`" };
      expect(shouldRetry(result, 0)).toBe(false);
      expect(shouldRetry(result, 1)).toBe(false);
    });

    it("does not retry when no output was produced and no transient error", () => {
      const result = { exitCode: 1, hasOutput: false, output: "" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry after retries are exhausted", () => {
      const result = { exitCode: 1, hasOutput: true, output: "rate_limit_exceeded" };
      expect(shouldRetry(result, MAX_RETRIES)).toBe(false);
    });

    it("does not retry on success", () => {
      const result = { exitCode: 0, hasOutput: true, output: "Task complete" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry when numerous permission-denied issues are present", () => {
      const result = { exitCode: 1, hasOutput: true, output: "permission denied\npermission denied\npermission denied" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry maximum effective tokens exceeded hard rails", () => {
      const result = {
        exitCode: 1,
        hasOutput: true,
        output: "CAPIError: 429 Maximum effective tokens exceeded (25296477.30 / 25000000).",
      };
      expect(shouldRetry(result, 0)).toBe(false);
    });
  });
});
