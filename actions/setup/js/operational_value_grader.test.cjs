import { describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { executeOperationalValueEvaluator } from "./operational_value_grader.cjs";
import { main, runCustomGraders } from "./custom_graders.cjs";
import { computeGraders } from "./trace_graders.cjs";

const provenance = { component: "agent", phase: "agent", path: "session.jsonl", index: 0 };
const session = [{ type: "tool.execution_complete", data: { success: true }, provenance }];
const source = `#!/usr/bin/env bash
[[ "$1" == "$GH_AW_SESSION_FILE" ]]
jq -s --argjson config "$GH_AW_GRADER_CONFIG" '[{id: "observations", value: (length + $config.offset)}, {id: "missing", value: null}]' "$1"
`;
const definition = (shellScript = source, extra = {}) => ({
  id: "operational-value",
  source: "operational-value",
  run: ".github/graders/score.sh",
  shellScript,
  config: { offset: 2 },
  ...extra,
});

describe("separate-job operational-value Bash", () => {
  it("passes a session file and config without the old request wrapper", () => {
    expect(executeOperationalValueEvaluator(source, session, { offset: 2 })).toEqual([
      { id: "observations", value: 3 },
      { id: "missing", value: null },
    ]);
    expect(session).toHaveLength(1);
  });

  it("keeps legacy metric arrays, using the first as the primary result", () => {
    const { output, manifest } = runCustomGraders(session, [definition(source, { direction: "higher_is_better", threshold: 4 })]);
    expect(output.results[0]).toMatchObject({
      id: "operational-value",
      source: "operational-value",
      value: 3,
      status: "fail",
      diagnostics: { missing: null },
      metrics: [
        { id: "observations", value: 3 },
        { id: "missing", value: null },
      ],
      implementation: { id: "gh-aw/operational-value", version: 2, digest: expect.stringMatching(/^[a-f0-9]{64}$/) },
    });
    expect(manifest.graders[0]).not.toHaveProperty("shellScript");
    expect(manifest.graders[0]).not.toHaveProperty("script");
  });

  it("does not inherit credentials, shell startup hooks, or the working checkout", () => {
    vi.stubEnv("GH_TOKEN", "test-credential");
    vi.stubEnv("BASH_ENV", "/does-not-exist");
    try {
      const script = `#!/bin/bash
[[ -z "\${GH_TOKEN+x}" && -z "\${GITHUB_TOKEN+x}" && -z "\${BASH_ENV+x}" && -z "\${GITHUB_WORKSPACE+x}" ]]
[[ "$PWD" == */gh-aw-operational-value-* ]]
printf '[{"id":"isolated-env","value":1}]'
`;
      expect(executeOperationalValueEvaluator(script, [], {})).toEqual([{ id: "isolated-env", value: 1 }]);
    } finally {
      vi.unstubAllEnvs();
    }
  });

  it.each([
    "printf not-json",
    "printf '[]'",
    'printf \'[{"id":"a","value":"unavailable"}]\'',
    'printf \'[{"id":"a","value":1},{"id":"a","value":2}]\'',
    'printf \'[{"id":"a","value":1,"extra":true}]\'',
    "exit 4",
    "cat /missing-grader-session",
  ])("surfaces invalid output or execution failures: %s", script => {
    expect(() => executeOperationalValueEvaluator(script, [], {})).toThrow(/operational-value/);
    expect(runCustomGraders([], [definition("#!/bin/bash\n" + script)]).output.results[0]).toMatchObject({ status: "error", value: null });
  });

  it("bounds execution and output and removes staging files on errors", () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-shell-test-"));
    try {
      expect(() => executeOperationalValueEvaluator("while :; do :; done", [], {}, { timeout: 50, temporaryRoot: root })).toThrow(/limit/);
      const start = Date.now();
      expect(() => executeOperationalValueEvaluator("sleep 30 & wait", [], {}, { timeout: 50, temporaryRoot: root })).toThrow(/limit/);
      expect(Date.now() - start).toBeLessThan(3000);
      expect(() => executeOperationalValueEvaluator("printf '%1100000s' x", [], {}, { temporaryRoot: root })).toThrow(/limit/);
      expect(fs.readdirSync(root)).toEqual([]);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  it("does not execute disabled shell definitions", () => {
    expect(runCustomGraders([], [definition("#!/bin/bash\nexit 9", { enabled: false })]).output.results).toEqual([]);
  });

  it.each(["#!/usr/bin/bash", "#!/usr/bin/env -S bash -eu", "#!/bin/bash -e", "#!/bin/env bash"])("accepts compiler-supported Bash shebangs: %s", shebang => {
    const script = shebang + '\nprintf \'[{"id":"compatible","value":1}]\'\n';
    expect(runCustomGraders([], [definition(script)]).output.results[0]).toMatchObject({ value: 1, status: "pass" });
  });

  it("publishes combined source-free artifacts and fails the job on shell errors", async () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-shell-artifact-test-"));
    const previousCore = global.core;
    global.core = { warning: vi.fn(), setFailed: vi.fn() };
    try {
      const builtins = computeGraders(session);
      const input = [...session, { type: "grader.manifest", data: builtins.manifest, provenance }, { type: "grader.result", data: builtins.output, provenance }];
      const original = input.map(event => JSON.stringify(event)).join("\n") + "\n";
      const sessionPath = path.join(root, "aw_session.jsonl");
      fs.writeFileSync(sessionPath, original);
      vi.stubEnv("GH_AW_CUSTOM_GRADER_MANIFEST_CHUNKS", "1");
      vi.stubEnv("GH_AW_CUSTOM_GRADER_MANIFEST_0", JSON.stringify({ graders: [definition("#!/bin/bash\nexit 3")] }));
      await main({ sessionPath, outputDirectory: path.join(root, "results") });
      const output = JSON.parse(fs.readFileSync(path.join(root, "results/grader_results.json"), "utf8"));
      expect(output.results.slice(0, builtins.output.results.length)).toEqual(builtins.output.results);
      expect(output.results.at(-1)).toMatchObject({ id: "operational-value", status: "error", source: "operational-value" });
      expect(global.core.setFailed).toHaveBeenCalled();
      expect(global.core.warning).toHaveBeenCalled();
      expect(fs.readFileSync(path.join(root, "results/grader_manifest.json"), "utf8")).not.toContain("shellScript");
      expect(fs.readFileSync(sessionPath, "utf8")).toBe(original);
    } finally {
      vi.unstubAllEnvs();
      global.core = previousCore;
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
