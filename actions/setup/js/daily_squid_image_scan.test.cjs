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
const gate = source
  .split("  - name: Enforce vendored vulnerability and license gates\n")[1]
  .split("timeout-minutes:")[0]
  .split("    run: |\n")[1]
  .replace(/^      /gm, "");

describe("daily container image scan analysis", () => {
  it("uses the same immutable Node image pin in the compiler caches and compiled manifest", () => {
    const image = "ghcr.io/github/gh-aw-node";
    const cachePaths = ["../../../.github/aw/actions-lock.json", "../../../pkg/actionpins/data/action_pins.json", "../../../pkg/workflow/data/action_pins.json"];
    const pins = cachePaths.map(path => JSON.parse(readFileSync(new URL(path, import.meta.url), "utf8")).containers[image].pinned_image);
    expect(pins[0]).toMatch(/^ghcr\.io\/github\/gh-aw-node@sha256:[a-f0-9]{64}$/);
    expect(new Set(pins).size).toBe(1);
    const manifest = JSON.parse(lock.match(/^# gh-aw-manifest: (.+)$/m)[1]);
    expect(manifest.containers.find(container => container.image === image).pinned_image).toBe(pins[0]);
  });

  it("classifies the locally built Node image as vendored without promising automatic release upgrades", () => {
    expect(source).toContain("`ghcr.io/github/gh-aw-node` is **vendored**");
    expect(source).toContain("actions/setup/js/Dockerfile.safe-outputs-mcp");
    expect(source).toContain("it does not rebuild images, select newer release tags, or publish a pin");
    expect(source).toContain("tag/digest has been updated and rescanned");
  });

  it("consolidates exact-title family duplicates without closing canonical or operational trackers", () => {
    const closeConfig = source.split("  close-issue:\n")[1].split("  noop:\n")[0];
    expect(closeConfig).toContain('required-title-prefix: "[container-image-scan] "');
    expect(closeConfig).toContain("required-labels: [cookie, security]");
    expect(closeConfig).toContain("state-reason: duplicate");
    expect(source).toContain("preserve unique unresolved findings and tracking links");
    expect(source).toContain("update that issue before closing any duplicate");
    expect(source).toContain("Close only exact-title duplicates");
    expect(source).toContain("`duplicate_of` set to the canonical issue");
    expect(source).toContain("canonical tracker, #52657, or operational-failure issues");
  });

  it("wires the same gates and duplicate constraints into the compiled workflow", () => {
    const compiledGate = lock.match(/name: Enforce vendored vulnerability and license gates\n\s+run: (".+")/);
    expect(compiledGate).not.toBeNull();
    expect(JSON.parse(compiledGate[1]).trim()).toBe(gate.trim());
    const configLine = lock.match(/^\s+GH_AW_SAFE_OUTPUTS_CONFIG: (".+")$/m);
    expect(configLine).not.toBeNull();
    const closeConfig = JSON.parse(JSON.parse(configLine[1])).close_issue;
    expect(closeConfig.required_title_prefix).toBe("[container-image-scan] ");
    expect(closeConfig.required_labels).toEqual(["cookie", "security"]);
    expect(closeConfig.state_reason).toBe("duplicate");
  });

  it.each([
    ["ghcr.io/github/gh-aw-node@sha256:abc:1:1: warning: [High] CVE-2026-1234: pkg@1 (fix: 2)", 1],
    ["ghcr.io/github/gh-aw-node:latest:1:1: error: [Critical] CVE-2026-1234: pkg@1", 1],
    ["ghcr.io/github/gh-aw-node@sha256:abc:1:1: warning: [High] CVE-2026-1234: pkg@1", 0],
    ["ghcr.io/github/gh-aw-node@sha256:abc:1:1: warning: [Medium] CVE-2026-1234: pkg@1 (fix: 2)", 0],
    ["node:lts-alpine:1:1: warning: [High] CVE-2026-1234: pkg@1 (fix: 2)", 0],
    ["ghcr.io/github/gh-aw-mcpg:v1:1:1: error: [Critical] CVE-2026-1234: pkg@1 (fix: 2)", 0],
    ["ghcr.io/github/gh-aw-node-other:1:1:1: warning: [High] CVE-2026-1234: pkg@1 (fix: 2)", 0],
    ["ghcr.io/github/gh-aw-node@sha256:abc:1:1: error: license policy violation: pkg@1", 1],
  ])("enforces vendored gates for %s", (finding, status) => {
    const directory = mkdtempSync(join(tmpdir(), "image-gate-"));
    try {
      const fixture = join(directory, "compile-output.txt");
      writeFileSync(fixture, `${finding}\n`);
      const result = spawnSync("bash", ["-c", gate.replaceAll(scanLog, fixture)], { encoding: "utf8", timeout: 10000 });
      expect(result.error).toBeUndefined();
      expect(result.status, result.stdout + result.stderr).toBe(status);
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });

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
