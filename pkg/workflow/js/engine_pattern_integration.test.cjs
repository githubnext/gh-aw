import { describe, it, expect, beforeEach, vi } from "vitest";

const { validateErrors } = await import("./validate_errors.cjs");

// Mock global objects for testing
global.console = {
  log: vi.fn(),
  warn: vi.fn(),
  debug: vi.fn(),
};

global.core = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn(() => ({ write: vi.fn() })),
    write: vi.fn().mockResolvedValue(),
  }
};

describe("Engine Pattern Integration Tests", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("CodexEngine Patterns", () => {
    const codexPatterns = [
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp"
      },
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex ERROR messages with timestamp"
      },
      {
        pattern: "(?i)unauthorized",
        level_group: 0,
        message_group: 0,
        description: "Unauthorized access error"
      },
      {
        pattern: "(?i)permission.*denied",
        level_group: 0,
        message_group: 0,
        description: "Generic permission denied error"
      }
    ];

    it("should detect Codex stream errors", () => {
      const logContent = `[2025-01-10T12:34:56] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…
[2025-01-10T12:34:58] stream error: connection timeout after 5 seconds`;

      const hasErrors = validateErrors(logContent, codexPatterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(2);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("exceeded retry limit"));
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("connection timeout"));
    });

    it("should detect Codex ERROR messages", () => {
      const logContent = `[2025-01-10T12:34:56] ERROR: Authentication failed
[2025-01-10T12:34:58] WARNING: Rate limit approaching`;

      // Add the WARNING pattern that matches the test data
      const patterns = [
        ...codexPatterns,
        {
          pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(WARN|WARNING):\\s+(.+)",
          level_group: 2,
          message_group: 3,
          description: "Codex warning messages with timestamp"
        }
      ];

      const hasErrors = validateErrors(logContent, patterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(1);
      expect(global.core.warning).toHaveBeenCalledTimes(1);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Authentication failed"));
    });

    it("should detect case-insensitive permission errors", () => {
      const logContent = `Unauthorized access to resource
PERMISSION DENIED for user
Access is forbidden`;

      // Add forbidden pattern to match all three test strings
      const patterns = [
        ...codexPatterns,
        {
          pattern: "(?i)forbidden",
          level_group: 0,
          message_group: 0,
          description: "Forbidden access error"
        }
      ];

      const hasErrors = validateErrors(logContent, patterns);
      
      // Permission patterns with level_group: 0 get classified as "unknown" level,
      // which are treated as warnings, not errors
      expect(hasErrors).toBe(false);
      expect(global.core.warning).toHaveBeenCalledTimes(3);
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Unauthorized"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("PERMISSION DENIED"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("forbidden"));
    });
  });

  describe("CopilotEngine Patterns", () => {
    const copilotPatterns = [
      {
        pattern: "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(ERROR)\\]\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Copilot CLI timestamped ERROR messages"
      },
      {
        pattern: "(Error):\\s+(.+)",
        level_group: 1,
        message_group: 2,
        description: "Generic error messages from Copilot CLI or Node.js"
      },
      {
        pattern: "npm ERR!\\s+(.+)",
        level_group: 0,
        message_group: 1,
        description: "NPM error messages during Copilot CLI installation or execution"
      },
      {
        pattern: "(?i)unauthorized",
        level_group: 0,
        message_group: 0,
        description: "Unauthorized access error"
      }
    ];

    it("should detect Copilot CLI timestamped errors", () => {
      const logContent = `2025-01-10T12:34:56.789Z [ERROR] Failed to authenticate with GitHub
2025-01-10T12:34:57.123Z [WARNING] Retrying connection...`;

      const hasErrors = validateErrors(logContent, copilotPatterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(1);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Failed to authenticate"));
    });

    it("should detect generic error messages", () => {
      const logContent = `Error: Cannot find module '@github/copilot'
Warning: Deprecated feature used`;

      const hasErrors = validateErrors(logContent, copilotPatterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(1);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Cannot find module"));
    });

    it("should detect NPM errors", () => {
      const logContent = `npm ERR! code EACCES
npm ERR! Permission denied
npm WARN deprecated package@1.0.0`;

      const hasErrors = validateErrors(logContent, copilotPatterns);
      
      // NPM ERR! patterns don't contain "error" in the match text, so they're treated as warnings
      // The pattern "npm ERR!" doesn't match the level inference which looks for "error" or "warn"
      expect(hasErrors).toBe(false);
      expect(global.core.warning).toHaveBeenCalledTimes(2);
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("code EACCES"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Permission denied"));
    });
  });

  describe("ClaudeEngine Patterns", () => {
    const claudePatterns = [
      {
        pattern: "(?i)access denied.*user.*not authorized",
        level_group: 0,
        message_group: 0,
        description: "Permission denied - user not authorized"
      },
      {
        pattern: "(?i)repository permission check failed",
        level_group: 0,
        message_group: 0,
        description: "Repository permission check failure"
      },
      {
        pattern: "(?i)forbidden",
        level_group: 0,
        message_group: 0,
        description: "Forbidden access error"
      }
    ];

    it("should detect Claude permission errors", () => {
      const logContent = `Access denied - user testuser not authorized for this action
Repository permission check failed for write access
Forbidden: Cannot modify protected branch`;

      const hasErrors = validateErrors(logContent, claudePatterns);
      
      // Claude permission patterns with level_group: 0 get classified as "unknown" level,
      // which are treated as warnings, not errors
      expect(hasErrors).toBe(false);
      expect(global.core.warning).toHaveBeenCalledTimes(3);
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("user testuser not authorized"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("permission check failed"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Forbidden"));
    });
  });

  describe("Mixed Engine Scenarios", () => {
    const allEnginePatterns = [
      // Codex patterns
      {
        pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Codex stream errors with timestamp"
      },
      // Copilot patterns
      {
        pattern: "(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d{3}Z)\\s+\\[(ERROR)\\]\\s+(.+)",
        level_group: 2,
        message_group: 3,
        description: "Copilot CLI timestamped ERROR messages"
      },
      // Claude patterns
      {
        pattern: "(?i)unauthorized",
        level_group: 0,
        message_group: 0,
        description: "Unauthorized access error"
      }
    ];

    it("should handle logs with mixed engine error formats", () => {
      const logContent = `[2025-01-10T12:34:56] stream error: connection failed
2025-01-10T12:34:57.123Z [ERROR] Authentication required
Unauthorized access to protected resource`;

      const hasErrors = validateErrors(logContent, allEnginePatterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(2);
      expect(global.core.warning).toHaveBeenCalledTimes(1);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("connection failed"));
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Authentication required"));
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Unauthorized access"));
    });
  });

  describe("Real-world Error Scenarios", () => {
    it("should handle the GitHub issue #668 401 Unauthorized scenario", () => {
      const patterns = [
        {
          pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+stream\\s+(error):\\s+(.+)",
          level_group: 2,
          message_group: 3,
          description: "Codex stream errors with timestamp"
        }
      ];

      const logContent = `[2025-09-10T17:54:49] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 1/5 in 216ms…
[2025-09-10T17:54:54] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 2/5 in 414ms…
[2025-09-10T17:54:58] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 3/5 in 821ms…
[2025-09-10T17:55:03] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 4/5 in 1.611s…
[2025-09-10T17:55:08] stream error: exceeded retry limit, last status: 401 Unauthorized; retrying 5/5 in 3.055s…`;

      const hasErrors = validateErrors(logContent, patterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(5);
      
      // Verify each error call contains the expected content
      for (let i = 0; i < 5; i++) {
        expect(global.core.error).toHaveBeenNthCalledWith(
          i + 1,
          expect.stringContaining("401 Unauthorized")
        );
        expect(global.core.error).toHaveBeenNthCalledWith(
          i + 1,
          expect.stringContaining("exceeded retry limit")
        );
      }
    });

    it("should correctly extract level and message groups", () => {
      const patterns = [
        {
          pattern: "\\[(\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2})\\]\\s+(ERROR):\\s+(.+)",
          level_group: 2,
          message_group: 3,
          description: "Codex ERROR messages with timestamp"
        }
      ];

      const logContent = `[2025-01-10T12:34:56] ERROR: Authentication failed with status 401`;

      const hasErrors = validateErrors(logContent, patterns);
      
      expect(hasErrors).toBe(true);
      expect(global.core.error).toHaveBeenCalledTimes(1);
      expect(global.core.error).toHaveBeenCalledWith(
        expect.stringContaining("Authentication failed with status 401")
      );
      expect(global.core.error).toHaveBeenCalledWith(
        expect.stringContaining("Pattern: Codex ERROR messages with timestamp")
      );
    });
  });
});