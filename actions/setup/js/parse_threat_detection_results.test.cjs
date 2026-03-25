import { describe, it, expect, beforeEach, vi } from "vitest";

// Create mock functions
const mockExistsSync = vi.fn(() => false);
const mockReadFileSync = vi.fn(() => "");

// Mock fs module
vi.mock("fs", () => {
  return {
    existsSync: mockExistsSync,
    readFileSync: mockReadFileSync,
    default: {
      existsSync: mockExistsSync,
      readFileSync: mockReadFileSync,
    },
  };
});

// Mock file_helpers to avoid transitive fs issues
vi.mock("./file_helpers.cjs", () => {
  return {
    listFilesRecursively: vi.fn(() => []),
  };
});

// Mock the global core object
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};
global.core = mockCore;

const { parseDetectionLog, extractFromStreamJson } = require("./parse_threat_detection_results.cjs");

describe("extractFromStreamJson", () => {
  it("should extract result from type:result JSON envelope", () => {
    const line = '{"type":"result","subtype":"success","is_error":false,"result":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}","stop_reason":"end_turn"}';
    const result = extractFromStreamJson(line);
    expect(result).toContain("THREAT_DETECTION_RESULT:");
  });

  it("should extract result when analysis text precedes the verdict line", () => {
    // The model may include explanatory text before THREAT_DETECTION_RESULT in the result field
    const line =
      '{"type":"result","subtype":"success","result":"**Analysis complete.**\\n\\nNo threats found.\\n\\nTHREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}","stop_reason":"end_turn"}';
    const result = extractFromStreamJson(line);
    // Ensure we extracted only the verdict line, not the preceding analysis text
    expect(result).toMatch(/^THREAT_DETECTION_RESULT:/);
    expect(result).not.toContain("**Analysis complete.**");
  });

  it("should allow parseDetectionLog to parse extracted verdict when analysis text precedes it", () => {
    const line =
      '{"type":"result","subtype":"success","result":"**Analysis complete.**\\n\\nNo threats found.\\n\\nTHREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}","stop_reason":"end_turn"}';
    const extracted = extractFromStreamJson(line);
    expect(extracted).not.toBeNull();
    const { verdict, error } = parseDetectionLog(extracted);
    expect(error).toBeUndefined();
    expect(verdict).toEqual({
      prompt_injection: false,
      secret_leak: false,
      malicious_patch: false,
      reasons: [],
    });
  });

  it("should return null for type:assistant JSON (not authoritative)", () => {
    const line = '{"type":"assistant","message":{"content":[{"type":"text","text":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false}"}]}}';
    const result = extractFromStreamJson(line);
    expect(result).toBeNull();
  });

  it("should return null for non-JSON lines", () => {
    expect(extractFromStreamJson("just plain text")).toBeNull();
    expect(extractFromStreamJson("THREAT_DETECTION_RESULT:{...}")).toBeNull();
  });

  it("should return null for JSON without result field", () => {
    const line = '{"type":"result","subtype":"success"}';
    expect(extractFromStreamJson(line)).toBeNull();
  });

  it("should return null for type:result where result does not start with prefix", () => {
    const line = '{"type":"result","result":"some other output"}';
    expect(extractFromStreamJson(line)).toBeNull();
  });

  it("should return null for malformed JSON", () => {
    expect(extractFromStreamJson("{not valid json}")).toBeNull();
  });
});

