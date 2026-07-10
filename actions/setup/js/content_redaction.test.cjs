// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
const { runContentRedaction } = require("./content_redaction.cjs");
const fs = require("fs");
const path = require("path");
const os = require("os");

describe("content_redaction", () => {
  let mockCore;
  let tmpDir;
  let outputFile;
  let policyFile;

  beforeEach(() => {
    // Create temp directory for test files
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "content-redaction-test-"));
    outputFile = path.join(tmpDir, "agent_output.json");
    policyFile = path.join(tmpDir, "policy.md");

    // Reset mocks before each test
    mockCore = {
      info: () => {},
      warning: () => {},
      error: () => {},
      debug: () => {},
      setOutput: () => {},
      messages: [],
      infos: [],
      warnings: [],
      errors: [],
      outputs: {},
    };

    // Capture all logged messages
    mockCore.info = msg => {
      mockCore.infos.push(msg);
      mockCore.messages.push({ level: "info", message: msg });
    };
    mockCore.warning = msg => {
      mockCore.warnings.push(msg);
      mockCore.messages.push({ level: "warning", message: msg });
    };
    mockCore.error = msg => {
      mockCore.errors.push(msg);
      mockCore.messages.push({ level: "error", message: msg });
    };
    mockCore.setOutput = (key, value) => {
      mockCore.outputs[key] = value;
    };
  });

  afterEach(() => {
    // Clean up temp directory
    if (tmpDir && fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("skips redaction when output file does not exist", () => {
    runContentRedaction({
      core: mockCore,
      outputFile: "/nonexistent/file.json",
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.infos).toContain("No agent output file found; skipping content redaction");
    expect(mockCore.outputs.skipped).toBe("true");
  });

  it("skips redaction when policy file does not exist", () => {
    // Create output file but no policy file
    fs.writeFileSync(outputFile, JSON.stringify({ items: [] }));

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile: "/nonexistent/policy.md",
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.warnings).toContain("Content redaction policy is empty; skipping redaction");
    expect(mockCore.outputs.skipped).toBe("true");
  });

  it("skips redaction when policy file is empty", () => {
    fs.writeFileSync(outputFile, JSON.stringify({ items: [] }));
    fs.writeFileSync(policyFile, "   \n  \n  "); // Whitespace only

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.warnings).toContain("Content redaction policy is empty; skipping redaction");
    expect(mockCore.outputs.skipped).toBe("true");
  });

  it("skips redaction when agent output is invalid JSON", () => {
    fs.writeFileSync(outputFile, "not valid json");
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.warnings.some(w => w.includes("Failed to parse agent output JSON"))).toBe(true);
    expect(mockCore.outputs.skipped).toBe("true");
  });

  it("skips redaction when agent output has no items array", () => {
    fs.writeFileSync(outputFile, JSON.stringify({ foo: "bar" }));
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.warnings).toContain("Agent output has no items array; skipping redaction");
    expect(mockCore.outputs.skipped).toBe("true");
  });

  it("processes items and passes all non-text-bearing types", () => {
    const agentOutput = {
      items: [
        { type: "add-comment", body: "Test comment" },
        { type: "noop", message: "Just logging" },
        { type: "missing-tool", tool: "something" },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.outputs.passed_count).toBe("3");
    expect(mockCore.outputs.blocked_count).toBe("0");
    expect(mockCore.outputs.rewritten_count).toBe("0");
    expect(mockCore.outputs.has_blocked_items).toBe("false");

    // Verify output file was updated
    const result = JSON.parse(fs.readFileSync(outputFile, "utf8"));
    expect(result.items).toHaveLength(3);
  });

  it("respects scope filter and only processes specified types", () => {
    const agentOutput = {
      items: [
        { type: "add-comment", body: "Test comment" },
        { type: "create-issue", title: "Issue title" },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: ["add-comment"], // Only process add-comment
    });

    // Both items should pass (1 in scope, 1 out of scope)
    expect(mockCore.outputs.passed_count).toBe("2");
    expect(mockCore.infos.some(i => i.includes("reviewing add-comment item"))).toBe(true);
    expect(mockCore.infos.some(i => i.includes("reviewing create-issue item"))).toBe(false);
  });

  it("normalizes type identifiers with dashes", () => {
    const agentOutput = {
      items: [
        { type: "add-comment", body: "Test comment" },
        { type: "create-pull-request", title: "PR title" },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "block",
      scope: [],
    });

    expect(mockCore.outputs.passed_count).toBe("2");
    expect(mockCore.infos.some(i => i.includes("reviewing add-comment item"))).toBe(true);
    expect(mockCore.infos.some(i => i.includes("reviewing create-pull-request item"))).toBe(true);
  });

  it("sets has_blocked_items to false when on-failure is warn", () => {
    const agentOutput = {
      items: [{ type: "add-comment", body: "Test comment" }],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    fs.writeFileSync(policyFile, "Do not disclose secrets");

    runContentRedaction({
      core: mockCore,
      outputFile,
      policyFile,
      onFailure: "warn", // Warn mode instead of block
      scope: [],
    });

    // Even if blocked > 0 (which it's not in this test), has_blocked_items should be false
    expect(mockCore.outputs.has_blocked_items).toBe("false");
  });
});
