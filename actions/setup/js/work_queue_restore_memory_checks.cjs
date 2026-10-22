"use strict";

const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const { restoreESLintMemory } = require("./work_queue_restore_memory.cjs");

const BASE = "46b68a61c366a01d86dc319b8e689d09a48bad04";
const SNAPSHOT = "b".repeat(40);
const SECOND_SNAPSHOT = "f".repeat(40);
const REF = `refs/heads/memory/eslint-refiner-runs/claims/${"c".repeat(64)}`;
const SECOND_REF = `refs/heads/memory/eslint-refiner-runs/claims/${"d".repeat(64)}`;

function fixture(count = 1) {
  const blobs = new Map();
  const blob = (name, content) => {
    const bytes = Buffer.from(content);
    const sha = createHash("sha1").update(`blob ${bytes.length}\0`).update(bytes).digest("hex");
    blobs.set(sha, { sha, encoding: "base64", size: bytes.length, content: bytes.toString("base64") });
    return { path: name, sha, type: "blob", mode: "100644", size: bytes.length };
  };
  const legacy = [blob("strategy.json", '{"strategy":"keep existing memory"}\n'), blob("history.jsonl", '{"run":1}\n{"run":2}\n')];
  const memory = blob("eslint-refiner.json", '{"work_id":"eslint-refiner:known","findings":["exact snapshot"]}\n');
  const secondMemory = blob("eslint-refiner.json", '{"work_id":"eslint-refiner:sibling","findings":["independent snapshot"]}\n');
  const calls = [];
  const state = {
    missingLegacy: false,
    truncated: false,
    repositoryId: 1036865607,
    refs: [
      { ref: REF, object: { sha: SNAPSHOT, type: "commit" } },
      { ref: SECOND_REF, object: { sha: SECOND_SNAPSHOT, type: "commit" } },
    ].slice(0, count),
    tamper: false,
  };
  const read = name => {
    calls.push(name);
    assert.ok(!/create|update|delete|dispatch|push/i.test(name));
  };
  const github = {
    rest: {
      repos: {
        get: async () => {
          read("repos.get");
          return { data: { id: state.repositoryId, full_name: "github/gh-aw" } };
        },
      },
      git: {
        getRef: async args => {
          read("git.getRef");
          assert.equal(args.ref, "heads/memory/eslint-refiner");
          if (state.missingLegacy) throw Object.assign(new Error("missing"), { status: 404 });
          return { data: { ref: `refs/${args.ref}`, object: { sha: BASE, type: "commit" } } };
        },
        getCommit: async args => {
          read("git.getCommit");
          assert.ok([BASE, SNAPSHOT, SECOND_SNAPSHOT].includes(args.commit_sha));
          return { data: { sha: args.commit_sha, tree: { sha: args.commit_sha === BASE ? "d".repeat(40) : args.commit_sha === SNAPSHOT ? "e".repeat(40) : "9".repeat(40) } } };
        },
        getTree: async args => {
          read("git.getTree");
          assert.equal(args.recursive, "1");
          return { data: { sha: args.tree_sha, truncated: state.truncated, tree: args.tree_sha === "d".repeat(40) ? legacy : [...legacy, args.tree_sha === "e".repeat(40) ? memory : secondMemory] } };
        },
        getBlob: async args => {
          read("git.getBlob");
          const data = structuredClone(blobs.get(args.file_sha));
          if (state.tamper) data.content = Buffer.from("x".repeat(data.size)).toString("base64");
          return { data };
        },
      },
    },
    paginate: async (route, args) => {
      read("paginate:GET");
      assert.equal(route, "GET /repos/{owner}/{repo}/git/matching-refs/{ref}");
      assert.equal(args.ref, "heads/memory/eslint-refiner-runs/claims/");
      assert.equal(args.per_page, 100);
      return structuredClone(state.refs);
    },
  };
  return { github, state, calls, blobs, legacy, blob };
}

