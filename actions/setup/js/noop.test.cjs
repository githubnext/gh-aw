import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
};

global.core = mockCore;

describe("noop.cjs (Handler Factory Architecture)", () => {
  let handler;

  beforeEach(async () => {
    vi.clearAllMocks();
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Load the module and create handler
    const { main } = require("./noop.cjs");
    handler = await main({});
  });

  it("should return a function from main()", async () => {
    const { main } = require("./noop.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should process single noop message", async () => {
    const message = { type: "noop", message: "No issues found in this review" };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.message).toBe("No issues found in this review");
    expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: No issues found in this review");
    expect(mockCore.setOutput).toHaveBeenCalledWith("noop_message", "No issues found in this review");
    expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_NOOP_MESSAGE", "No issues found in this review");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should process multiple noop messages", async () => {
    const messages = [
      { type: "noop", message: "First message" },
      { type: "noop", message: "Second message" },
      { type: "noop", message: "Third message" },
    ];

    for (const msg of messages) {
      const result = await handler(msg, {});
      expect(result.success).toBe(true);
    }

    expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: First message");
    expect(mockCore.info).toHaveBeenCalledWith("No-op message 2: Second message");
    expect(mockCore.info).toHaveBeenCalledWith("No-op message 3: Third message");
    // Only first message is exported
    expect(mockCore.setOutput).toHaveBeenCalledWith("noop_message", "First message");
    expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_NOOP_MESSAGE", "First message");
  });

  it("should show preview in staged mode", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    const { main } = require("./noop.cjs");
    const stagedHandler = await main({});

    const message = { type: "noop", message: "Test message in staged mode" };
    const result = await stagedHandler(message, {});

    expect(result.success).toBe(true);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("📝 No-op message 1 preview written to step summary"));
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode"));
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Test message in staged mode"));
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should handle message with no content", async () => {
    const message = { type: "noop" };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: (no message)");
  });

  it("should generate proper step summary format", async () => {
    const message = { type: "noop", message: "Analysis complete" };

    await handler(message, {});

    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("## No-Op Message 1");
    expect(summaryCall).toContain("Analysis complete");
  });
});
