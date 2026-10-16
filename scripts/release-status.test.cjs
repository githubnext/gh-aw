"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");
const { test } = require("node:test");

function step(source, name) {
  const marker = `- name: ${name}\n`;
  assert.ok(source.includes(marker), `Missing step: ${name}`);
  return source.split(marker)[1].split(/\n *- name:/)[0];
}

function block(source, key) {
  const match = source.match(new RegExp(`^( +)${key}: \\|\\n((?:\\1  .*\\n|\\n)+)`, "m"));
  assert.ok(match, `Missing ${key} block`);
  return match[2].replace(new RegExp(`^${match[1]}  `, "gm"), "");
}

const workflows = path.resolve(__dirname, "../.github/workflows");
for (const [file, status] of [
  ["release.md", "pre-release"],
  ["release.lock.yml", "pre-release"],
  ["promote-release.yml", "stable release"],
]) {
  const source = fs.readFileSync(path.join(workflows, file), "utf8");
  const sync = step(source, `Synchronize gh-aw-actions ${status} status`);
  const script = block(sync, "run");
  const prerelease = status === "pre-release";

  function run(t, { token = "test-token", exitCode = 0, tag = "v0.91.9" } = {}) {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "release-status-"));
    t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
    const calls = path.join(directory, "calls.json");
    fs.writeFileSync(
      path.join(directory, "gh"),
      `#!/usr/bin/env node
const fs = require("node:fs");
if (process.env.GH_TOKEN !== "test-token") process.exit(99);
fs.writeFileSync(process.env.CALL_LOG, JSON.stringify(process.argv.slice(2)));
process.exit(Number(process.env.EXIT_CODE));
`,
      { mode: 0o755 }
    );
    const result = spawnSync("bash", ["-e", "-c", script], {
      cwd: directory,
      env: {
        PATH: `${directory}:${process.env.PATH}`,
        GH_TOKEN: token,
        RELEASE_TAG: tag,
        CALL_LOG: calls,
        EXIT_CODE: String(exitCode),
      },
      encoding: "utf8",
    });
    return { result, args: fs.existsSync(calls) ? JSON.parse(fs.readFileSync(calls, "utf8")) : null };
  }

  test(`${file}: synchronizes the matching release using explicit cross-repository credentials`, t => {
    assert.match(sync, /GH_TOKEN: \$\{\{ secrets\.GH_AW_ACTIONS_TOKEN \}\}/);
    assert.match(sync, prerelease ? /RELEASE_TAG: \$\{\{ needs\.config\.outputs\.release_tag \}\}/ : /RELEASE_TAG: \$\{\{ steps\.promote_release\.outputs\.release_tag \}\}/);
    const { result, args } = run(t);
    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(args, ["release", "edit", "v0.91.9", "--repo", "github/gh-aw-actions", `--prerelease=${prerelease}`, `--latest=${!prerelease}`]);
  });

  test(`${file}: missing credentials fail before contacting GitHub`, t => {
    const { result, args } = run(t, { token: "" });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /GH_AW_ACTIONS_TOKEN must have contents:write access/);
    assert.equal(args, null);
  });

  test(`${file}: missing releases and permission errors fail synchronization`, t => {
    const { result } = run(t, { exitCode: 1 });
    assert.equal(result.status, 1);
  });

  test(`${file}: passes the release tag as data, not shell code`, t => {
    const tag = 'v0.91.9; $(exit 99) "quoted"';
    const { result, args } = run(t, { tag });
    assert.equal(result.status, 0, result.stderr);
    assert.equal(args[2], tag);
  });
}

test("promotion passes the validated, trimmed tag to synchronization and remains retryable", async () => {
  const source = fs.readFileSync(path.join(workflows, "promote-release.yml"), "utf8");
  const promote = step(source, "Validate and promote release");
  assert.match(promote, /id: promote_release/);
  const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
  const script = new AsyncFunction("github", "context", "core", "process", block(promote, "script"));
  const updates = [];
  const outputs = [];
  await script(
    {
      rest: {
        repos: {
          getReleaseByTag: async parameters => {
            assert.equal(parameters.tag, "v0.91.9");
            return { data: { id: 123, prerelease: false, draft: false } };
          },
          updateRelease: async parameters => updates.push(parameters),
        },
      },
    },
    { repo: { owner: "github", repo: "gh-aw" } },
    { info: () => {}, setOutput: (...args) => outputs.push(args), setFailed: assert.fail },
    { env: { INPUT_VERSION: " v0.91.9 " } }
  );
  assert.deepEqual(updates, [{ owner: "github", repo: "gh-aw", release_id: 123, prerelease: false, make_latest: "true" }]);
  assert.deepEqual(outputs, [["release_tag", "v0.91.9"]]);
});
