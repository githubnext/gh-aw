import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), addHeading: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
};

global.core = mockCore;

describe("missing_tool.cjs (Handler Factory Architecture)", () => {
  let handler;

  beforeEach(async () => {
    vi.clearAllMocks();

    // Load the module and create handler
    const { main } = require("./missing_tool.cjs");
    handler = await main({});
  });

  it("should return a function from main()", async () => {
    const { main } = require("./missing_tool.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should process missing-tool entry", async () => {
    const message = { type: "missing_tool", tool: "docker", reason: "Need containerization support", alternatives: "Use VM or manual setup" };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.tool.tool).toBe("docker");
    expect(result.tool.reason).toBe("Need containerization support");
    expect(result.tool.alternatives).toBe("Use VM or manual setup");
    expect(result.tool.timestamp).toBeDefined();
    expect(mockCore.info).toHaveBeenCalledWith("Recorded missing tool: docker");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should process multiple missing-tool entries", async () => {
    const messages = [
      { type: "missing_tool", tool: "docker", reason: "Need containerization" },
      { type: "missing_tool", tool: "kubectl", reason: "Need k8s support" },
    ];

    for (const msg of messages) {
      const result = await handler(msg, {});
      expect(result.success).toBe(true);
    }

    expect(mockCore.info).toHaveBeenCalledWith("Recorded missing tool: docker");
    expect(mockCore.info).toHaveBeenCalledWith("Recorded missing tool: kubectl");
  });

  it("should skip entries missing tool field", async () => {
    const message = { type: "missing_tool", reason: "No tool specified" };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toBe("Missing 'tool' field");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing 'tool' field"));
  });

  it("should skip entries missing reason field", async () => {
    const message = { type: "missing_tool", tool: "some-tool" };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toBe("Missing 'reason' field");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing 'reason' field"));
  });

  it("should respect max reports limit", async () => {
    const { main } = require("./missing_tool.cjs");
    const limitedHandler = await main({ max: 2 });

    const messages = [
      { type: "missing_tool", tool: "tool1", reason: "reason1" },
      { type: "missing_tool", tool: "tool2", reason: "reason2" },
      { type: "missing_tool", tool: "tool3", reason: "reason3" },
    ];

    const result1 = await limitedHandler(messages[0], {});
    expect(result1.success).toBe(true);

    const result2 = await limitedHandler(messages[1], {});
    expect(result2.success).toBe(true);

    const result3 = await limitedHandler(messages[2], {});
    expect(result3.success).toBe(false);
    expect(result3.error).toContain("Max count");
    expect(mockCore.info).toHaveBeenCalledWith("Reached maximum number of missing tool reports (2)");
  });

  it("should work without max limit", async () => {
    const messages = [
      { type: "missing_tool", tool: "tool1", reason: "reason1" },
      { type: "missing_tool", tool: "tool2", reason: "reason2" },
      { type: "missing_tool", tool: "tool3", reason: "reason3" },
    ];

    for (const msg of messages) {
      const result = await handler(msg, {});
      expect(result.success).toBe(true);
    }
  });

  it("should add timestamp to reported tools", async () => {
    const message = { type: "missing_tool", tool: "test-tool", reason: "testing timestamp" };

    const beforeTime = new Date();
    const result = await handler(message, {});
    const afterTime = new Date();

    expect(result.tool.timestamp).toBeDefined();
    const timestamp = new Date(result.tool.timestamp);
    expect(timestamp).toBeInstanceOf(Date);
    expect(timestamp.getTime()).toBeGreaterThanOrEqual(beforeTime.getTime());
    expect(timestamp.getTime()).toBeLessThanOrEqual(afterTime.getTime());
  });

  it("should handle alternatives field", async () => {
    const messageWithAlternatives = { type: "missing_tool", tool: "docker", reason: "Need containerization", alternatives: "Use VM" };
    const result1 = await handler(messageWithAlternatives, {});

    expect(result1.success).toBe(true);
    expect(result1.tool.alternatives).toBe("Use VM");

    // Create a new handler for second test
    const { main } = require("./missing_tool.cjs");
    const handler2 = await main({});

    const messageWithoutAlternatives = { type: "missing_tool", tool: "kubectl", reason: "Need k8s" };
    const result2 = await handler2(messageWithoutAlternatives, {});

    expect(result2.success).toBe(true);
    expect(result2.tool.alternatives).toBe(null);
  });

  it("should set outputs for first missing tool", async () => {
    const message = { type: "missing_tool", tool: "test-tool", reason: "test reason" };

    await handler(message, {});

    expect(mockCore.setOutput).toHaveBeenCalledWith("tools_reported", expect.any(String));
    expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", expect.any(String));
  });
});
