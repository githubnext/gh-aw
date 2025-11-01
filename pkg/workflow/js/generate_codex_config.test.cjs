// @ts-check
const { describe, it, expect, beforeEach, afterEach } = require("vitest");
const fs = require("fs");
const path = require("path");

// Mock @actions/core
global.core = {
  info: () => {},
  setFailed: () => {},
  summary: {
    addHeading: () => global.core.summary,
    addRaw: () => global.core.summary,
    write: () => {},
  },
};

describe("generate_codex_config", () => {
  const testOutputPath = "/tmp/test-codex-config.toml";

  beforeEach(() => {
    // Clean up any previous test files
    if (fs.existsSync(testOutputPath)) {
      fs.unlinkSync(testOutputPath);
    }
  });

  afterEach(() => {
    // Clean up test files
    if (fs.existsSync(testOutputPath)) {
      fs.unlinkSync(testOutputPath);
    }
    // Clear environment variables
    delete process.env.GH_AW_MCP_CONFIG_JSON;
    delete process.env.GH_AW_MCP_CONFIG;
  });

  it("should generate basic TOML config with history", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {},
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    // Load and execute the script
    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    // Verify the file was created
    expect(fs.existsSync(testOutputPath)).toBe(true);

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("[history]");
    expect(content).toContain('persistence = "none"');
  });

  it("should generate GitHub MCP server config in local mode", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {
        github: {
          command: "docker",
          args: ["run", "-i", "--rm", "-e", "GITHUB_PERSONAL_ACCESS_TOKEN", "-e", "GITHUB_TOOLSETS=default", "ghcr.io/github/github-mcp-server:latest"],
          user_agent: "test-workflow",
          startup_timeout_sec: 300,
          tool_timeout_sec: 60,
          env: {
            GITHUB_PERSONAL_ACCESS_TOKEN: "${{ secrets.GITHUB_TOKEN }}",
          },
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("[mcp_servers.github]");
    expect(content).toContain('command = "docker"');
    expect(content).toContain('user_agent = "test-workflow"');
    expect(content).toContain("startup_timeout_sec = 300");
    expect(content).toContain("tool_timeout_sec = 60");
    expect(content).toContain("[mcp_servers.github.env]");
    expect(content).toContain('GITHUB_PERSONAL_ACCESS_TOKEN = "${{ secrets.GITHUB_TOKEN }}"');
  });

  it("should generate GitHub MCP server config in remote mode", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {
        github: {
          type: "http",
          url: "https://api.githubcopilot.com/mcp/",
          bearer_token_env_var: "GH_AW_GITHUB_TOKEN",
          user_agent: "test-workflow",
          startup_timeout_sec: 300,
          tool_timeout_sec: 60,
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("[mcp_servers.github]");
    expect(content).toContain('url = "https://api.githubcopilot.com/mcp/"');
    expect(content).toContain('bearer_token_env_var = "GH_AW_GITHUB_TOKEN"');
    expect(content).toContain('user_agent = "test-workflow"');
    expect(content).not.toContain('command = "docker"');
  });

  it("should generate Playwright MCP server config", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {
        playwright: {
          command: "npx",
          args: ["@playwright/mcp@latest", "--output-dir", "/tmp/gh-aw/mcp-logs/playwright", "--allowed-origins", "example.com;github.com"],
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("[mcp_servers.playwright]");
    expect(content).toContain('command = "npx"');
    expect(content).toContain('"@playwright/mcp@latest"');
    expect(content).toContain('"--allowed-origins"');
  });

  it("should generate multiple MCP servers", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {
        github: {
          command: "docker",
          args: ["run", "-i", "--rm"],
          user_agent: "test",
        },
        playwright: {
          command: "npx",
          args: ["@playwright/mcp@latest"],
        },
        "safe-outputs": {
          command: "node",
          args: ["/tmp/gh-aw/safeoutputs/mcp-server.cjs"],
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("[mcp_servers.github]");
    expect(content).toContain("[mcp_servers.playwright]");
    expect(content).toContain('[mcp_servers."safe-outputs"]');
  });

  it("should handle custom configuration", () => {
    const config = {
      history: {
        persistence: "none",
      },
      mcp_servers: {},
      custom_config: "[custom]\nvalue = 42\n",
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain("# Custom configuration");
    expect(content).toContain("[custom]");
    expect(content).toContain("value = 42");
  });

  it("should escape special characters in strings", () => {
    const config = {
      mcp_servers: {
        test: {
          command: 'test"with"quotes',
          args: ["backslash\\path", "newline\nhere"],
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    delete require.cache[require.resolve("./generate_codex_config.cjs")];
    require("./generate_codex_config.cjs");

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain('\\"'); // Escaped quotes
    expect(content).toContain("\\\\"); // Escaped backslash
    expect(content).toContain("\\n"); // Escaped newline
  });
});