describe("parseDetectionLog", () => {
  describe("valid results", () => {
    it("should parse a clean verdict (no threats)", () => {
      const content = 'Some debug output\nTHREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}\nMore output';
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict).toEqual({
        prompt_injection: false,
        secret_leak: false,
        malicious_patch: false,
        reasons: [],
      });
    });

    it("should parse a verdict with threats detected", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":true,"reasons":["found backdoor","injected prompt"]}';
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict.prompt_injection).toBe(true);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(true);
      expect(verdict.reasons).toEqual(["found backdoor", "injected prompt"]);
    });

    it("should handle leading/trailing whitespace on the result line", () => {
      const content = '  THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}  ';
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict.prompt_injection).toBe(false);
    });

    it("should reject non-boolean values with a type error", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":1,"secret_leak":0,"malicious_patch":"yes","reasons":"not-array"}';
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain('Invalid type for "prompt_injection"');
      expect(error).toContain("expected boolean");
    });

    it('should reject string "false" as non-boolean', () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":"false","secret_leak":false,"malicious_patch":false,"reasons":[]}';
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain('Invalid type for "prompt_injection"');
      expect(error).toContain("got string");
    });

    it('should reject string "true" as non-boolean', () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":"true","secret_leak":false,"malicious_patch":false,"reasons":[]}';
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain('Invalid type for "prompt_injection"');
    });
  });

  describe("no result line", () => {
    it("should return error when no THREAT_DETECTION_RESULT line exists", () => {
      const content = "Some debug output\nNo result here\nMore output";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("No THREAT_DETECTION_RESULT found");
    });

    it("should return error for empty content", () => {
      const { verdict, error } = parseDetectionLog("");

      expect(verdict).toBeUndefined();
      expect(error).toContain("No THREAT_DETECTION_RESULT found");
    });

    it("should return error when content has only whitespace", () => {
      const { verdict, error } = parseDetectionLog("   \n  \n  ");

      expect(verdict).toBeUndefined();
      expect(error).toContain("No THREAT_DETECTION_RESULT found");
    });
  });

  describe("multiple result lines", () => {
    it("should deduplicate identical THREAT_DETECTION_RESULT lines", () => {
      // --debug-file and tee both write to the same file, causing duplicates
      const content = [
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict).toEqual({
        prompt_injection: false,
        secret_leak: false,
        malicious_patch: false,
        reasons: [],
      });
    });

    it("should error when conflicting THREAT_DETECTION_RESULT lines found", () => {
      const content = [
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":["injection"]}',
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("Multiple conflicting THREAT_DETECTION_RESULT entries");
    });

    it("should include unique lines in error for debugging", () => {
      const content = [
        'THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
        "some other output",
        'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":[]}',
      ].join("\n");
      const { error } = parseDetectionLog(content);

      expect(error).toContain("[1]");
      expect(error).toContain("[2]");
    });
  });

  describe("invalid JSON", () => {
    it("should return error when JSON is malformed", () => {
      const content = "THREAT_DETECTION_RESULT:{not valid json}";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("Failed to parse JSON from THREAT_DETECTION_RESULT");
      expect(error).toContain("Raw value:");
    });

    it("should return error when JSON is empty", () => {
      const content = "THREAT_DETECTION_RESULT:";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("Failed to parse JSON");
    });

    it("should return error when JSON is truncated", () => {
      const content = 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":';
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("Failed to parse JSON");
    });

    it("should return error when JSON is null", () => {
      const content = "THREAT_DETECTION_RESULT:null";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("must be an object");
      expect(error).toContain("got null");
    });

    it("should return error when JSON is an array", () => {
      const content = "THREAT_DETECTION_RESULT:[]";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("must be an object");
      expect(error).toContain("got array");
    });

    it("should return error when JSON is a string", () => {
      const content = 'THREAT_DETECTION_RESULT:"clean"';
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("must be an object");
      expect(error).toContain("got string");
    });

    it("should return error when JSON is a number", () => {
      const content = "THREAT_DETECTION_RESULT:42";
      const { verdict, error } = parseDetectionLog(content);

      expect(verdict).toBeUndefined();
      expect(error).toContain("must be an object");
      expect(error).toContain("got number");
    });
  });

  describe("stream-json format (--output-format stream-json)", () => {
    it("should extract result from type:result JSON envelope", () => {
      const content = [
        "2026-03-23T00:04:39.809Z [DEBUG] Fast mode unavailable",
        '{"type":"assistant","message":{"model":"claude-sonnet-4-6","content":[{"type":"text","text":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}"}]}}',
        '{"type":"result","subtype":"success","is_error":false,"result":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}","stop_reason":"end_turn"}',
        "2026-03-23T00:04:42.251Z [DEBUG] LSP server manager shut down successfully",
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict).toEqual({
        prompt_injection: false,
        secret_leak: false,
        malicious_patch: false,
        reasons: [],
      });
    });

    it("should extract threats from stream-json format", () => {
      const content = [
        '{"type":"result","subtype":"success","result":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":true,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[\\"Injected JSON payload in prompt.txt\\"]}","stop_reason":"end_turn"}',
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict.prompt_injection).toBe(true);
      expect(verdict.reasons).toEqual(["Injected JSON payload in prompt.txt"]);
    });

    it("should not double-count from assistant and result envelopes", () => {
      // Both assistant and result contain the same THREAT_DETECTION_RESULT
      // The parser should only extract from type:result (authoritative)
      const content = [
        '{"type":"assistant","message":{"content":[{"type":"text","text":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}"}]}}',
        '{"type":"result","result":"THREAT_DETECTION_RESULT:{\\"prompt_injection\\":false,\\"secret_leak\\":false,\\"malicious_patch\\":false,\\"reasons\\":[]}"}',
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict).toBeDefined();
    });
  });

  describe("non-stream format (--print / --output-format text)", () => {
    it("should extract from a realistic non-stream detection log", () => {
      const content = [
        "● Read workflow prompt and agent output files (shell)",
        "  │ cat /tmp/gh-aw/threat-detection/aw-prompts/prompt.txt",
        "  └ 195 lines...",
        "",
        "Looking at the content carefully, I notice a classic prompt injection pattern.",
        "",
        'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":["Injected JSON payload in prompt.txt"]}',
        "",
        "Total usage est:        1 Premium request",
      ].join("\n");
      const { verdict, error } = parseDetectionLog(content);

      expect(error).toBeUndefined();
      expect(verdict.prompt_injection).toBe(true);
      expect(verdict.secret_leak).toBe(false);
      expect(verdict.malicious_patch).toBe(false);
      expect(verdict.reasons).toEqual(["Injected JSON payload in prompt.txt"]);
    });
  });
});

