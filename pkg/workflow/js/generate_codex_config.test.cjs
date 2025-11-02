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

// Import the functions from the script
const { escapeTOMLString, escapeTOMLKey, toTOMLValue, renderMCPServer, generateCodexConfig, main } = require("./generate_codex_config.cjs");

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

    // Execute main function
    main();

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

    main();

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

    main();

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

    main();

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

    main();

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

    main();

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

    main();

    const content = fs.readFileSync(testOutputPath, "utf8");
    expect(content).toContain('\\"'); // Escaped quotes
    expect(content).toContain("\\\\"); // Escaped backslash
    expect(content).toContain("\\n"); // Escaped newline
  });

  it("should properly escape keys with special characters", () => {
    const config = {
      mcp_servers: {
        "server-with-dash": {
          command: "test",
          args: ["arg1"],
        },
        "server.with.dots": {
          command: "test2",
          args: ["arg2"],
        },
        "server with spaces": {
          command: "test3",
          args: ["arg3"],
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    main();

    const content = fs.readFileSync(testOutputPath, "utf8");
    // Keys with dashes should be quoted
    expect(content).toContain('[mcp_servers."server-with-dash"]');
    // Keys with dots should be quoted
    expect(content).toContain('[mcp_servers."server.with.dots"]');
    // Keys with spaces should be quoted
    expect(content).toContain('[mcp_servers."server with spaces"]');
  });

  it("should properly escape environment variable keys with special characters", () => {
    const config = {
      mcp_servers: {
        test: {
          command: "node",
          args: ["server.js"],
          env: {
            NORMAL_KEY: "value1",
            "KEY-WITH-DASH": "value2",
            "KEY.WITH.DOT": "value3",
            "KEY WITH SPACE": "value4",
          },
        },
      },
    };

    process.env.GH_AW_MCP_CONFIG_JSON = JSON.stringify(config);
    process.env.GH_AW_MCP_CONFIG = testOutputPath;

    main();

    const content = fs.readFileSync(testOutputPath, "utf8");
    // Normal keys don't need quotes
    expect(content).toContain("NORMAL_KEY =");
    // Keys with special chars should be quoted
    expect(content).toContain('"KEY-WITH-DASH" =');
    expect(content).toContain('"KEY.WITH.DOT" =');
    expect(content).toContain('"KEY WITH SPACE" =');
  });

  it("should test escapeTOMLKey function directly", () => {
    // Simple keys that don't need quoting
    expect(escapeTOMLKey("simple")).toBe("simple");
    expect(escapeTOMLKey("simple_key")).toBe("simple_key");
    expect(escapeTOMLKey("simple-key")).toBe("simple-key");
    
    // Keys that need quoting
    expect(escapeTOMLKey("key.with.dots")).toBe('"key.with.dots"');
    expect(escapeTOMLKey("key with spaces")).toBe('"key with spaces"');
    expect(escapeTOMLKey("key=value")).toBe('"key=value"');
    expect(escapeTOMLKey("key[bracket]")).toBe('"key[bracket]"');
    
    // Keys starting with numbers need quoting
    expect(escapeTOMLKey("123key")).toBe('"123key"');
    
    // Keys with quotes should be escaped
    expect(escapeTOMLKey('key"with"quotes')).toBe('"key\\"with\\"quotes"');
  });
});
