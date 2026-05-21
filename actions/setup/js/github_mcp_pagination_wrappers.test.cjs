// @ts-check
/**
 * Regression tests for github-mcp-pagination-wrappers
 *
 * Validates that the list_workflows and list_label wrapper scripts correctly
 * pass per_page to the underlying GitHub REST API and return exactly the
 * requested number of items.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createShellHandler } from "./mcp_handler_shell.cjs";
import fs from "fs";
import path from "path";
import os from "os";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Path to the skill scripts relative to the actions/setup/js directory
const repoRoot = path.resolve(__dirname, "../../..");
const workflowsQueryScript = path.join(repoRoot, ".github/skills/github-workflows-query/query-workflows.sh");
const labelsQueryScript = path.join(repoRoot, ".github/skills/github-labels-query/query-labels.sh");

// Helper to create a mock gh command that returns controlled JSON
function createMockGh(tempDir, responseJson) {
  const mockPath = path.join(tempDir, "gh");
  // Write a script that echoes the provided JSON regardless of arguments
  fs.writeFileSync(mockPath, `#!/bin/bash\necho '${responseJson.replace(/'/g, "'\\''")}'\n`);
  fs.chmodSync(mockPath, "755");
  return mockPath;
}

describe("github-mcp-pagination-wrappers", () => {
  let tempDir;
  let mockServer;
  let originalPath;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "pagination-wrappers-test-"));
    mockServer = { debug: () => {}, debugError: () => {} };
    originalPath = process.env.PATH;
  });

  afterEach(() => {
    process.env.PATH = originalPath;
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("list_workflows wrapper (query-workflows.sh)", () => {
    it("skill script exists and is executable", () => {
      expect(fs.existsSync(workflowsQueryScript)).toBe(true);
      const stat = fs.statSync(workflowsQueryScript);
      // Check owner executable bit
      expect(stat.mode & 0o100).toBeTruthy();
    });

    it("per_page=1 returns exactly one workflow", async () => {
      // Mock gh: returns a response with exactly 1 workflow (simulating per_page=1)
      const mockResponse = JSON.stringify({
        total_count: 5,
        workflows: [
          {
            id: 1,
            node_id: "W_kgDO1",
            name: "CI",
            path: ".github/workflows/ci.yml",
            state: "active",
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
            url: "https://api.github.com/repos/testowner/testrepo/actions/workflows/1",
            html_url: "https://github.com/testowner/testrepo/actions/workflows/ci.yml",
            badge_url: "https://github.com/testowner/testrepo/actions/workflows/ci.yml/badge.svg",
          },
        ],
      });
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo", per_page: "1" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);

      expect(data.workflows).toHaveLength(1);
      expect(data.total_count).toBe(5);
      expect(data.per_page).toBe(1);
      expect(data.page).toBe(1);
    });

    it("uses default per_page=10 when not specified", async () => {
      // Mock gh: returns response with 10 workflows
      const workflows = Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        node_id: `W_${i + 1}`,
        name: `Workflow ${i + 1}`,
        path: `.github/workflows/workflow-${i + 1}.yml`,
        state: "active",
        created_at: "2024-01-01T00:00:00Z",
        updated_at: "2024-01-01T00:00:00Z",
        url: `https://api.github.com/repos/o/r/actions/workflows/${i + 1}`,
        html_url: `https://github.com/o/r/actions/workflows/workflow-${i + 1}.yml`,
        badge_url: "",
      }));
      const mockResponse = JSON.stringify({ total_count: 42, workflows });
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);

      expect(data.per_page).toBe(10);
      expect(data.workflows).toHaveLength(10);
    });

    it("returns error on missing owner", async () => {
      createMockGh(tempDir, "{}");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      await expect(handler({ repo: "testrepo" })).rejects.toThrow();
    });

    it("returns error on missing repo", async () => {
      createMockGh(tempDir, "{}");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      await expect(handler({ owner: "testowner" })).rejects.toThrow();
    });

    it("returns error when per_page is out of range", async () => {
      createMockGh(tempDir, "{}");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      await expect(handler({ owner: "testowner", repo: "testrepo", per_page: "101" })).rejects.toThrow();
      await expect(handler({ owner: "testowner", repo: "testrepo", per_page: "0" })).rejects.toThrow();
    });

    it("response includes expected workflow fields", async () => {
      const mockResponse = JSON.stringify({
        total_count: 1,
        workflows: [
          {
            id: 99,
            node_id: "W_99",
            name: "Deploy",
            path: ".github/workflows/deploy.yml",
            state: "active",
            created_at: "2024-06-01T00:00:00Z",
            updated_at: "2024-06-15T00:00:00Z",
            url: "https://api.github.com/repos/o/r/actions/workflows/99",
            html_url: "https://github.com/o/r/actions/workflows/deploy.yml",
            badge_url: "https://github.com/o/r/actions/workflows/deploy.yml/badge.svg",
          },
        ],
      });
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_workflows", workflowsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo", per_page: "1" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);
      const wf = data.workflows[0];

      expect(wf).toHaveProperty("id", 99);
      expect(wf).toHaveProperty("name", "Deploy");
      expect(wf).toHaveProperty("path", ".github/workflows/deploy.yml");
      expect(wf).toHaveProperty("state", "active");
      expect(wf).toHaveProperty("url");
      expect(wf).toHaveProperty("html_url");
      expect(wf).toHaveProperty("badge_url");
    });
  });

  describe("list_label wrapper (query-labels.sh)", () => {
    it("skill script exists and is executable", () => {
      expect(fs.existsSync(labelsQueryScript)).toBe(true);
      const stat = fs.statSync(labelsQueryScript);
      expect(stat.mode & 0o100).toBeTruthy();
    });

    it("per_page=1 returns exactly one label", async () => {
      // Mock gh: returns array with 1 label (simulating per_page=1)
      const mockResponse = JSON.stringify([
        {
          id: 1,
          node_id: "LA_kgDO1",
          url: "https://api.github.com/repos/testowner/testrepo/labels/bug",
          name: "bug",
          color: "d73a4a",
          default: true,
          description: "Something isn't working",
        },
      ]);
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo", per_page: "1" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);

      expect(data.labels).toHaveLength(1);
      expect(data.item_count).toBe(1);
      expect(data.per_page).toBe(1);
      expect(data.page).toBe(1);
    });

    it("uses default per_page=10 when not specified", async () => {
      const labels = Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        node_id: `LA_${i + 1}`,
        url: `https://api.github.com/repos/o/r/labels/label-${i + 1}`,
        name: `label-${i + 1}`,
        color: "0075ca",
        default: false,
        description: `Label ${i + 1}`,
      }));
      const mockResponse = JSON.stringify(labels);
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);

      expect(data.per_page).toBe(10);
      expect(data.labels).toHaveLength(10);
      expect(data.item_count).toBe(10);
    });

    it("returns error on missing owner", async () => {
      createMockGh(tempDir, "[]");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      await expect(handler({ repo: "testrepo" })).rejects.toThrow();
    });

    it("returns error on missing repo", async () => {
      createMockGh(tempDir, "[]");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      await expect(handler({ owner: "testowner" })).rejects.toThrow();
    });

    it("returns error when per_page is out of range", async () => {
      createMockGh(tempDir, "[]");
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      await expect(handler({ owner: "testowner", repo: "testrepo", per_page: "101" })).rejects.toThrow();
      await expect(handler({ owner: "testowner", repo: "testrepo", per_page: "0" })).rejects.toThrow();
    });

    it("response includes expected label fields", async () => {
      const mockResponse = JSON.stringify([
        {
          id: 42,
          node_id: "LA_42",
          url: "https://api.github.com/repos/o/r/labels/enhancement",
          name: "enhancement",
          color: "a2eeef",
          default: true,
          description: "New feature or request",
        },
      ]);
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo", per_page: "1" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);
      const label = data.labels[0];

      expect(label).toHaveProperty("id", 42);
      expect(label).toHaveProperty("name", "enhancement");
      expect(label).toHaveProperty("color", "a2eeef");
      expect(label).toHaveProperty("default", true);
      expect(label).toHaveProperty("description", "New feature or request");
      expect(label).toHaveProperty("url");
    });

    it("handles page parameter correctly", async () => {
      const mockResponse = JSON.stringify([
        {
          id: 11,
          node_id: "LA_11",
          url: "https://api.github.com/repos/o/r/labels/label-11",
          name: "label-11",
          color: "0075ca",
          default: false,
          description: "Page 2 result",
        },
      ]);
      createMockGh(tempDir, mockResponse);
      process.env.PATH = `${tempDir}:${originalPath}`;

      const handler = createShellHandler(mockServer, "list_label", labelsQueryScript, 30);
      const result = await handler({ owner: "testowner", repo: "testrepo", per_page: "1", page: "2" });

      const output = JSON.parse(result.content[0].text);
      const data = JSON.parse(output.stdout);

      expect(data.per_page).toBe(1);
      expect(data.page).toBe(2);
      expect(data.labels).toHaveLength(1);
    });
  });
});
