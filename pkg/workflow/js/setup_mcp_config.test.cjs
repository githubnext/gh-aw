import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global variables
global.core = mockCore;

describe("setup_mcp_config.cjs", () => {
  let setupScript;
  let tempDir;
  let originalCwd;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create a temporary directory for testing
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "mcp-test-"));
    originalCwd = process.cwd();
    originalEnv = { ...process.env };

    // Change to temp directory
    process.chdir(tempDir);

    // Read the script content
    const scriptPath = path.join(
      originalCwd,
      "pkg/workflow/js/setup_mcp_config.cjs"
    );
    setupScript = fs.readFileSync(scriptPath, "utf8");

    // Make fs available globally for the evaluated script
    global.fs = fs;
    global.require = require;
  });

  afterEach(() => {
    // Restore original working directory and environment
    process.chdir(originalCwd);
    process.env = originalEnv;

    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    // Clean up globals
    delete global.fs;
    delete global.require;
  });

  describe("main function", () => {
    it("should generate MCP config for GitHub tool", async () => {
      const toolsConfig = {
        github: {
          allowed: ["list_issues", "create_issue"],
          docker_image_version: "v1.2.3"
        }
      };

      process.env.GITHUB_AW_TOOLS_CONFIG = JSON.stringify(toolsConfig);

      // Execute the script
      eval(setupScript);

      // Check that info was logged
      expect(mockCore.info).toHaveBeenCalledWith(
        expect.stringContaining("Generated MCP configuration at:")
      );

      // Check that setOutput was called
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "mcp_config_path",
        "/tmp/mcp-config/mcp-servers.json"
      );

      // Check that the file was created with correct content
      const configPath = "/tmp/mcp-config/mcp-servers.json";
      expect(fs.existsSync(configPath)).toBe(true);

      const configContent = JSON.parse(fs.readFileSync(configPath, "utf8"));
      expect(configContent).toEqual({
        mcpServers: {
          github: {
            command: "docker",
            args: [
              "run",
              "-i",
              "--rm",
              "-e",
              "GITHUB_PERSONAL_ACCESS_TOKEN",
              "ghcr.io/github/github-mcp-server:v1.2.3"
            ],
            env: {
              GITHUB_PERSONAL_ACCESS_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
            }
          }
        }
      });
    });

    it("should generate MCP config for time MCP from direct configuration", async () => {
      const toolsConfig = {
        "time": {
          mcp: {
            type: "stdio",
            command: "npx",
            args: ["-y", "mcp-time-server"],
            env: {
              TIMEZONE: "UTC"
            }
          },
          allowed: ["current_time", "set_timezone"]
        }
      };

      process.env.GITHUB_AW_TOOLS_CONFIG = JSON.stringify(toolsConfig);

      eval(setupScript);

      const configPath = "/tmp/mcp-config/mcp-servers.json";
      const configContent = JSON.parse(fs.readFileSync(configPath, "utf8"));
      
      expect(configContent.mcpServers.time).toEqual({
        command: "npx",
        args: ["-y", "mcp-time-server"],
        env: {
          TIMEZONE: "UTC"
        }
      });
    });

    it("should fail when GITHUB_AW_TOOLS_CONFIG is not set", async () => {
      delete process.env.GITHUB_AW_TOOLS_CONFIG;

      eval(setupScript);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "GITHUB_AW_TOOLS_CONFIG environment variable not set"
      );
    });
  });
});