/** @param {{restore: () => Promise<unknown>, writeError?: Error}} options */
function preparationFixture({ restore, writeError }) {
  const source = fs.readFileSync(path.join(__dirname, "../../../.github/workflows/eslint-refiner.md"), "utf8");
  const script = source.match(/      script: \|\n([\s\S]*?)(?=\ntools:)/)?.[1];
  assert.ok(script, "Refiner memory preparation script must exist");
  const writes = [];
  const directories = [];
  const github = {};
  const mocks = {
    "node:fs": {
      mkdirSync: (directory, options) => directories.push([directory, options.recursive]),
      writeFileSync: (filename, contents, options) => {
        if (writeError) throw writeError;
        writes.push([filename, contents, options.mode]);
      },
    },
    "node:path": path,
    "/runner-temp/gh-aw/actions/work_queue_restore_memory.cjs": {
      restoreESLintMemory: async client => {
        assert.equal(client, github);
        return restore();
      },
    },
  };
  const run = () =>
    vm.runInNewContext(`(async () => {${script}\n})()`, {
      require: name => {
        assert.ok(Object.hasOwn(mocks, name), "Preparation requires only reviewed dependencies");
        return mocks[name];
      },
      github,
      process: { env: { RUNNER_TEMP: "/runner-temp" } },
    });
  return { run, source, writes, directories };
}

