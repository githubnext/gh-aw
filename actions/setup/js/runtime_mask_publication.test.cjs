import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { collectUnifiedSession, writeUnifiedSession } from "./unified_session.cjs";
import { publishUnifiedSessionSummary } from "./unified_session_render.cjs";

const redactScript = fs.readFileSync(path.join(import.meta.dirname, "redact_secrets.cjs"), "utf8");
const bootstrapScript = fs.readFileSync(path.join(import.meta.dirname, "log_parser_bootstrap.cjs"), "utf8");
const collectorScript = fs.readFileSync(path.join(import.meta.dirname, "collect_ndjson_output.cjs"), "utf8");

describe("runtime mask publication boundary", () => {
  let root, otherRoot, originalEnv, originalCore;
  const opaque = 'dynamic"\\credential%0A';
  const secrets = [opaque, "runtime-line-one", "runtime-line-two", "runtime-line-three", "x$"];
  const text = `${opaque} runtime-line-one\nruntime-line-two\rruntime-line-three x$`;

  beforeEach(() => {
    originalEnv = { ...process.env };
    originalCore = global.core;
    root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-mask-publication-"));
    otherRoot = path.join(root, "runner-temp");
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
    };
    delete process.env.GH_AW_SECRET_NAMES;
    delete process.env.GITHUB_STEP_SUMMARY;
    delete process.env.GH_AW_SAFE_OUTPUTS;
  });

  afterEach(() => {
    process.env = originalEnv;
    global.core = originalCore;
    vi.restoreAllMocks();
    fs.rmSync(root, { recursive: true, force: true });
  });

  function write(file, value) {
    const target = path.join(root, file);
    fs.mkdirSync(path.dirname(target), { recursive: true });
    fs.writeFileSync(target, typeof value === "string" ? value : JSON.stringify(value) + "\n");
    return target;
  }

  async function redactSources() {
    const script = redactScript.replace('findFiles("/tmp/gh-aw", targetExtensions)', "findFiles(root, targetExtensions)").replace("findFiles(`${process.env.RUNNER_TEMP}/gh-aw`, targetExtensions)", "findFiles(otherRoot, targetExtensions)");
    await eval(`(async () => { ${script}; await main(); })()`);
  }

  function expectSafe(value) {
    if (typeof value === "string") for (const secret of secrets) expect(value).not.toContain(secret);
    else if (Array.isArray(value)) value.forEach(expectSafe);
    else if (value && typeof value === "object")
      Object.entries(value).forEach(([key, nested]) => {
        expectSafe(key);
        expectSafe(nested);
      });
  }

  it("sanitizes all uploaded sources before bootstrap, then safely collects and renders with no mask handoff", async () => {
    const encoded = opaque.replace(/%/g, "%25");
    const builtin = "ghs_" + "a".repeat(40);
    write("agent-stdio.log", `::add-mask::${encoded}\n::add-mask:: runtime-line-one%0Aruntime-line-two%0Druntime-line-three \n::add-mask::x$\n::add-mask::${builtin}\n${text}\n`);
    const nativePath = write("sandbox/agent/logs/copilot-session-state/session/events.jsonl", { type: "assistant.message", id: opaque, data: { content: text }, timestamp: "2026-10-02T00:00:01Z" });
    const sources = [
      write("mcp-logs/rpc.jsonl", { event: "rpc_request", server_id: opaque, payload: { params: { arguments: { credential: text } } } }),
      write("sandbox/firewall/logs/audit.jsonl", { event: "token_steering", message: text }),
      write("safeoutputs.jsonl", { type: "add_comment", repo: opaque, body: text }),
      write("safe-output-items.jsonl", { type: "add_comment", repo: opaque, url: text }),
      write("safe-output-errors.json", { errors: [{ type: "add_comment", message: text }] }),
      write("experiments/assignments.json", { assignments: { variant: text } }),
      write("agent/graders/grader_results.json", { results: [{ id: opaque, error: text }] }),
      write("evals/evals.jsonl", { id: opaque, answer: text }),
      write("runner-temp/mcp-logs/rpc.jsonl", { payload: { secret: text, builtin } }),
      nativePath,
    ];
    await redactSources();
    expect(core.setFailed).not.toHaveBeenCalled();
    expect(fs.existsSync(path.join(root, "publication-masks.json"))).toBe(false);
    expect(fs.existsSync(path.join(root, "usage/publication-masks.json"))).toBe(false);

    process.env.GH_AW_AGENT_OUTPUT = nativePath;
    const script = bootstrapScript.replaceAll("/tmp/gh-aw", root);
    const parseLog = content => ({ markdown: "native session", logEntries: content.trim().split("\n").map(JSON.parse) });
    await eval(`(async () => { ${script}; await runLogParser({ parseLog, parserName: "Copilot" }); })()`);
    expect(fs.readFileSync(path.join(root, "agent-stdio.log"), "utf8")).not.toContain("::add-mask::");
    expectSafe(JSON.parse(fs.readFileSync(path.join(root, "agent-session.jsonl"), "utf8")));

    for (const useNativeFallback of [false, true]) {
      if (useNativeFallback) fs.unlinkSync(path.join(root, "agent-session.jsonl"));
      expect(collectUnifiedSession({ rootDir: root }).maskedValues).toEqual([]);
      const events = writeUnifiedSession({ rootDir: root });
      expect(events[0].data.version).toBe(1);
      expectSafe(events);
      await publishUnifiedSessionSummary(path.join(root, "usage/aw_session.jsonl"));
      expectSafe(core.info.mock.calls.flat());
      expectSafe(core.summary.addRaw.mock.calls.flat());
      const summaryPath = path.join(root, "step-summary.md");
      process.env.GITHUB_STEP_SUMMARY = summaryPath;
      await publishUnifiedSessionSummary(path.join(root, "usage/aw_session.jsonl"));
      expectSafe(fs.readFileSync(summaryPath, "utf8"));
      delete process.env.GITHUB_STEP_SUMMARY;
    }
    for (const file of sources) expectSafe(JSON.parse(fs.readFileSync(file, "utf8")));
    expect(fs.readFileSync(sources.at(-2), "utf8")).not.toContain(builtin);
  });

  it("sanitizes sources with oversized runtime masks without removing artifacts", async () => {
    const mask = "x".repeat(200000);
    const stdio = write("agent-stdio.log", `::add-mask::${mask}\noutput ${mask}\n`);
    const source = write("safeoutputs.jsonl", { type: "add_comment", body: mask });
    await redactSources();
    expect(core.setFailed).not.toHaveBeenCalled();
    expect(core.warning).not.toHaveBeenCalled();
    expect(fs.readFileSync(stdio, "utf8")).toBe("output ***\n");
    expect(JSON.parse(fs.readFileSync(source, "utf8"))).toEqual({ type: "add_comment", body: "***" });
  });

  it("sanitizes regex-shaped multiline gateway text without removing artifacts", async () => {
    const gatewayAgentId = "runtime-gateway-agent-id";
    const actualGatewayLines = [
      "MCP_GATEWAY_AGENT_ID=$(openssl rand -base64 45 | tr -d '/+=')",
      `echo "::add-mask::${gatewayAgentId}"`,
      "export MCP_GATEWAY_AGENT_ID",
      'export MCP_GATEWAY_PAYLOAD_DIR="/tmp/gh-aw/mcp-payloads"',
      'mkdir -p "/tmp/gh-aw/mcp-payloads"',
    ];
    const mask = actualGatewayLines.join("\n");
    const stdio = write("agent-stdio.log", `::add-mask::${mask.replace(/\n/g, "%0A")}\n${mask}\n`);
    const source = write("safeoutputs.jsonl", { type: "add_comment", body: mask });
    await redactSources();
    expect(core.setFailed).not.toHaveBeenCalled();
    expect(core.warning).not.toHaveBeenCalled();
    expect(fs.readFileSync(stdio, "utf8")).toBe("***\n***\n***\n***\n");
    expect(JSON.parse(fs.readFileSync(source, "utf8"))).toEqual({ type: "add_comment", body: "***\n***\n***\n***" });
  });

  it("redacts runtime masks from malformed mixed records without changing unrelated paths", async () => {
    write("agent-stdio.log", "::add-mask::opaque\n");
    const record = '"bad\t" "\\u006fpaque" C:\\temp\\file.txt';
    const source = write("mcp-logs/rpc.jsonl", record);
    await redactSources();
    expect(core.setFailed).not.toHaveBeenCalled();
    const redacted = fs.readFileSync(source, "utf8").trim();
    expect(redacted).not.toContain("opaque");
    expect(JSON.parse(redacted)).toBe('"bad\t" "***" C:\\temp\\file.txt');
  });

  it("keeps review messages executable after short-mask redaction across live, fallback, and collected outputs", async () => {
    const originalGithub = global.github;
    const originalContext = global.context;
    try {
      core.debug = vi.fn();
      core.setOutput = vi.fn();
      core.exportVariable = vi.fn();
      global.github = { rest: { pulls: { get: vi.fn().mockResolvedValue({ data: {} }) } } };
      global.context = {
        eventName: "pull_request",
        runId: 123,
        repo: { owner: "test-owner", repo: "test-repo" },
        payload: { pull_request: { number: 12, head: { sha: "test-sha" } } },
      };
      const messages = [
        { type: "create_pull_request_review_comment", path: "src/file.js", line: 12, side: "RIGHT", pull_request_number: 12, body: "bad body" },
        { type: "submit_pull_request_review", event: "COMMENT", pull_request_number: 12, body: "bad body" },
      ];
      write("agent-stdio.log", "::add-mask::b\n");
      process.env.GH_AW_SAFE_OUTPUTS = write("runner-temp/safeoutputs/outputs.jsonl", messages.map(item => JSON.stringify(item)).join("\n") + "\n");
      const config = { create_pull_request_review_comment: { max: 10 }, submit_pull_request_review: { max: 1, allowed_events: ["COMMENT"] } };
      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = write("runner-temp/safeoutputs/config.json", config);
      const fields = { body: { required: true, type: "string" }, pull_request_number: { type: "number" } };
      const validation = {
        create_pull_request_review_comment: { defaultMax: 10, fields: { ...fields, path: { type: "string" }, line: { type: "number" }, side: { type: "string", enum: ["LEFT", "RIGHT"] } } },
        submit_pull_request_review: { defaultMax: 1, fields: { ...fields, event: { type: "string", enum: ["COMMENT"] } } },
      };
      process.env.GH_AW_VALIDATION_CONFIG_PATH = write("runner-temp/safeoutputs/validation.json", validation);
      const fallback = write("safeoutputs.jsonl", messages.map(item => JSON.stringify(item)).join("\n") + "\n");
      const collected = write("agent_output.json", { items: messages, errors: [] });
      const generic = write("mcp-logs/rpc.jsonl", messages[1]);
      await redactSources();
      expect(core.setFailed).not.toHaveBeenCalled();
      const expected = messages.map(item => ({ ...item, body: "***ad ***ody" }));
      for (const file of [process.env.GH_AW_SAFE_OUTPUTS, fallback]) {
        expect(fs.readFileSync(file, "utf8").trim().split("\n").map(JSON.parse)).toEqual(expected);
      }
      const output = JSON.parse(fs.readFileSync(collected, "utf8"));
      expect(output).toEqual({ items: expected, errors: [] });
      expect(JSON.parse(fs.readFileSync(generic, "utf8"))).toEqual({ type: "su***mit_pull_request_review", event: "COMMENT", "pull_request_num***er": 12, "***ody": "***ad ***ody" });
      expect(JSON.parse(fs.readFileSync(process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH, "utf8"))).toEqual(config);
      expect(JSON.parse(fs.readFileSync(process.env.GH_AW_VALIDATION_CONFIG_PATH, "utf8"))).toEqual(validation);
      const script = collectorScript
        .replace('const { AGENT_OUTPUT_FILENAME, TMP_GH_AW_PATH } = require("./constants.cjs");', 'const { AGENT_OUTPUT_FILENAME } = require("./constants.cjs"); const TMP_GH_AW_PATH = root;')
        .replaceAll("/tmp/gh-aw", root);
      await eval(`(async () => { ${script}; await main(); })()`);
      const collectedOutput = JSON.parse(fs.readFileSync(collected, "utf8"));
      expect(collectedOutput.errors).toEqual([]);
      expect(collectedOutput.items).toEqual(expected);
      const { createReviewBuffer } = require("./pr_review_buffer.cjs");
      const buffer = createReviewBuffer();
      for (const [index, module] of ["./create_pr_review_comment.cjs", "./submit_pr_review.cjs"].entries()) {
        const handler = await require(module).main({ _prReviewBuffer: buffer });
        expect((await handler(collectedOutput.items[index], {})).success).toBe(true);
      }
      expect(buffer.getBufferedCount()).toBe(1);
      expect(core.warning).not.toHaveBeenCalled();
    } finally {
      global.github = originalGithub;
      global.context = originalContext;
      require("./safe_output_type_validator.cjs").resetValidationConfigCache();
    }
  });

  it("redacts built-in, custom, encoded, and numeric payload secrets without changing protocol keys", async () => {
    const builtin = "ghs_" + "b".repeat(40);
    process.env.GH_AW_SECRET_NAMES = "CUSTOM";
    process.env.SECRET_CUSTOM = "body-value";
    write("agent-stdio.log", "::add-mask::b\n::add-mask::9007199254740993\n");
    const source = write("safeoutputs.jsonl", `{"type":"create_pull_request_review_comment","path":"src/file.js","line":9007199254740993,"body":"${builtin} \\u0062ody-value"}\n`);
    await redactSources();
    expect(core.setFailed).not.toHaveBeenCalled();
    const message = JSON.parse(fs.readFileSync(source, "utf8"));
    expect(message.type).toBe("create_pull_request_review_comment");
    expect(message.line).toBe("***");
    expect(message.body).not.toContain(builtin);
    expect(message.body).not.toContain("body-value");
    expect(message.body).not.toContain("ody-value");
    expect(message.body).not.toContain("b");
  });

  it.each(['{"type":"add_comment","body":"opaque"', '{"type":"opaque_job","body":"opaque"}', '{"type":"add_comment","opaque_field":"value","body":"opaque"}'])(
    "removes unsafe executable output instead of publishing a malformed or rerouted message",
    content => {
      const { processFile } = require("./redact_secrets.cjs");
      const source = write("safeoutputs.jsonl", content);
      expect(processFile(source, [], ["opaque"])).toBe(0);
      expect(fs.existsSync(source)).toBe(false);
      expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Removed artifact source after secret redaction failed"));
      expect(core.warning.mock.calls.flat().join("\n")).not.toContain("opaque");
    }
  );

  it("removes a source on failed runtime-mask writes without logging masks and continues sanitizing the other files", async () => {
    write("agent-stdio.log", `::add-mask::${opaque.replace(/%/g, "%25")}\n`);
    const failing = write("mcp-logs/first.jsonl", { credential: opaque });
    const remaining = write("safeoutputs.jsonl", { type: "add_comment", body: opaque });
    const originalWrite = fs.writeFileSync;
    vi.spyOn(fs, "writeFileSync").mockImplementation((file, ...args) => {
      if (file === failing) throw new Error(`write denied: ${opaque}`);
      return originalWrite(file, ...args);
    });
    await redactSources();
    expect(fs.existsSync(failing)).toBe(false);
    expectSafe(JSON.parse(fs.readFileSync(remaining, "utf8")));
    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("Removed artifact source after secret redaction failed"));
    expectSafe(core.warning.mock.calls.flat());
    expectSafe(core.setFailed.mock.calls.flat());
  });

  it("removes publication sources if stdio masks cannot be read", async () => {
    const stdio = write("agent-stdio.log", `::add-mask::${opaque.replace(/%/g, "%25")}\n`);
    const source = write("mcp-logs/rpc.jsonl", { credential: opaque });
    const originalRead = fs.readFileSync;
    vi.spyOn(fs, "readFileSync").mockImplementation((file, ...args) => {
      if (file === stdio) throw new Error("read denied");
      return originalRead(file, ...args);
    });
    await redactSources();
    expect(fs.existsSync(stdio)).toBe(false);
    expect(fs.existsSync(source)).toBe(false);
    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("runtime mask collection failed"));
  });
});