describe("main", () => {
  let mod;

  beforeEach(async () => {
    vi.clearAllMocks();
    mockExistsSync.mockReturnValue(false);
    mockReadFileSync.mockReturnValue("");
    // Reset RUN_DETECTION environment variable
    delete process.env.RUN_DETECTION;
    // Re-import to get fresh module with mocks
    mod = await import("./parse_threat_detection_results.cjs");
  });

  describe("when detection was not needed (RUN_DETECTION != 'true')", () => {
    it("should set conclusion=skipped when RUN_DETECTION is undefined", async () => {
      delete process.env.RUN_DETECTION;

      await mod.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("success", "true");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should set conclusion=skipped when RUN_DETECTION is 'false'", async () => {
      process.env.RUN_DETECTION = "false";

      await mod.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("success", "true");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should set conclusion=skipped when RUN_DETECTION is empty string", async () => {
      process.env.RUN_DETECTION = "";

      await mod.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "skipped");
      expect(mockCore.setOutput).toHaveBeenCalledWith("success", "true");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("when detection is needed (RUN_DETECTION === 'true')", () => {
    beforeEach(() => {
      process.env.RUN_DETECTION = "true";
    });

    it("should fail when detection log file does not exist", async () => {
      mockExistsSync.mockReturnValue(false);

      await mod.main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
      expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Detection log file not found"));
    });

    // Note: The following tests are skipped because mocking fs for CJS modules
    // is difficult in vitest (same issue as safe_output_validator.test.cjs).
    // The core parsing logic is thoroughly tested via parseDetectionLog above.
    // These tests document the expected behavior of main() for each scenario.
    describe.skip("with detection log file present (CJS fs mock limitation)", () => {
      it("should fail when detection log has no result line", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockReturnValue("just some debug output\nno result here\n");

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
        expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("No THREAT_DETECTION_RESULT found"));
      });

      it("should fail when detection log has multiple result lines", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockReturnValue(
          ['THREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}', 'THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":[]}'].join(
            "\n"
          )
        );

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
        expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Multiple conflicting THREAT_DETECTION_RESULT entries"));
      });

      it("should fail when result JSON is invalid", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockReturnValue("THREAT_DETECTION_RESULT:{bad json}");

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
        expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to parse JSON"));
      });

      it("should set conclusion=success with clean verdict", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockReturnValue('debug output\nTHREAT_DETECTION_RESULT:{"prompt_injection":false,"secret_leak":false,"malicious_patch":false,"reasons":[]}\n');

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "success");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "true");
        expect(mockCore.setFailed).not.toHaveBeenCalled();
      });

      it("should set conclusion=failure when threats are detected", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockReturnValue('THREAT_DETECTION_RESULT:{"prompt_injection":true,"secret_leak":false,"malicious_patch":false,"reasons":["found injection"]}');

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
        expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Security threats detected: prompt injection"));
      });

      it("should fail when readFileSync throws", async () => {
        mockExistsSync.mockReturnValue(true);
        mockReadFileSync.mockImplementation(() => {
          throw new Error("EACCES: permission denied");
        });

        await mod.main();

        expect(mockCore.setOutput).toHaveBeenCalledWith("conclusion", "failure");
        expect(mockCore.setOutput).toHaveBeenCalledWith("success", "false");
        expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to read detection log"));
      });
    });
  });
});
