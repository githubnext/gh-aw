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
    const content = `# Main workflow

Handle the issue.

## agent: planner
---
engine: copilot
---
You are a planning assistant.`;

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("# Main workflow\n\nHandle the issue.");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toContain("You are a planning assistant.");
    expect(agents[0].content).toContain("engine: copilot");
  });

  it("extracts multiple agent blocks", () => {
    const content = `Main prompt.

## agent: planner
Planner prompt.

## agent: executor
Executor prompt.`;

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("Main prompt.");
    expect(agents).toHaveLength(2);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toBe("Planner prompt.");
    expect(agents[1].name).toBe("executor");
    expect(agents[1].content).toBe("Executor prompt.");
  });

  it("respects ## end: name marker", () => {
    const content = `Main prompt.

## agent: planner
Planner content.
## end: planner

This content is outside any agent block.`;

    const { mainContent, agents } = extractInlineSubAgents(content);

    expect(mainContent).toBe("Main prompt.");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("planner");
    expect(agents[0].content).toBe("Planner content.");
    expect(agents[0].content).not.toContain("outside any agent block");
  });

  it("end marker stops agent block; content between blocks is excluded", () => {
    const content = `Main.

## agent: planner
Planner.
## end: planner

Between-agents content.

## agent: executor
Executor.
## end: executor`;

    const { agents } = extractInlineSubAgents(content);

    expect(agents).toHaveLength(2);
    expect(agents[0].content).toBe("Planner.");
    expect(agents[1].content).toBe("Executor.");
  });

  it("mismatched end marker is treated as plain text", () => {
    const content = `Main.

## agent: planner
Planner content.
## end: executor
More planner content.`;

    const { agents } = extractInlineSubAgents(content);

    expect(agents).toHaveLength(1);
    expect(agents[0].content).toContain("Planner content.");
    expect(agents[0].content).toContain("More planner content.");
  });

  it("orphan end marker (no open agent) is treated as plain text", () => {
    const content = "Main.\n## end: nobody\nMore main.";
    const { mainContent, agents } = extractInlineSubAgents(content);
    expect(mainContent).toBe(content);
    expect(agents).toHaveLength(0);
  });

  it("end marker with trailing whitespace is recognised", () => {
    const content = "Main.\n\n## agent: a\nContent.\n## end: a   \nTrailing.";
    const { agents } = extractInlineSubAgents(content);
    expect(agents).toHaveLength(1);
    expect(agents[0].content).toBe("Content.");
  });

  it("agent at start of file produces empty main content", () => {
    const content = `## agent: only
Agent content.`;
    const { mainContent, agents } = extractInlineSubAgents(content);
    expect(mainContent).toBe("");
    expect(agents).toHaveLength(1);
    expect(agents[0].name).toBe("only");
  });

  it("agent content is trimmed", () => {
    const content = "Main.\n\n## agent: a\n\n\n  Trimmed.  \n\n";
    const { agents } = extractInlineSubAgents(content);
    expect(agents[0].content).toBe("Trimmed.");
  });

  it("trailing newlines are stripped from main content", () => {
    const content = "Line 1.\nLine 2.\n\n\n## agent: a\nContent.";
    const { mainContent } = extractInlineSubAgents(content);
    expect(mainContent).toBe("Line 1.\nLine 2.");
  });

  it("accepts valid name variants", () => {
    const cases = [
      { sep: "## agent: my-agent", name: "my-agent" },
      { sep: "## agent: my_agent", name: "my_agent" },
      { sep: "## agent: agent1", name: "agent1" },
      { sep: "## agent: MyAgent", name: "MyAgent" },
      { sep: "## agent: a", name: "a" },
    ];
    for (const { sep, name } of cases) {
      const { agents } = extractInlineSubAgents(`Main.\n\n${sep}\nContent.`);
      expect(agents).toHaveLength(1);
      expect(agents[0].name).toBe(name);
    }
  });

  it("does not recognize invalid separator forms", () => {
    const invalids = ["## agent: 1agent", "## agent: my agent", "## agent: my/agent", "## agent:", "# agent: myagent", "### agent: myagent"];
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
    const content = `# Workflow

Main prompt.

## agent: helper
---
engine: copilot
---
You are a helper.`;

    const result = writeInlineSubAgents(content, tmpDir);

    expect(result).toBe("# Workflow\n\nMain prompt.");

    const agentPath = path.join(tmpDir, ".github", "agents", "helper.md");
    expect(fs.existsSync(agentPath)).toBe(true);
    const written = fs.readFileSync(agentPath, "utf8");
    expect(written).toContain("You are a helper.");
    expect(written).toContain("engine: copilot");
  });

  it("writes multiple agent files", () => {
    const content = `Main.

## agent: planner
Planner.

## agent: executor
Executor.`;

    writeInlineSubAgents(content, tmpDir);

    expect(fs.existsSync(path.join(tmpDir, ".github", "agents", "planner.md"))).toBe(true);
    expect(fs.existsSync(path.join(tmpDir, ".github", "agents", "executor.md"))).toBe(true);
  });

  it("agent file content ends with a newline", () => {
    const content = "Main.\n\n## agent: a\nContent without trailing newline";
    writeInlineSubAgents(content, tmpDir);
    const written = fs.readFileSync(path.join(tmpDir, ".github", "agents", "a.md"), "utf8");
    expect(written.endsWith("\n")).toBe(true);
  });

  it("creates .github/agents directory if it does not exist", () => {
    const content = "Main.\n\n## agent: new\nContent.";
    const agentsDir = path.join(tmpDir, ".github", "agents");
    expect(fs.existsSync(agentsDir)).toBe(false);
    writeInlineSubAgents(content, tmpDir);
    expect(fs.existsSync(agentsDir)).toBe(true);
  });

  it("strips agent blocks but content after end marker stays out of agent file", () => {
    const content = `Main.

## agent: a
Agent body.
## end: a

Footer content that should not appear in the agent file.`;

    const result = writeInlineSubAgents(content, tmpDir);

    expect(result).toBe("Main.");
    const written = fs.readFileSync(path.join(tmpDir, ".github", "agents", "a.md"), "utf8");
    expect(written).toContain("Agent body.");
    expect(written).not.toContain("Footer content");
  });
});