function registerTests({ describe, it }) {
  describe("read-only restore of persistent Claim and legacy ESLint memory", () => {
    it("runs the Refiner preparation step and writes history at the exact agent-visible prompt path", async () => {
      const history = { legacy: { files: [] }, snapshots: [] };
      const { run, source, writes, directories } = preparationFixture({ restore: async () => history });
      await run();
      const filename = "/tmp/gh-aw/agent/eslint-refiner-memory/history.json";
      assert.deepEqual(directories, [["/tmp/gh-aw/agent/eslint-refiner-memory", true]]);
      assert.deepEqual(writes, [[filename, JSON.stringify(history) + "\n", 0o400]]);
      assert.ok(source.includes(`cat ${filename}`));
      assert.ok(source.includes(`Read \`${filename}\``));
      assert.ok(source.includes("`cat` command exits non-zero or its output is not valid JSON"));
    });

    it("surfaces preparation write failures rather than substituting empty history", async () => {
      const writeError = Object.assign(new Error("EACCES: permission denied writing history.json"), { code: "EACCES" });
      const { run, writes } = preparationFixture({ restore: async () => ({ legacy: { files: [] }, snapshots: [] }), writeError });
      await assert.rejects(run(), error => error === writeError);
      assert.deepEqual(writes, []);
    });

    it("surfaces native history read failures before preparing a history file", async () => {
      const f = fixture();
      const readError = Object.assign(new Error("Memory history blob read forbidden"), { status: 403 });
      f.github.rest.git.getBlob = async () => {
        throw readError;
      };
      const { run, writes, directories } = preparationFixture({ restore: () => restoreESLintMemory(f.github) });
      await assert.rejects(run(), error => error === readError);
      assert.deepEqual(writes, []);
      assert.deepEqual(directories, []);
    });

    it("prepares valid empty history when the legacy branch and prior snapshots are absent", async () => {
      const f = fixture(0);
      f.state.missingLegacy = true;
      f.legacy.length = 0;
      const { run, writes } = preparationFixture({ restore: () => restoreESLintMemory(f.github) });
      await run();
      assert.equal(writes.length, 1);
      const history = JSON.parse(writes[0][1]);
      assert.equal(history.legacy.commit, BASE);
      assert.deepEqual(history.legacy.files, []);
      assert.deepEqual(history.snapshots, []);
    });

    it("restores independently addressed multi-Claim history without narrowing it or granting current Claim authority", async () => {
      const f = fixture(2);
      f.state.refs.reverse();
      const result = await restoreESLintMemory(f.github);
      assert.deepEqual(
        result.snapshots.map(snapshot => [snapshot.ref, snapshot.commit, snapshot.memory.work_id]),
        [
          [REF, SNAPSHOT, "eslint-refiner:known"],
          [SECOND_REF, SECOND_SNAPSHOT, "eslint-refiner:sibling"],
        ]
      );
      assert.notEqual(result.snapshots[0].blob, result.snapshots[1].blob);
      assert.equal(f.calls.filter(name => name === "git.getBlob").length, 4);
      assert.equal("assignment" in result, false);
      assert.equal("claims" in result, false);
      assert.equal("verified" in result, false);
    });

    it("rejects native malformed JSON, duplicate keys and traversal paths without returning partial history", async () => {
      for (const content of ['{"partial":', '{"field":1,"field":2}', '{"field":"\\ud800"}']) {
        const f = fixture(2);
        f.legacy[0] = f.blob("strategy.json", content);
        await assert.rejects(restoreESLintMemory(f.github));
        assert.equal(
          f.calls.some(name => /create|update|delete|dispatch|push/i.test(name)),
          false
        );
      }
      const foreign = fixture();
      foreign.legacy[0].path = "../foreign.json";
      await assert.rejects(restoreESLintMemory(foreign.github), /malformed/);
      const duplicated = fixture();
      duplicated.legacy.push(duplicated.legacy[0]);
      await assert.rejects(restoreESLintMemory(duplicated.github), /duplicate/);
    });

    it("restores legacy JSON/JSONL and new snapshots from exact native refs, commits, trees and blob bytes", async () => {
      const f = fixture();
      const result = await restoreESLintMemory(f.github);
      assert.equal(result.legacy.commit, BASE);
      assert.deepEqual(
        result.legacy.files.map(file => file.path),
        ["strategy.json", "history.jsonl"]
      );
      assert.equal(result.snapshots.length, 1);
      assert.equal(result.snapshots[0].commit, SNAPSHOT);
      assert.equal(result.snapshots[0].ref, REF);
      assert.deepEqual({ ...result.snapshots[0].memory }, { work_id: "eslint-refiner:known", findings: ["exact snapshot"] });
      assert.equal("verified" in result, false);
      assert.equal("authority_resource" in result, false);
      assert.equal(f.calls.filter(name => name === "git.getBlob").length, 3);
    });

    it("retains strict JSONL validation rather than skipping malformed restored memory records", async () => {
      for (const invalid of ['{"partial":', '{"field":1,"field":2}', '{"field":1e0}', '{"field":"\\ud800"}', '\ufeff{"field":1}', '\u00a0{"field":1}']) {
        const f = fixture();
        f.legacy[1] = f.blob("history.jsonl", `{"run":1}\n${invalid}\n{"run":2}\n`);
        await assert.rejects(restoreESLintMemory(f.github));
      }
      const f = fixture();
      const content = '\n{"run":1}\r\n \t\n{"run":2}';
      f.legacy[1] = f.blob("history.jsonl", content);
      const restored = await restoreESLintMemory(f.github);
      assert.equal(restored.legacy.files.find(file => file.path === "history.jsonl").content, content);
    });

    it("retains the pinned original history when the legacy branch is absent and supports first-run snapshot absence", async () => {
      const f = fixture();
      f.state.missingLegacy = true;
      f.state.refs = [];
      const result = await restoreESLintMemory(f.github);
      assert.equal(result.legacy.commit, BASE);
      assert.equal(result.legacy.files.length, 2);
      assert.deepEqual(result.snapshots, []);
    });

    it("rejects foreign identities, incomplete trees, duplicate or malformed refs and changed blob bytes", async () => {
      for (const mutate of [
        f => {
          f.state.repositoryId = 7;
        },
        f => {
          f.state.truncated = true;
        },
        f => {
          f.state.refs.push(f.state.refs[0]);
        },
        f => {
          f.state.refs[0].ref = "refs/heads/foreign";
        },
        f => {
          f.state.refs[0].object.sha = "main";
        },
        f => {
          f.state.tamper = true;
        },
      ]) {
        const f = fixture();
        mutate(f);
        await assert.rejects(restoreESLintMemory(f.github));
      }
    });

    it("fails closed rather than silently discarding older history or oversized files", async () => {
      const f = fixture();
      f.state.refs = Array.from({ length: 129 }, () => f.state.refs[0]);
      await assert.rejects(restoreESLintMemory(f.github), /at most 128/);
      const oversized = fixture();
      oversized.legacy[0].size = 1048577;
      await assert.rejects(restoreESLintMemory(oversized.github), /byte limit/);
    });
  });
}

module.exports = { registerTests };
if (require.main === module) registerTests(require("node:test"));
