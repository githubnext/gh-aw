import { describe, it, expect, vi, beforeEach } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

describe("upload_code_coverage (Handler Factory Architecture)", () => {
  let handler;

  beforeEach(async () => {
    vi.clearAllMocks();
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    const { main } = require("./upload_code_coverage.cjs");
    handler = await main({ max: 1 });
  });

  it("should return a function from main()", async () => {
    const { main } = require("./upload_code_coverage.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should process a valid upload_code_coverage message and set outputs", async () => {
    const message = {
      type: "upload_code_coverage",
      file: "cobertura.xml",
      language: "Go",
      label: "code-coverage/unit-tests",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.file).toBe("cobertura.xml");
    expect(result.language).toBe("Go");
    expect(result.label).toBe("code-coverage/unit-tests");

    expect(mockCore.setOutput).toHaveBeenCalledWith("upload_code_coverage_file", "cobertura.xml");
    expect(mockCore.setOutput).toHaveBeenCalledWith("upload_code_coverage_language", "Go");
    expect(mockCore.setOutput).toHaveBeenCalledWith("upload_code_coverage_label", "code-coverage/unit-tests");
  });

  it("should trim whitespace in file/language/label", async () => {
    const message = {
      type: "upload_code_coverage",
      file: "  cobertura.xml  ",
      language: "  Go  ",
      label: "  code-coverage/unit-tests  ",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.file).toBe("cobertura.xml");
    expect(result.language).toBe("Go");
    expect(result.label).toBe("code-coverage/unit-tests");
  });

  it("should fail when file is missing", async () => {
    const message = {
      type: "upload_code_coverage",
      language: "Go",
      label: "code-coverage/unit-tests",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toMatch(/file/);
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should fail when language is missing", async () => {
    const message = {
      type: "upload_code_coverage",
      file: "cobertura.xml",
      label: "code-coverage/unit-tests",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toMatch(/language/);
  });

  it("should fail when label is missing", async () => {
    const message = {
      type: "upload_code_coverage",
      file: "cobertura.xml",
      language: "Go",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toMatch(/label/);
  });

  it("should fail when max count is reached", async () => {
    const { main } = require("./upload_code_coverage.cjs");
    const limitedHandler = await main({ max: 1 });

    const message = {
      type: "upload_code_coverage",
      file: "cobertura.xml",
      language: "Go",
      label: "code-coverage/unit-tests",
    };

    const first = await limitedHandler(message, {});
    expect(first.success).toBe(true);

    const second = await limitedHandler(message, {});
    expect(second.success).toBe(false);
    expect(second.error).toMatch(/Max count/);
  });

  it("should support staged mode preview without setting outputs", async () => {
    const { main } = require("./upload_code_coverage.cjs");
    const stagedHandler = await main({ max: 1, staged: true });

    const message = {
      type: "upload_code_coverage",
      file: "cobertura.xml",
      language: "Go",
      label: "code-coverage/unit-tests",
    };

    const result = await stagedHandler(message, {});

    expect(result.success).toBe(true);
    expect(result.staged).toBe(true);
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });
});
