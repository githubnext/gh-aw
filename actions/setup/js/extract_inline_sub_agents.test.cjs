// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach } from "vitest";
const fs = require("fs");
const path = require("path");
const os = require("os");

// Provide a minimal core mock so the module loads correctly.
global.core = {
  info: () => {},
  warning: () => {},
  error: () => {},
  setFailed: () => {},
};

const { extractInlineSubAgents, writeInlineSubAgents } = require("./extract_inline_sub_agents.cjs");

// Helper: returns a ## agent: `name` start marker line.
const agentMarker = name => `## agent: \`${name}\``;

// ─────────────────────────────────────────────────────────────────────────────
// extractInlineSubAgents — unit tests
// ─────────────────────────────────────────────────────────────────────────────

describe("extractInlineSubAgents", () => {
  it("returns original content unchanged when no markers present", () => {
    const content = "# Hello\n\nThis is a workflow.";
    const { mainContent, agents } = extractInlineSubAgents(content);
    expect(mainContent).toBe(content);
    expect(agents).toHaveLength(0);
  });

  it("returns empty main content and no agents for empty string", () => {
    const { mainContent, agents } = extractInlineSubAgents("");
    expect(mainContent).toBe("");
    expect(agents).toHaveLength(0);
  });

  it("extracts a single agent block", () => {
    const content = ["# Main workflow", "", "Handle the issue.", "", agentMarker("planner"), "---", "engine: copilot", "---", "You are a planning assistant."].join("\n");

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("# Main workflow\n\nHandle the issue.");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toContain("You are a planning assistant.");
    expect(agents[0].content).toContain("engine: copilot");
  });

  it("extracts multiple agent blocks", () => {
    const content = ["Main prompt.", "", agentMarker("planner"), "Planner prompt.", "", agentMarker("executor"), "Executor prompt."].join("\n");

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("Main prompt.");
    expect(agents).toHaveLength(2);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toBe("Planner prompt.");
    expect(agents[1].name).toBe("executor");
    expect(agents[1].content).toBe("Executor prompt.");
  });

  it("agent block ends at next H2 heading", () => {
    const content = ["Main prompt.", "", agentMarker("planner"), "Planner content.", "", "## Summary", "This content is outside the agent block."].join("\n");

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("Main prompt.");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toBe("Planner content.");
    expect(agents[0].content).not.toContain("Summary");
    expect(agents[0].content).not.toContain("outside the agent block");
  });

  it("next agent marker (H2) ends the previous agent block", () => {
    const content = ["Main.", "", agentMarker("planner"), "Planner.", "", agentMarker("executor"), "Executor."].join("\n");

    const { agents } = extractInlineSubAgents(content);

    expect(agents).toHaveLength(2);
    expect(agents[0].content).toBe("Planner.");
    expect(agents[1].content).toBe("Executor.");
  });

  it("agent at start of file produces empty main content", () => {
    const content = agentMarker("only") + "\nAgent content.";
    const { mainContent, agents } = extractInlineSubAgents(content);
    expect(mainContent).toBe("");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("only");
  });

  it("agent content is trimmed", () => {
    const content = "Main.\n\n" + agentMarker("a") + "\n\n\n  Trimmed.  \n\n";
    const { agents } = extractInlineSubAgents(content);
    expect(agents[0].content).toBe("Trimmed.");
  });

  it("trailing newlines are stripped from main content", () => {
    const content = "Line 1.\nLine 2.\n\n\n" + agentMarker("a") + "\nContent.";
    const { mainContent } = extractInlineSubAgents(content);
    expect(mainContent).toBe("Line 1.\nLine 2.");
  });

  it("accepts valid lowercase name variants", () => {
    const cases = [{ name: "my-agent" }, { name: "my_agent" }, { name: "agent1" }, { name: "a" }, { name: "planner-v2" }];
    for (const { name } of cases) {
      const { agents } = extractInlineSubAgents("Main.\n\n" + agentMarker(name) + "\nContent.");
      expect(agents).toHaveLength(1);
      expect(agents[0].name).toBe(name);
    }
  });

  it("does not recognize invalid separator forms", () => {
    const invalids = ["## agent: `1agent`", "## agent: `my agent`", "## agent: `my/agent`", "## agent:", "## agent: myagent", "## agent: `MyAgent`", "# agent: `myagent`", "### agent: `myagent`"];
    for (const sep of invalids) {
      const content = `Main.\n\n${sep}\nContent.`;
      const { mainContent, agents } = extractInlineSubAgents(content);
      expect(mainContent).toBe(content);
      expect(agents).toHaveLength(0);
    }
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// writeInlineSubAgents — integration tests (real filesystem)
// ─────────────────────────────────────────────────────────────────────────────

describe("writeInlineSubAgents", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "inline-agents-test-"));
  });

  afterEach(() => {
    if (fs.existsSync(tmpDir)) {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("returns original content unchanged when no markers present", () => {
    const content = "# Workflow\n\nNo agents here.";
    const result = writeInlineSubAgents(content, tmpDir);
    expect(result).toBe(content);
    const agentsDir = path.join(tmpDir, ".github", "agents");
    expect(fs.existsSync(agentsDir)).toBe(false);
  });

  it("writes a single agent file and returns main content", () => {
    const content = ["# Workflow", "", "Main prompt.", "", agentMarker("helper"), "---", "engine: copilot", "---", "You are a helper."].join("\n");

    const result = writeInlineSubAgents(content, tmpDir);

    expect(result).toBe("# Workflow\n\nMain prompt.");

    const agentPath = path.join(tmpDir, ".github", "agents", "helper.md");
    expect(fs.existsSync(agentPath)).toBe(true);
    const written = fs.readFileSync(agentPath, "utf8");
    expect(written).toContain("You are a helper.");
    expect(written).toContain("engine: copilot");
  });

  it("writes multiple agent files", () => {
    const content = ["Main.", "", agentMarker("planner"), "Planner.", "", agentMarker("executor"), "Executor."].join("\n");

    writeInlineSubAgents(content, tmpDir);

    expect(fs.existsSync(path.join(tmpDir, ".github", "agents", "planner.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, ".github", "agents", "executor.md"))).toBe(true);
  });

  it("agent file content ends with a newline", () => {
    const content = "Main.\n\n" + agentMarker("a") + "\nContent without trailing newline";
    writeInlineSubAgents(content, tmpDir);
    const written = fs.readFileSync(path.join(tmpDir, ".github", "agents", "a.md"), "utf8");
    expect(written.endsWith("\n")).toBe(true);
  });

  it("creates .github/agents directory if it does not exist", () => {
    const content = "Main.\n\n" + agentMarker("new") + "\nContent.";
    const agentsDir = path.join(tmpDir, ".github", "agents");
    expect(fs.existsSync(agentsDir)).toBe(false);
    writeInlineSubAgents(content, tmpDir);
    expect(fs.existsSync(agentsDir)).toBe(true);
  });

  it("agent block ends at H2 — content after is not written to agent file", () => {
    const content = ["Main.", "", agentMarker("a"), "Agent body.", "", "## Notes", "Footer content that should not appear in the agent file."].join("\n");

    const result = writeInlineSubAgents(content, tmpDir);

    expect(result).toBe("Main.");
    const written = fs.readFileSync(path.join(tmpDir, ".github", "agents", "a.md"), "utf8");
    expect(written).toContain("Agent body.");
    expect(written).not.toContain("Footer content");
  });
});
