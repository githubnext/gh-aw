import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

describe("gateway_difc_filtered.cjs", () => {
  let parseDifcFilteredEvents;
  let getDifcFilteredEvents;
  let generateDifcFilteredSection;
  let testDir;
  let originalEnv;

  beforeEach(async () => {
    // Create a temporary directory for test files
    testDir = path.join(os.tmpdir(), `gh-aw-test-difc-${Date.now()}`);
    fs.mkdirSync(testDir, { recursive: true });

    // Set up prompts directory with the remediation md file
    const promptsDir = path.join(testDir, "prompts");
    fs.mkdirSync(promptsDir, { recursive: true });
    const remediationSrc = path.resolve(path.dirname(new URL(import.meta.url).pathname), "../md/integrity_filter_remediation.md");
    fs.copyFileSync(remediationSrc, path.join(promptsDir, "integrity_filter_remediation.md"));
    originalEnv = process.env.GH_AW_PROMPTS_DIR;
    process.env.GH_AW_PROMPTS_DIR = promptsDir;

    // Dynamic import to get fresh module state
    const module = await import("./gateway_difc_filtered.cjs");
    parseDifcFilteredEvents = module.parseDifcFilteredEvents;
    getDifcFilteredEvents = module.getDifcFilteredEvents;
    generateDifcFilteredSection = module.generateDifcFilteredSection;
  });

  afterEach(() => {
    // Restore environment
    if (originalEnv !== undefined) {
      process.env.GH_AW_PROMPTS_DIR = originalEnv;
    } else {
      delete process.env.GH_AW_PROMPTS_DIR;
    }
    // Clean up test directory
    if (testDir && fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  describe("parseDifcFilteredEvents", () => {
    it("should return empty array for empty content", () => {
      expect(parseDifcFilteredEvents("")).toEqual([]);
      expect(parseDifcFilteredEvents("\n\n")).toEqual([]);
    });

    it("should extract DIFC_FILTERED events from JSONL content", () => {
      const content = [
        JSON.stringify({
          timestamp: "2026-03-18T17:30:00Z",
          type: "DIFC_FILTERED",
          server_id: "github",
          tool_name: "list_issues",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo/issues/42",
          number: "42",
        }),
        JSON.stringify({ timestamp: "2026-03-18T17:30:01Z", type: "RESPONSE", server_id: "github" }),
      ].join("\n");

      const events = parseDifcFilteredEvents(content);
      expect(events).toHaveLength(1);
      expect(events[0].tool_name).toBe("list_issues");
      expect(events[0].number).toBe("42");
    });

    it("should extract multiple DIFC_FILTERED events", () => {
      const content = [JSON.stringify({ type: "DIFC_FILTERED", tool_name: "tool1", reason: "r1" }), JSON.stringify({ type: "DIFC_FILTERED", tool_name: "tool2", reason: "r2" }), JSON.stringify({ type: "REQUEST", tool_name: "tool3" })].join(
        "\n"
      );

      const events = parseDifcFilteredEvents(content);
      expect(events).toHaveLength(2);
      expect(events[0].tool_name).toBe("tool1");
      expect(events[1].tool_name).toBe("tool2");
    });

    it("should skip malformed JSON lines", () => {
      const content = ["{ not valid json", JSON.stringify({ type: "DIFC_FILTERED", tool_name: "valid_tool" }), "another bad line"].join("\n");

      const events = parseDifcFilteredEvents(content);
      expect(events).toHaveLength(1);
      expect(events[0].tool_name).toBe("valid_tool");
    });

    it("should skip blank lines", () => {
      const content = "\n" + JSON.stringify({ type: "DIFC_FILTERED", tool_name: "t1" }) + "\n\n" + JSON.stringify({ type: "DIFC_FILTERED", tool_name: "t2" }) + "\n";

      const events = parseDifcFilteredEvents(content);
      expect(events).toHaveLength(2);
    });

    it("should ignore lines without DIFC_FILTERED string for efficiency", () => {
      const content = [JSON.stringify({ type: "REQUEST", tool_name: "not_filtered" }), JSON.stringify({ type: "RESPONSE", result: "ok" })].join("\n");

      const events = parseDifcFilteredEvents(content);
      expect(events).toHaveLength(0);
    });
  });

  describe("getDifcFilteredEvents", () => {
    it("should return empty array when neither log file exists", () => {
      const nonExistent1 = path.join(testDir, "nonexistent1.jsonl");
      const nonExistent2 = path.join(testDir, "nonexistent2.jsonl");

      const events = getDifcFilteredEvents(nonExistent1, nonExistent2);
      expect(events).toEqual([]);
    });

    it("should read events from primary gateway.jsonl path", () => {
      const jsonlPath = path.join(testDir, "gateway.jsonl");
      const content = JSON.stringify({ type: "DIFC_FILTERED", tool_name: "list_issues", reason: "test" });
      fs.writeFileSync(jsonlPath, content);

      const events = getDifcFilteredEvents(jsonlPath, path.join(testDir, "rpc.jsonl"));
      expect(events).toHaveLength(1);
      expect(events[0].tool_name).toBe("list_issues");
    });

    it("should fall back to rpc-messages.jsonl when gateway.jsonl does not exist", () => {
      const rpcPath = path.join(testDir, "rpc-messages.jsonl");
      const content = JSON.stringify({ type: "DIFC_FILTERED", tool_name: "get_issue", reason: "secrecy" });
      fs.writeFileSync(rpcPath, content);

      const events = getDifcFilteredEvents(path.join(testDir, "nonexistent.jsonl"), rpcPath);
      expect(events).toHaveLength(1);
      expect(events[0].tool_name).toBe("get_issue");
    });

    it("should prefer primary path over fallback when both exist", () => {
      const jsonlPath = path.join(testDir, "gateway.jsonl");
      const rpcPath = path.join(testDir, "rpc-messages.jsonl");
      fs.writeFileSync(jsonlPath, JSON.stringify({ type: "DIFC_FILTERED", tool_name: "primary_tool" }));
      fs.writeFileSync(rpcPath, JSON.stringify({ type: "DIFC_FILTERED", tool_name: "fallback_tool" }));

      const events = getDifcFilteredEvents(jsonlPath, rpcPath);
      expect(events).toHaveLength(1);
      expect(events[0].tool_name).toBe("primary_tool");
    });

    it("should return empty array when log file is empty", () => {
      const jsonlPath = path.join(testDir, "gateway.jsonl");
      fs.writeFileSync(jsonlPath, "");

      const events = getDifcFilteredEvents(jsonlPath, path.join(testDir, "rpc.jsonl"));
      expect(events).toEqual([]);
    });
  });

  describe("generateDifcFilteredSection", () => {
    it("should return empty string when no filtered events", () => {
      expect(generateDifcFilteredSection([])).toBe("");
      expect(generateDifcFilteredSection(null)).toBe("");
      expect(generateDifcFilteredSection(undefined)).toBe("");
    });

    it("should generate tip alert section for single filtered item", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo/issues/42",
          number: "42",
        },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("> [!NOTE]");
      expect(result).toContain("> <details>");
      expect(result).toContain("> </details>");
      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 1 item</b></summary>");
      expect(result).toContain("[#42](https://github.com/org/repo/issues/42)");
      expect(result).toContain("`list_issues`");
      expect(result).toContain("Integrity check failed");
    });

    it("should generate tip alert section for multiple filtered items", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo/issues/42",
          number: "42",
        },
        {
          type: "DIFC_FILTERED",
          tool_name: "get_issue",
          reason: "Secrecy check failed",
          html_url: "https://github.com/org/repo/issues/99",
          number: "99",
        },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("> [!NOTE]");
      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 2 items</b></summary>");
      expect(result).toContain("[#42](https://github.com/org/repo/issues/42)");
      expect(result).toContain("[#99](https://github.com/org/repo/issues/99)");
    });

    it("should use description as reference when html_url is absent", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          description: "resource:list_issues",
          reason: "Integrity check failed",
        },
      ];

      const result = generateDifcFilteredSection(events);

      // type prefix "resource:" should be stripped from the description label
      expect(result).toContain("list_issues");
      expect(result).not.toContain("resource:list_issues");
      expect(result).not.toContain("[#");
    });

    it("should use tool_name as reference when html_url and description are absent", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "my_tool",
          reason: "check failed",
        },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("`my_tool`");
    });

    it("should use html_url directly as label when number is absent", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_repos",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo",
        },
      ];

      const result = generateDifcFilteredSection(events);

      // html_url used as label when no number
      expect(result).toContain("[https://github.com/org/repo](https://github.com/org/repo)");
    });

    it("should include explanation text about why filtering happened", () => {
      const events = [{ type: "DIFC_FILTERED", tool_name: "tool", reason: "reason" }];
      const result = generateDifcFilteredSection(events);

      expect(result).toContain("blocked because they don't meet");
      expect(result).toContain("GitHub integrity level");
    });

    it("should include frontmatter yaml sample for adjusting min-integrity", () => {
      const events = [{ type: "DIFC_FILTERED", tool_name: "tool", reason: "reason" }];
      const result = generateDifcFilteredSection(events);

      expect(result).toContain("```yaml");
      expect(result).toContain("min-integrity:");
    });

    it("should render item lines without parentheses around tool and reason", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo/issues/42",
          number: "42",
        },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("`list_issues`: Integrity check failed");
      expect(result).not.toMatch(/\(`list_issues`/);
    });

    it("should start with double newline and note alert", () => {
      const events = [{ type: "DIFC_FILTERED", tool_name: "tool", reason: "reason" }];
      const result = generateDifcFilteredSection(events);

      expect(result).toMatch(/^\n\n> \[!NOTE\]/);
    });

    it("should use correct singular/plural form", () => {
      const singleEvent = [{ type: "DIFC_FILTERED", tool_name: "tool", reason: "reason" }];
      const singleResult = generateDifcFilteredSection(singleEvent);
      expect(singleResult).toContain("1 item");
      expect(singleResult).not.toContain("items");

      const multiEvents = [
        { type: "DIFC_FILTERED", tool_name: "tool1", reason: "r1" },
        { type: "DIFC_FILTERED", tool_name: "tool2", reason: "r2" },
      ];
      const multiResult = generateDifcFilteredSection(multiEvents);
      expect(multiResult).toContain("2 items");
    });

    it("should deduplicate filtered events with identical fields", () => {
      const events = [
        { type: "DIFC_FILTERED", tool_name: "list_issues", reason: "Integrity check failed", html_url: "https://github.com/org/repo/issues/42", number: "42" },
        { type: "DIFC_FILTERED", tool_name: "list_issues", reason: "Integrity check failed", html_url: "https://github.com/org/repo/issues/42", number: "42" },
        { type: "DIFC_FILTERED", tool_name: "get_issue", reason: "Secrecy check failed", html_url: "https://github.com/org/repo/issues/99", number: "99" },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 2 items</b></summary>");
      expect(result).toContain("[#42](https://github.com/org/repo/issues/42)");
      expect(result).toContain("[#99](https://github.com/org/repo/issues/99)");
    });

    it("should replace newlines in reason with spaces", () => {
      const events = [{ type: "DIFC_FILTERED", tool_name: "tool", reason: "line1\nline2" }];
      const result = generateDifcFilteredSection(events);

      expect(result).toContain("line1 line2");
      expect(result).not.toContain("line1\nline2");
    });

    it("should strip 'Resource X' prefix from reason to avoid duplication", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          description: "issue:github/gh-aw#21982",
          reason: "Resource 'issue:github/gh-aw#21982' has lower integrity than agent requires. The agent cannot read data with integrity below \"approved\".",
        },
      ];

      const result = generateDifcFilteredSection(events);

      // Resource prefix should be stripped from the reason
      expect(result).not.toContain("Resource 'issue:github/gh-aw#21982'");
      expect(result).toContain("has lower integrity than agent requires");
      // The type prefix "issue:" should be stripped from the description label
      expect(result).toContain("github/gh-aw#21982");
      expect(result).not.toContain("issue:github/gh-aw#21982");
    });

    it("should show at most 16 items and ellipse the rest", () => {
      const events = Array.from({ length: 20 }, (_, i) => ({
        type: "DIFC_FILTERED",
        tool_name: `tool_${i}`,
        reason: "reason",
        html_url: `https://github.com/org/repo/issues/${i + 1}`,
        number: String(i + 1),
      }));

      const result = generateDifcFilteredSection(events);

      // Summary still shows the total count
      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 20 items</b></summary>");
      // First 16 items rendered
      expect(result).toContain("[#1](https://github.com/org/repo/issues/1)");
      expect(result).toContain("[#16](https://github.com/org/repo/issues/16)");
      // Items 17-20 not rendered individually
      expect(result).not.toContain("[#17]");
      // Ellipsis line present
      expect(result).toContain("... and 4 more items");
    });

    it("should not show ellipsis when 16 or fewer items", () => {
      const events = Array.from({ length: 16 }, (_, i) => ({
        type: "DIFC_FILTERED",
        tool_name: `tool_${i}`,
        reason: "reason",
      }));

      const result = generateDifcFilteredSection(events);

      expect(result).not.toContain("more item");
    });

    it("should use singular form in ellipsis for exactly 1 remaining item", () => {
      const events = Array.from({ length: 17 }, (_, i) => ({
        type: "DIFC_FILTERED",
        tool_name: `tool_${i}`,
        reason: "reason",
      }));

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("... and 1 more item");
      expect(result).not.toContain("... and 1 more items");
    });

    it("should show events with #unknown description using tool_name instead", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "search_issues",
          description: "github:#unknown",
          reason: "has lower integrity than agent requires.",
        },
        {
          type: "DIFC_FILTERED",
          tool_name: "list_issues",
          reason: "Integrity check failed",
          html_url: "https://github.com/org/repo/issues/42",
          number: "42",
        },
      ];

      const result = generateDifcFilteredSection(events);

      // Both entries should be shown; #unknown text hidden, tool_name used instead
      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 2 items</b></summary>");
      expect(result).not.toContain("#unknown");
      expect(result).toContain("`search_issues`");
      expect(result).toContain("[#42](https://github.com/org/repo/issues/42)");
    });

    it("should show entry using tool_name when description is #unknown", () => {
      const events = [
        {
          type: "DIFC_FILTERED",
          tool_name: "search_issues",
          description: "github:#unknown",
          reason: "has lower integrity",
        },
      ];

      const result = generateDifcFilteredSection(events);

      expect(result).toContain("> <summary><b>🔒 Integrity filter blocked 1 item</b></summary>");
      expect(result).toContain("`search_issues`");
      expect(result).not.toContain("#unknown");
    });
  });
});
