// @ts-check
import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { redactSafeOutputContent } = require("./safe_output_redaction.cjs");
const { applyAddMaskRedaction } = require("./add_mask_redaction.cjs");
const tools = require("./safe_outputs_tools.json");

function redact(file, content, masks) {
  return redactSafeOutputContent(file, content, masks, text => applyAddMaskRedaction(text, masks));
}

describe("safe-output schema redaction", () => {
  it.each(["b", "body", "submit_pull_request_review", "COMMENT", "pull_request_number"])("preserves public review vocabulary while redacting the same text in payloads (%s)", mask => {
    const message = { type: "submit_pull_request_review", event: "COMMENT", pull_request_number: 12, body: `prefix ${mask} suffix` };
    const content = JSON.stringify(message) + "\n";
    expect(JSON.parse(redact("safeoutputs.jsonl", content, [mask]))).toEqual({ ...message, body: "prefix *** suffix" });
    expect(JSON.parse(redact("agent_output.json", JSON.stringify({ items: [message], errors: [] }), [mask]))).toEqual({ items: [{ ...message, body: "prefix *** suffix" }], errors: [] });
  });

  it("preserves only declared nested keys, never arbitrary payload keys or values", () => {
    const message = { type: "ledger_append", record: { body: "body", nested: [{ body: "body" }] } };
    expect(JSON.parse(redact("safeoutputs.jsonl", JSON.stringify(message), ["body"]))).toEqual({ type: "ledger_append", record: { "***": "***", nested: [{ "***": "***" }] } });
  });

  it.each(["body", "type", "event", "items", "COMMENT"])("does not treat public words as exempt inside free-form content (%s)", mask => {
    const message = { type: "add_comment", body: `before ${mask} after` };
    const result = JSON.parse(redact("safeoutputs.jsonl", JSON.stringify(message), [mask]));
    expect(result).toEqual({ type: "add_comment", body: "before *** after" });
  });

  it("redacts escaped strings and original high-precision numeric tokens", () => {
    const content = '{"type":"create_pull_request_review_comment","body":"\\u006fpaque","path":"src/file.js","line":9007199254740993}';
    expect(JSON.parse(redact("safeoutputs.jsonl", content, ["opaque", "9007199254740993"]))).toEqual({
      type: "create_pull_request_review_comment",
      body: "***",
      path: "src/file.js",
      line: "***",
    });
  });

  it("retains unchanged JSON/JSONL formatting and large records", () => {
    const content = ' \t{"type":"add_comment", "body":"' + "x".repeat(10 * 1024 * 1024) + '"}\r\n\n';
    expect(redact("safeoutputs.jsonl", content, ["absent"])).toBe(content);
    const masked = redact("safeoutputs.jsonl", content, ["xxx"]);
    expect(JSON.parse(masked.trim()).type).toBe("add_comment");
    expect(JSON.parse(masked.trim()).body).not.toContain("xxx");
  });

  it.each(["\n", "\r\n"])("preserves blank lines and record boundaries while redacting multiple records (%j)", newline => {
    const first = '{"type":"add_comment","body":"opaque"}';
    const second = ' \t{"type":"add_comment", "body":"unchanged"}';
    const content = newline + first + newline + " \t" + newline + second + newline;
    expect(redact("safeoutputs.jsonl", content, ["absent"])).toBe(content);
    expect(redact("safeoutputs.jsonl", content, ["opaque"])).toBe(newline + '{"type":"add_comment","body":"***"}' + newline + " \t" + newline + second + newline);
  });

  it.each(["", "\n", "\r\n \t\n"])("preserves empty or whitespace-only output (%j)", content => {
    expect(redact("safeoutputs.jsonl", content, ["opaque"])).toBe(content);
  });

  it("rejects a malformed record after valid records and blank lines", () => {
    const content = '{"type":"add_comment","body":"opaque"}\n \t\n{"type":"add_comment","body":';
    expect(() => redact("safeoutputs.jsonl", content, ["opaque"])).toThrow("Invalid JSON safe-output record during secret redaction");
  });

  it("keeps unknown custom types and fields only when redaction does not alter their routing", () => {
    const content = '{"type":"custom_job","custom_input":"opaque"}';
    expect(JSON.parse(redact("safeoutputs.jsonl", content, ["opaque"]))).toEqual({ type: "custom_job", custom_input: "***" });
    expect(() => redact("safeoutputs.jsonl", content, ["custom_job"])).toThrow("Secret redaction would change a custom safe-output type");
    expect(() => redact("safeoutputs.jsonl", content, ["custom_input"])).toThrow("Secret redaction would change a custom safe-output field");
  });

  it.each(['{"type":"add_comment","body":"opaque"', '{"body":"opaque"}', "null", "[]", '"opaque"'])("rejects malformed or invalid executable records instead of using mixed-log recovery (%s)", content => {
    expect(() => redact("safeoutputs.jsonl", content, ["opaque"])).toThrow();
  });

  it("rejects an invalid agent-output envelope and propagates redactor failures", () => {
    expect(() => redact("agent_output.json", '{"items":{},"body":"opaque"}', ["opaque"])).toThrow("Invalid agent-output envelope");
    expect(() =>
      redactSafeOutputContent("safeoutputs.jsonl", '{"type":"add_comment","body":"opaque"}', ["opaque"], () => {
        throw new Error("synthetic redactor failure");
      })
    ).toThrow("synthetic redactor failure");
  });

  it("never exempts unknown enum values or fields from masking", () => {
    expect(JSON.parse(redact("safeoutputs.jsonl", '{"type":"submit_pull_request_review","event":"opaque","body":"opaque"}', ["opaque"]))).toEqual({
      type: "submit_pull_request_review",
      event: "***",
      body: "***",
    });
    expect(() => redact("safeoutputs.jsonl", '{"type":"add_comment","opaque":"text","body":"text"}', ["opaque"])).toThrow("Secret redaction would change a custom safe-output field");
  });

  it("preserves declared aliases without exempting their values", () => {
    const content = '{"type":"submit_pull_request_review","action":"COMMENT","pullRequestNumber":"12","body":"opaque"}';
    expect(JSON.parse(redact("safeoutputs.jsonl", content, ["action", "pullRequestNumber", "opaque"]))).toEqual({
      type: "submit_pull_request_review",
      action: "COMMENT",
      pullRequestNumber: "12",
      body: "***",
    });
  });

  it("does not enable protocol exceptions for arbitrary artifacts or similarly named files", () => {
    for (const file of ["log.json", "agent_output.json.bak", "safeoutputs.txt", "safe-output-items.jsonl"]) {
      expect(redact(file, '{"type":"add_comment","body":"opaque"}', ["opaque"])).toBeNull();
    }
  });

  it("keeps public configuration and validation vocabulary without disabling real constraints", () => {
    const config = '{"submit_pull_request_review":{"max":1,"allowed_events":["COMMENT"]}}';
    const validation = '{"submit_pull_request_review":{"defaultMax":1,"fields":{"body":{"type":"string","maxLength":65000},"event":{"type":"string","enum":["COMMENT"]}}}}';
    expect(redact("/tmp/test/safeoutputs/config.json", config, ["b"])).toBe(config);
    expect(redact("/tmp/test/safeoutputs/validation.json", validation, ["b"])).toBe(validation);
    expect(() => redact("/tmp/test/safeoutputs/config.json", config, ["1"])).toThrow("safe-output constraint");
    expect(() => redact("/tmp/test/safeoutputs/config.json", '{"add_comment":{"repo":"opaque/repo"}}', ["opaque"])).toThrow("safe-output control data");
    expect(() => redact("/tmp/test/safeoutputs/config.json", '{"custom_job":{"max":1}}', ["custom_job"])).toThrow("custom safe-output control field");
    expect(() => redact("/tmp/test/safeoutputs/config.json", '{"add_comment":{"max":9007199254740993}}', ["9007199254740993"])).toThrow("safe-output constraint");
  });

  it("covers each shipped tool and property against seeded short-mask collisions", () => {
    let state = 0x5eed;
    for (const tool of tools) {
      const properties = tool.inputSchema.properties || {};
      for (const key of Object.keys(properties)) {
        state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
        const vocabulary = tool.name + key;
        const mask = vocabulary[state % vocabulary.length];
        const item = { type: tool.name, [key]: `payload ${mask} payload` };
        const result = JSON.parse(redact("safeoutputs.jsonl", JSON.stringify(item), [mask]));
        expect(result.type, `${tool.name}.${key}`).toBe(tool.name);
        expect(Object.hasOwn(result, key), `${tool.name}.${key}`).toBe(true);
        expect(result[key], `${tool.name}.${key}`).toBe(applyAddMaskRedaction(item[key], [mask]));
      }
    }
  });
});
