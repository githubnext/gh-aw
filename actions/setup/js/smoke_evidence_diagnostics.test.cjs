import { describe, it, expect, vi, afterEach } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { recordSmokeFailure, readSmokeFailure, withSmokeEvidence, sanitizeSmokeFailure, routingFailureDescription } from "./smoke_evidence_diagnostics.cjs";
import { buildFailureDiagnosticsContext } from "./handle_agent_failure.cjs";

const roots = [];
function root() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "smoke-diagnostics-test-"));
  roots.push(dir);
  return dir;
}
afterEach(() => {
  vi.restoreAllMocks();
  for (const dir of roots.splice(0)) fs.rmSync(dir, { recursive: true });
});

describe("smoke assertion diagnostics", () => {
  it("identifies a missing routing evidence file or declared sub-agent without copying observed payloads", () => {
    expect(routingFailureDescription("FAIL E2 evidence: missing usage/aw_session.jsonl")).toBe("Unified session usage/aw_session.jsonl is missing");
    expect(routingFailureDescription("FAIL S1 sub-agent mini-whoami: observed PRIVATE payload")).toContain("mini-whoami:");
    expect(routingFailureDescription("FAIL S1 sub-agent PRIVATE: observed PRIVATE payload")).not.toContain("PRIVATE");
  });
  it("records missing evidence and the actual inspected file without exception payloads", async () => {
    const rootDir = root();
    const core = { error: vi.fn() };
    await expect(
      withSmokeEvidence({ rootDir, core, kind: "goose", paths: { gooseResult: "/private/payload.json" } }, async ({ fs: tracked }) => {
        tracked.readFileSync("/private/payload.json");
      })
    ).rejects.toThrow("Smoke evidence file is missing or unreadable");
    const text = readSmokeFailure("Assert Goose smoke evidence", rootDir);
    expect(text).toContain("agent/smoke-test-goose-<run-id>.json");
    expect(text).not.toContain("/private");
    expect(core.error).toHaveBeenCalled();
  });

  it("keeps assertion values, secrets, and JSON parse excerpts out of diagnostics", async () => {
    for (const parseFailure of [true, false]) {
      const rootDir = root();
      const secret = "PRIVATE-AGENT-CONTENT-and-secret";
      await expect(
        withSmokeEvidence({ rootDir, core: { error() {} }, kind: "goose", paths: {} }, async ({ assert, parse }) => {
          if (parseFailure) parse(secret);
          else assert.equal(secret, "expected", "Pull request title does not match GitHub");
        })
      ).rejects.not.toThrow(secret);
      expect(fs.readFileSync(path.join(rootDir, "smoke-evidence.json"), "utf8")).not.toContain(secret);
    }
  });

  it("renders only fixed catalogued entries even for forged artifacts", () => {
    const rootDir = root();
    const file = path.join(rootDir, "smoke-evidence.json");
    for (const record of [
      { kind: "goose", failures: ["secret ``` injected"], files: ["stdio"] },
      { kind: "goose", failures: ["Goose execution failed"], files: ["../../secret"] },
      { kind: "__proto__", failures: ["Goose execution failed"], files: ["stdio"] },
      { kind: { toString: null }, failures: ["Goose execution failed"], files: ["stdio"] },
      { kind: "goose", failures: ["Goose execution failed"], files: [{ toString: null }] },
    ]) {
      fs.writeFileSync(file, JSON.stringify(record));
      expect(readSmokeFailure("Assert Goose smoke evidence", rootDir)).toBe("");
    }
    fs.writeFileSync(file, "x".repeat(16385));
    expect(readSmokeFailure("Assert Goose smoke evidence", rootDir)).toBe("");
    fs.unlinkSync(file);
    fs.symlinkSync(path.join(rootDir, "missing"), file);
    expect(readSmokeFailure("Assert Goose smoke evidence", rootDir)).toBe("");
  });

  it("reconstructs the uploaded artifact and discards unknown payload fields", () => {
    const rootDir = root();
    const file = path.join(rootDir, "smoke-evidence.json");
    fs.writeFileSync(file, JSON.stringify({ kind: "goose", failures: ["Goose execution failed"], files: ["stdio"], payload: "PRIVATE raw logs" }));
    sanitizeSmokeFailure(rootDir);
    expect(fs.readFileSync(file, "utf8")).not.toContain("PRIVATE");
    fs.writeFileSync(file, JSON.stringify({ kind: "goose", failures: ["PRIVATE raw logs"], files: ["stdio"] }));
    sanitizeSmokeFailure(rootDir);
    expect(fs.existsSync(file)).toBe(false);
    fs.mkdirSync(file);
    fs.writeFileSync(path.join(file, "private.log"), "PRIVATE directory payload");
    sanitizeSmokeFailure(rootDir);
    expect(fs.existsSync(file)).toBe(false);
  });

  it("preserves the safe assertion failure when diagnostics cannot be persisted", async () => {
    const rootDir = root();
    fs.writeFileSync(path.join(rootDir, "not-a-directory"), "");
    const core = { error: vi.fn(), warning: vi.fn() };
    await expect(
      withSmokeEvidence({ rootDir: path.join(rootDir, "not-a-directory"), kind: "goose", core, paths: {} }, async ({ assert }) => {
        assert.equal(false, true, "Goose execution failed");
      })
    ).rejects.toThrow("Goose execution failed");
    expect(core.warning).toHaveBeenCalledWith("Unable to persist smoke evidence diagnostics");
  });

  it("does not authorize upload if catalogue reconstruction fails", () => {
    const rootDir = root();
    const file = path.join(rootDir, "smoke-evidence.json");
    const outputFile = path.join(rootDir, "step-output");
    fs.writeFileSync(outputFile, "");
    fs.writeFileSync(file, JSON.stringify({ kind: "goose", failures: ["Goose execution failed"], files: ["stdio"], payload: "PRIVATE raw logs" }));
    vi.spyOn(fs, "mkdtempSync").mockImplementation(() => {
      throw new Error("EACCES");
    });
    expect(() => sanitizeSmokeFailure(rootDir, outputFile)).toThrow("EACCES");
    expect(fs.readFileSync(outputFile, "utf8")).not.toContain("ready=true");
  });

  it("adds the smoke cause to the failure issue context only for its failed step", () => {
    const rootDir = root();
    recordSmokeFailure({ rootDir, kind: "routing", failures: ["Each declared sub-agent must start exactly once"], files: ["session", "proxy"] });
    const read = fs.readFileSync;
    const stat = fs.lstatSync;
    vi.spyOn(fs, "readFileSync").mockImplementation((file, ...args) => read(file === "/tmp/gh-aw/smoke-evidence.json" ? path.join(rootDir, "smoke-evidence.json") : file, ...args));
    vi.spyOn(fs, "lstatSync").mockImplementation((file, ...args) => stat(file === "/tmp/gh-aw/smoke-evidence.json" ? path.join(rootDir, "smoke-evidence.json") : file, ...args));
    const options = { failureCategories: ["agent_failure"], engineFailureContext: "", failingStep: "Assert model-routing and sub-agent evidence" };
    const text = buildFailureDiagnosticsContext(options);
    expect(text).toContain("Each declared sub-agent must start exactly once");
    expect(text).toContain("usage/aw_session.jsonl");
    expect(text).not.toContain("did not record a specific cause");
    expect(buildFailureDiagnosticsContext({ ...options, failingStep: "Execute agent" })).not.toContain("Missing or invalid smoke evidence");
  });
});
