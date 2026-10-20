import { afterAll } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-test-safeoutputs-"));
const cli = path.join(tempDir, "safeoutputs");
fs.copyFileSync(fileURLToPath(new URL("./fixtures/safeoutputs.cjs", import.meta.url)), cli);
fs.chmodSync(cli, 0o700);

// A local output path does not rebind the real CLI's production MCP gateway.
// Isolate both the CLI override and PATH for tests and their subprocesses.
process.env.GH_AW_SAFEOUTPUTS_CLI = cli;
process.env.PATH = `${tempDir}${path.delimiter}${process.env.PATH || ""}`;
process.env.GH_AW_SAFE_OUTPUTS = path.join(tempDir, "safeoutputs.jsonl");
delete process.env.GH_AW_AWF_EXECUTION_EVIDENCE_FILE;
delete process.env.GH_AW_AWF_EXECUTION_COMPONENT;

afterAll(() => fs.rmSync(tempDir, { recursive: true, force: true }));
