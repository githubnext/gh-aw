import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

const require = createRequire(import.meta.url);
const { buildCopilotSDKPermissionHandler } = require("./copilot_sdk_permissions.cjs");
const { parseCopilotSDKToolConfig } = require("./copilot_sdk_tool_config.cjs");

const scanLog = "/tmp/gh-aw/agent/image-scan/compile-output.txt";
const source = readFileSync(new URL("../../../.github/workflows/daily-squid-image-scan.md", import.meta.url), "utf8");
const lock = readFileSync(new URL("../../../.github/workflows/daily-squid-image-scan.lock.yml", import.meta.url), "utf8");
const commands = [...source.matchAll(/```bash\n([\s\S]*?)\n```/g)].flatMap(match => match[1].trim().split("\n"));
const configLine = lock.match(/^\s+GH_AW_COPILOT_SDK_TOOL_CONFIG: '(.+)'$/m);
const config = parseCopilotSDKToolConfig(configLine?.[1]);
const onPermission = buildCopilotSDKPermissionHandler(config.permissions, () => {
  throw new Error("The scanner must not rely on unrestricted permission approval");
});

describe("daily container image scan analysis", () => {
  it("documents paginated native reads and shell commands allowed by the compiled SDK contract", () => {
    expect(source).toContain("native `grep`");
    expect(source).toContain("native `view` with\n`view_range`");
    expect(source).toContain("Read every\nfinding, including lower severities, license results, and scanner errors");
    expect(source).toContain("Do not retry with another unapproved command");
    expect(onPermission({ kind: "read", path: scanLog })).toEqual({ kind: "approve-once" });
    expect(commands).toHaveLength(2);
    for (const command of commands) {
      expect(command).toContain(scanLog);
      expect(onPermission({ kind: "shell", fullCommandText: command })).toEqual({ kind: "approve-once" });
    }
  });

  it.each(["cd", "cut", "sed", "awk", "python", "python3", "jq"])("keeps %s and unrestricted shell access denied", command => {
    expect(config.permissions.allowedTools).not.toContain("shell");
    expect(onPermission({ kind: "shell", fullCommandText: `${command} ${scanLog}` }).kind).toBe("reject");
  });

  it("summarizes all severities from a large log without merging image tags", () => {
    const directory = mkdtempSync(join(tmpdir(), "image-scan-"));
    try {
      const fixture = join(directory, "compile-output.txt");
      const image = "ghcr.io/example/image";
      const severities = ["Critical", "High", "Medium", "Low", "Negligible", "Unknown"];
      const findings = severities.map(severity => `${image}:v1:1:1: warning: [${severity}] CVE-2026-1234: package@1.0`);
      findings.push(findings[0], `${image}:v2:1:1: error: [Critical] CVE-2026-5678: package@2.0`);
      const lines = [...Array(10000).fill("Compilation progress without image findings.".repeat(2)), ...findings];
      const log = `${lines.join("\n")}\n`;
      expect(Buffer.byteLength(log)).toBeGreaterThan(500000);
      writeFileSync(fixture, log);

      const results = commands.map(command => {
        const result = spawnSync("bash", ["-c", command.replaceAll(scanLog, JSON.stringify(fixture))], {
          encoding: "utf8",
          env: { PATH: process.env.PATH },
          timeout: 10000,
        });
        expect(result.error).toBeUndefined();
        expect(result.status, result.stderr).toBe(0);
        return result.stdout.trim();
      });
      expect(Number.parseInt(results[0], 10)).toBe(lines.length);
      const counts = Object.fromEntries(
        results[1].split("\n").map(line => {
          const [, count, finding] = line.match(/^\s*(\d+) (.+)$/);
          return [finding, Number(count)];
        })
      );
      expect(counts).toEqual({
        ...Object.fromEntries(severities.map(severity => [`${image}:v1:1:1: warning: [${severity}]`, severity === "Critical" ? 2 : 1])),
        [`${image}:v2:1:1: error: [Critical]`]: 1,
      });
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
});
