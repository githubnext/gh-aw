import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createRequire } from "node:module";
import { tmpdir } from "node:os";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { parseDocument } from "./work_queue_yaml.cjs";

const require = createRequire(import.meta.url);
const { buildCopilotSDKPermissionHandler } = require("./copilot_sdk_permissions.cjs");
const root = resolve(import.meta.dirname, "../../..");
const names = ["draft-pr-cleanup", "instructions-janitor", "daily-mcp-concurrency-analysis"];
const readWorkflow = name => {
  const source = readFileSync(resolve(root, `.github/workflows/${name}.md`), "utf8");
  const [, frontmatter, prompt] = source.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/);
  const document = parseDocument(frontmatter, { version: "1.2", uniqueKeys: true });
  if (document.errors.length) throw new AggregateError(document.errors, `Invalid ${name} source`);
  const lock = readFileSync(resolve(root, `.github/workflows/${name}.lock.yml`), "utf8");
  const compiled = parseDocument(lock, { version: "1.2", uniqueKeys: true });
  if (compiled.errors.length) throw new AggregateError(compiled.errors, `Invalid ${name} lock`);
  const manifest = JSON.parse(lock.match(/^# gh-aw-manifest: (.+)$/m)[1]);
  const steps = compiled.toJS().jobs.agent.steps;
  const sdkConfig = steps.find(step => step.env?.GH_AW_COPILOT_SDK_TOOL_CONFIG)?.env.GH_AW_COPILOT_SDK_TOOL_CONFIG;
  return { workflow: document.toJS(), prompt, lock, manifest, steps, sdkConfig: sdkConfig && JSON.parse(sdkConfig) };
};

describe("workflow native MCP permission alignment", () => {
  it.each(names)("%s retains native MCP registration without widening authorization", name => {
    const { workflow, prompt, lock, manifest } = readWorkflow(name);
    expect(workflow.tools["cli-proxy"]).toBe(false);
    expect(lock).not.toContain("export GH_AW_MCP_CLI_SERVERS=");
    expect(manifest.mcp_servers.find(server => server.name === "safeoutputs").tools).toContain("report_incomplete");
    expect(prompt).toContain("native");
    expect(prompt).toContain("report_incomplete");
    expect(workflow.permissions.contents).toBe("read");
    expect(workflow.permissions["pull-requests"]).toBe("read");
    expect(workflow.strict).toBe(true);
    expect(lock).not.toContain("--permission-mode bypassPermissions");
    expect(lock).not.toContain("--allow-all-tools");
  });

  it("allows draft reads and safe outputs but rejects the observed compound commands", () => {
    const { workflow, prompt, sdkConfig } = readWorkflow("draft-pr-cleanup");
    expect(workflow.tools.bash).toBe(false);
    expect(workflow["max-tool-denials"]).toBe(3);
    expect(sdkConfig.capabilities).toMatchObject({ bash: false, cliProxy: false, mcp: true });
    const denied = [];
    const handler = buildCopilotSDKPermissionHandler(sdkConfig.permissions, () => ({ kind: "approve-once" }), { onDenied: command => denied.push(command) });
    for (const [serverName, toolName] of [
      ["github", "search_pull_requests"],
      ["safeoutputs", "add_comment"],
      ["safeoutputs", "report_incomplete"],
    ]) {
      expect(handler({ kind: "mcp", serverName, toolName })).toEqual({ kind: "approve-once" });
    }
    for (const fullCommandText of [
      'github search_pull_requests --query "repo:github/gh-aw is:pr is:open is:draft" --perPage 100 | python3 -c "import json"',
      'github search_pull_requests --query "repo:github/gh-aw is:pr is:open is:draft" > /tmp/gh-aw/agent/o.json; python3 -c "import json"',
      'cd /tmp/gh-aw/agent && github search_pull_requests --query "repo:github/gh-aw is:pr is:open is:draft"',
    ]) {
      expect(handler({ kind: "shell", commands: [], fullCommandText }).kind).toBe("reject");
    }
    expect(denied).toHaveLength(3);
    expect(handler({ kind: "mcp", serverName: "undeclared", toolName: "write" }).kind).toBe("reject");
    expect(prompt).toContain("at most 20 results per page");
    expect(prompt).not.toMatch(/"type": "(?:add_labels|add_comment|close_pull_request)"/);
    expect(prompt).toContain("Label removal is not enabled");
  });

  it("routes Janitor publication through native MCP with dontAsk still enforced", () => {
    const { workflow, prompt, lock } = readWorkflow("instructions-janitor");
    expect(lock).toContain("--permission-mode dontAsk");
    expect(lock).toContain("mcp__safeoutputs");
    expect(workflow["safe-outputs"]["create-pull-request"]["allowed-files"]).toEqual([".github/aw/**"]);
    expect(prompt).toContain("Use native `get_latest_release`");
    expect(prompt).toContain("Call native `create_pull_request` once");
    expect(prompt).toContain("Never repeat a denied write in another syntax");
    expect(workflow.tools.bash).not.toContain("jq *");
  });

  it("exposes concurrency script and raw ledger append with real compiled SDK permissions", () => {
    const { workflow, sdkConfig, manifest, steps, prompt } = readWorkflow("daily-mcp-concurrency-analysis");
    expect(workflow["max-tool-denials"]).toBe(3);
    expect(sdkConfig.capabilities).toMatchObject({ bash: true, edit: true, cliProxy: false, mcp: true });
    expect(manifest.mcp_servers.find(server => server.name === "mcpscripts").tools).toContain("concurrency-audit-state");
    const config = JSON.parse(steps.find(step => step.env?.GH_AW_SAFE_OUTPUTS_CONFIG).env.GH_AW_SAFE_OUTPUTS_CONFIG);
    expect(config.ledger_append.ledgers[0]).toMatchObject({ name: "concurrency-audits", max_record_kb: 4, max_patch_kb: 10 });
    const handler = buildCopilotSDKPermissionHandler(sdkConfig.permissions, () => ({ kind: "approve-once" }));
    for (const [serverName, toolName] of [
      ["mcpscripts", "concurrency-audit-state"],
      ["safeoutputs", "ledger_append"],
      ["safeoutputs", "noop"],
      ["safeoutputs", "report_incomplete"],
    ]) {
      expect(handler({ kind: "mcp", serverName, toolName })).toEqual({ kind: "approve-once" });
    }
    expect(handler({ kind: "write", fileName: "/tmp/gh-aw/cache-memory/concurrency-state.json" })).toEqual({ kind: "approve-once" });
    expect(handler({ kind: "shell", commands: [], fullCommandText: workflow.tools.bash[0] })).toEqual({ kind: "approve-once" });
    expect(handler({ kind: "shell", commands: [], fullCommandText: 'mkdir -p /tmp/gh-aw/cache-memory && echo "{}" > /tmp/gh-aw/cache-memory/concurrency-state.json && safeoutputs noop \'{"message":"done"}\'' }).kind).toBe("reject");
    expect(prompt).toContain("Complete the cache read-back and ledger query/submission before calling `noop`");
  });

  it.each(names)("%s produces native gateway config rather than CLI-only config", name => {
    const { workflow, manifest, steps } = readWorkflow(name);
    const directory = mkdtempSync(resolve(tmpdir(), "gh-aw-tool-policy-"));
    try {
      const gateway = resolve(directory, "gateway.json");
      const toolsOutput = resolve(directory, "tools.json");
      const configPath = resolve(directory, "config.json");
      writeFileSync(configPath, steps.find(step => step.env?.GH_AW_SAFE_OUTPUTS_CONFIG).env.GH_AW_SAFE_OUTPUTS_CONFIG);
      const registryResult = spawnSync(process.execPath, [resolve(import.meta.dirname, "generate_safe_outputs_tools.cjs")], {
        env: {
          PATH: process.env.PATH,
          HOME: directory,
          RUNNER_TEMP: directory,
          GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH: resolve(root, "pkg/workflow/js/safe_outputs_tools.json"),
          GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
          GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH: resolve(directory, "tools_meta.json"),
          GH_AW_SAFE_OUTPUTS_TOOLS_PATH: toolsOutput,
          GH_AW_TOOLS_META_JSON: steps.find(step => step.env?.GH_AW_TOOLS_META_JSON).env.GH_AW_TOOLS_META_JSON,
        },
        encoding: "utf8",
        timeout: 10000,
      });
      expect(registryResult.error).toBeUndefined();
      expect(registryResult.status, registryResult.stderr).toBe(0);
      const safeOutputTools = JSON.parse(readFileSync(toolsOutput, "utf8"));
      expect(safeOutputTools.map(tool => tool.name)).toContain("report_incomplete");
      if (name === "daily-mcp-concurrency-analysis") {
        const ledgerAppend = safeOutputTools.find(tool => tool.name === "ledger_append");
        expect(ledgerAppend).toBeDefined();
        expect(ledgerAppend.inputSchema.properties).toHaveProperty("record");
        expect(ledgerAppend.inputSchema.properties).toHaveProperty("ledger");
      }
      const mcpServers = Object.fromEntries(manifest.mcp_servers.map(server => [server.name, { url: `http://fixture.invalid:80/mcp/${server.name}`, tools: server.tools }]));
      mcpServers.safeoutputs.tools = safeOutputTools.map(tool => tool.name);
      writeFileSync(gateway, JSON.stringify({ mcpServers }));
      const result = spawnSync(process.execPath, [resolve(import.meta.dirname, `convert_gateway_config_${workflow.engine.id}.cjs`)], {
        env: {
          PATH: process.env.PATH,
          HOME: directory,
          RUNNER_TEMP: directory,
          MCP_GATEWAY_OUTPUT: gateway,
          MCP_GATEWAY_DOMAIN: "fixture.invalid",
          MCP_GATEWAY_PORT: "80",
          GH_AW_MCP_CLI_SERVERS: "[]",
        },
        encoding: "utf8",
        timeout: 10000,
      });
      expect(result.error).toBeUndefined();
      expect(result.status, result.stderr).toBe(0);
      expect(result.stderr).toContain("0 filtered (CLI-mounted)");
      const output = resolve(directory, workflow.engine.id === "claude" ? "gh-aw/mcp-config/mcp-servers.json" : ".copilot/mcp-config.json");
      const converted = JSON.parse(readFileSync(output, "utf8"));
      expect(Object.keys(converted.mcpServers).sort()).toEqual(Object.keys(mcpServers).sort());
      expect(converted.mcpServers.safeoutputs.url).toBe("http://fixture.invalid:80/mcp/safeoutputs");
      expect(statSync(output).mode & 0o777).toBe(0o600);
      if (workflow.engine.id === "claude") expect(converted.mcpServers.safeoutputs.type).toBe("http");
      else expect(converted.mcpServers.safeoutputs.tools).toContain("report_incomplete");
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });

  it("does not force native MCP scripts through a shell-only runtime prompt", () => {
    const prompt = readFileSync(resolve(root, "actions/setup/md/mcp_cli_tools_with_safeoutputs_prompt.md"), "utf8");
    expect(prompt).toContain("prefer its native functions when they are explicitly exposed in your tool inventory");
    expect(prompt).not.toContain("For `mcpscripts`, always use the CLI commands above");
    expect(prompt).toContain("A CLI example does not grant permission");
  });
});
