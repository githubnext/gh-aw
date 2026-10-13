"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const { withClaimExecution } = require("./work_queue_claim_scope.cjs");
const { createClaimAdapterHandler, preparedAdapterPath, verifyClaimAdapterOutput, validateAdapter, createDeclaredAdapterVerifier } = require("./work_queue_claim_adapters.cjs");
const { verifyClaimDelivery } = require("./work_queue_delivery.cjs");
const { canonical } = require("./work_queue_codec.cjs");
const { prepareAdapterContext } = require("./work_queue_prepare_claim_adapter.cjs");

const BASE = "a".repeat(40);
const BASE_TREE = "b".repeat(40);
const hash = value => crypto.createHash("sha1").update(value).digest("hex");
const blobHash = value => hash(Buffer.concat([Buffer.from(`blob ${Buffer.byteLength(value)}\0`), Buffer.from(value)]));

function assigned(dispatch = "dispatch", count = 2) {
  /** @type {Array<{work_id: string, result_commit_id: string, descriptor: Record<string, any>}>} */
  const resultRefs = [];
  return {
    version: 3,
    dispatch_id: dispatch,
    request_id: "request",
    commit_id: "commit",
    policy_epoch: "epoch",
    pool: "default",
    worker_profile: "default",
    claims: ["h1", "h2"].slice(0, count).map(handle => ({
      handle,
      claim_id: `${dispatch}:claim:${handle}`,
      work_id: `${dispatch}:work:${handle}`,
      work: { effect_contract: { version: 1, outputs: [{ type: "code", min: 1, max: 1 }] } },
      result_refs: [...resultRefs],
    })),
  };
}

function configured(pullRequest = true) {
  return {
    mode: "prepared",
    "effect-type": "git_tree",
    "target-repo": "owner/repo",
    "field-map": { files: "files", ...(pullRequest ? { title: "title", body: "body" } : {}) },
    "git-tree": { "base-revision": BASE, "branch-prefix": "automation/code", ...(pullRequest ? { "pull-request": true, "base-branch": "main" } : {}) },
  };
}

function native() {
  /** @type {Map<string, string | Buffer>} */
  const blobs = new Map([[blobHash("base"), "base"]]);
  /** @type {Map<string, Record<string, any>>} */
  const commits = new Map([[BASE, { sha: BASE, tree: { sha: BASE_TREE }, parents: [] }]]);
  const refs = new Map();
  const trees = new Map([[BASE_TREE, [{ path: "README.md", type: "blob", mode: "100644", sha: blobHash("base") }]]]);
  const pulls = new Map();
  const writes = [];
  /** @type {{mutateReadback: ((tree: any) => void) | null, afterBlob: (() => void) | null}} */
  const state = { mutateReadback: null, afterBlob: null };
  const rest = {
    repos: {
      get: async ({ owner, repo }) => {
        assert.equal(`${owner}/${repo}`, "owner/repo");
        return { data: { id: 7, full_name: "owner/repo" } };
      },
    },
    git: {
      getCommit: async ({ commit_sha }) => ({ data: structuredClone(commits.get(commit_sha)) }),
      getTree: async ({ tree_sha }) => {
        const data = { sha: tree_sha, truncated: false, tree: structuredClone(trees.get(tree_sha)) };
        if (state.mutateReadback && tree_sha !== BASE_TREE) state.mutateReadback(data);
        return { data };
      },
      getRef: async ({ ref }) => {
        if (!refs.has(ref)) throw Object.assign(new Error("missing"), { status: 404 });
        return { data: { object: { sha: refs.get(ref) } } };
      },
      getBlob: async ({ file_sha }) => {
        const text = blobs.get(file_sha);
        assert.ok(text !== undefined);
        return { data: { sha: file_sha, encoding: "base64", content: Buffer.from(text).toString("base64") } };
      },
      createBlob: async ({ content, encoding }) => {
        writes.push("blob");
        assert.equal(encoding, "base64");
        const text = Buffer.from(content, encoding);
        const sha = blobHash(text);
        blobs.set(sha, text);
        if (state.afterBlob) state.afterBlob();
        return { data: { sha } };
      },
      createTree: async ({ base_tree, tree }) => {
        writes.push("tree");
        const base = trees.get(base_tree);
        assert.ok(base);
        const leaves = new Map(base.filter(entry => entry.type !== "tree").map(entry => [entry.path, entry]));
        for (const entry of tree) entry.sha === null ? leaves.delete(entry.path) : leaves.set(entry.path, entry);
        const entries = [...leaves.values()];
        const directories = new Set();
        for (const entry of entries) {
          const segments = entry.path.split("/");
          for (let count = 1; count < segments.length; count++) directories.add(segments.slice(0, count).join("/"));
        }
        for (const directory of directories) entries.push({ path: directory, mode: "040000", type: "tree", sha: hash(directory) });
        const sha = hash(canonical(entries));
        trees.set(sha, entries);
        return { data: { sha } };
      },
      createCommit: async ({ tree, parents, message }) => {
        writes.push("commit");
        assert.equal(parents.length, 1);
        assert.ok(commits.has(parents[0]));
        const sha = hash(canonical({ tree, parents, message }));
        commits.set(sha, { sha, tree: { sha: tree }, parents: parents.map(parent => ({ sha: parent })), message });
        return { data: { sha } };
      },
      createRef: async ({ ref, sha }) => {
        writes.push("ref");
        assert.match(ref, /^refs\/heads\/automation\/code\/claims\/[a-f0-9]{64}$/);
        assert.ok(!refs.has(ref.slice(5)));
        refs.set(ref.slice(5), sha);
        return { data: { object: { sha } } };
      },
    },
    pulls: {
      create: async ({ head, base, title, body }) => {
        writes.push("pull");
        const number = pulls.size + 1;
        const data = { id: number + 100, number, title, body, head: { sha: refs.get(`heads/${head}`), ref: head, repo: { full_name: "owner/repo" } }, base: { ref: base, repo: { full_name: "owner/repo" } } };
        pulls.set(number, data);
        return { data: structuredClone(data) };
      },
      get: async ({ pull_number }) => ({ data: structuredClone(pulls.get(pull_number)) }),
    },
  };
  return { github: { rest }, state, writes, refs, trees, commits, blobs, pulls };
}

async function prepare(root, assignment, handle, adapter, message, github) {
  const filename = preparedAdapterPath(root, handle, "code", assignment);
  fs.mkdirSync(path.dirname(filename), { recursive: true });
  fs.writeFileSync(filename, JSON.stringify({ version: 3, claim_handle: handle, type: "code", messages: [{ input: message, payload: Object.fromEntries(Object.values(adapter["field-map"]).map(field => [field, message[field]])) }] }));
  return createClaimAdapterHandler({ adapter, filename, github });
}

function registerTests({ describe, it }) {
  describe("immutable per-Claim prepared code delivery", () => {
    for (const finalPullRequest of [false, true]) {
      it(`continues from verified checkpoints and verifies the cumulative tree${finalPullRequest ? " and final pull request" : ""}`, async () => {
        const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code-continuation");
        const adapter = configured(false);
        adapter["git-tree"]["checkpoint-output"] = "code";
        const host = native();
        const authorize = async request => ({ claim_handle: request.claim_handle, authorized: true });
        let assignment = assigned("initial", 1);
        let parentCommit = BASE;
        let parentWork;
        let descriptor;
        try {
          for (let hop = 0; hop < 3; hop++) {
            if (hop > 0) {
              assignment = assigned(`continuation-${hop}`, 1);
              assignment.claims[0].work.work_queue_continuation = { root_work_id: "initial:work:h1", previous_work_id: parentWork, hop, handoff: "Resume checkpoint." };
              assignment.claims[0].result_refs = [{ work_id: parentWork, result_commit_id: `result-${hop}`, descriptor }];
            }
            const member = assignment.claims[0];
            const deliverPullRequest = finalPullRequest && hop === 2;
            const currentAdapter = deliverPullRequest ? configured(true) : adapter;
            currentAdapter["git-tree"]["checkpoint-output"] = "code";
            const message = {
              type: "code",
              claim_handle: member.handle,
              files: [{ path: `hop-${hop}.txt`, content: `checkpoint ${hop}` }],
              ...(deliverPullRequest ? { title: "Completed cumulative task", body: "Three verified checkpoints." } : {}),
            };
            await withClaimExecution({ assignment, claim_handle: member.handle, authorize, effects: [] }, async () => {
              const handler = await prepare(root, assignment, member.handle, currentAdapter, message, host.github);
              const result = await handler(message);
              const commit = host.commits.get(result.commit);
              assert.ok(commit);
              assert.equal(commit.parents[0].sha, parentCommit);
              const delivery = await verifyClaimDelivery({
                assignment,
                claim_handle: member.handle,
                messages: [message],
                results: [{ messageIndex: 0, success: true, result }],
                github: host.github,
                authorize,
                verifyOutput: input => verifyClaimAdapterOutput({ ...input, adapter: currentAdapter }),
              });
              assert.equal(delivery.verification, "verified", JSON.stringify(delivery));
              descriptor = delivery.descriptor;
              const tree = host.trees.get(commit.tree.sha);
              assert.ok(tree);
              for (let previous = 0; previous <= hop; previous++) {
                assert.ok(tree.some(entry => entry.path === `hop-${previous}.txt` && entry.sha === blobHash(`checkpoint ${previous}`)));
              }
              parentCommit = result.commit;
              if (hop < 2) host.refs.delete(`heads/${result.branch}`);
            });
            parentWork = member.work_id;
          }
          assert.equal(host.refs.size, 1);
          assert.equal(host.pulls.size, finalPullRequest ? 1 : 0);
          if (finalPullRequest) {
            const pull = [...host.pulls.values()][0];
            assert.equal(pull.head.sha, parentCommit);
            assert.equal(pull.base.ref, "main");
            assert.equal(descriptor.outputs[0].resource.kind, "pull_request");
          }
        } finally {
          fs.rmSync(root, { recursive: true, force: true });
        }
      });
    }

    it("rejects missing, ambiguous, foreign and tampered predecessor checkpoint evidence before native mutation", async () => {
      const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code-checkpoint-invalid");
      const adapter = configured(false);
      adapter["git-tree"]["checkpoint-output"] = "code";
      const proof = {
        type: "code",
        resource: { kind: "git_commit", repository: "owner/repo", id: BASE, ref: `heads/automation/code/claims/${"c".repeat(64)}` },
        evidence: { source: "independent_git_tree_readback", commit: BASE, tree: BASE_TREE, parent: "d".repeat(40) },
      };
      const mutations = [
        member => {
          member.result_refs = [];
        },
        member => {
          member.result_refs[0].work_id = "another-work";
        },
        member => {
          member.result_refs[0].descriptor.outputs = [];
        },
        member => {
          member.result_refs[0].descriptor.outputs = [null];
        },
        member => {
          member.result_refs[0].descriptor.outputs.push(structuredClone(proof));
        },
        member => {
          member.result_refs[0].descriptor.version = 2;
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].resource.repository = "foreign/repo";
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].resource.kind = "pull_request";
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].resource.ref = "heads/main";
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].evidence.source = "agent_report";
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].evidence.commit = "e".repeat(40);
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].evidence.tree = "e".repeat(40);
        },
        member => {
          member.result_refs[0].descriptor.outputs[0].evidence.parent = "main";
        },
        () => {},
      ];
      try {
        for (const mutate of mutations) {
          const assignment = assigned("invalid", 1);
          const member = assignment.claims[0];
          member.work.work_queue_continuation = { root_work_id: "parent", previous_work_id: "parent", hop: 1, handoff: "Resume." };
          member.result_refs = [{ work_id: "parent", result_commit_id: "verified-result", descriptor: { version: 1, outputs: [structuredClone(proof)] } }];
          mutate(member);
          const host = native();
          const message = { type: "code", claim_handle: member.handle, files: [{ path: "child.txt", content: "must not write" }] };
          const authorize = async request => ({ claim_handle: request.claim_handle, authorized: true });
          await withClaimExecution({ assignment, claim_handle: member.handle, authorize, effects: [] }, async () => {
            const handler = await prepare(root, assignment, member.handle, adapter, message, host.github);
            await assert.rejects(handler(message), /checkpoint|verified Result|base commit/);
          });
          assert.deepEqual(host.writes, []);
        }
        for (const value of ["", "Code", "../code", "a".repeat(129), 1, null]) {
          const invalid = structuredClone(adapter);
          invalid["git-tree"]["checkpoint-output"] = value;
          assert.throws(() => validateAdapter(invalid), /checkpoint-output/);
        }
      } finally {
        fs.rmSync(root, { recursive: true, force: true });
      }
    });

    it("retains fixed-base delivery when checkpoint reuse is not configured", async () => {
      const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code-fixed-base");
      const assignment = assigned("fixed", 1);
      const member = assignment.claims[0];
      member.work.work_queue_continuation = { root_work_id: "parent", previous_work_id: "parent", hop: 1, handoff: "Resume." };
      const adapter = configured(false);
      const host = native();
      const message = { type: "code", claim_handle: member.handle, files: [{ path: "fixed.txt", content: "fixed base" }] };
      const authorize = async request => ({ claim_handle: request.claim_handle, authorized: true });
      try {
        await withClaimExecution({ assignment, claim_handle: member.handle, authorize, effects: [] }, async () => {
          const handler = await prepare(root, assignment, member.handle, adapter, message, host.github);
          const result = await handler(message);
          const commit = host.commits.get(result.commit);
          assert.ok(commit);
          assert.equal(commit.parents[0].sha, BASE);
          assert.equal((await verifyClaimAdapterOutput({ claim: member, result, github: host.github, adapter })).verified, true);
        });
      } finally {
        fs.rmSync(root, { recursive: true, force: true });
      }
    });

    for (const count of [1, 2]) {
      it(`prepares and independently settles every member of an original ${count}-Claim assignment`, async () => {
        const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code-matrix");
        const assignment = assigned("dispatch", count);
        const adapter = configured(false);
        const host = native();
        const inputs = assignment.claims.map(member => ({ type: "code", ...(count === 1 ? {} : { claim_handle: member.handle }), files: [{ path: `${member.handle}.txt`, content: member.work_id }] }));
        const authorize = async request => {
          assert.equal(request.assignment.claims.length, count);
          assert.equal(request.assignment.worker_profile, assignment.worker_profile);
          return { claim_handle: request.claim_handle, authorized: true };
        };
        const results = [];
        try {
          for (const [index, member] of assignment.claims.entries()) {
            const prepared = await prepareAdapterContext({ assignment, adapter, index, type: "code", messages: inputs, authorize, artifactRoot: root });
            assert.deepEqual(prepared.messages, [{ ...inputs[index], claim_handle: member.handle }]);
            fs.writeFileSync(prepared.outputFile, JSON.stringify({ version: 3, claim_handle: member.handle, type: "code", messages: [{ input: prepared.messages[0], payload: { files: inputs[index].files } }] }));
            const effects = [];
            await withClaimExecution({ assignment, claim_handle: member.handle, authorize, effects }, async () => {
              const handler = await createClaimAdapterHandler({ adapter, filename: prepared.outputFile, github: host.github });
              if (count > 1) await assert.rejects(handler({ type: "code", files: inputs[index].files }), /claim_handle is required/);
              const result = await handler(inputs[index]);
              results.push(result);
              const delivery = await verifyClaimDelivery({
                assignment,
                claim_handle: member.handle,
                messages: [inputs[index]],
                results: [{ messageIndex: 0, success: true, result }],
                effects,
                github: host.github,
                authorize,
                verifyOutput: input => verifyClaimAdapterOutput({ ...input, adapter }),
              });
              assert.equal(delivery.verification, "verified");
              if (index > 0) assert.equal((await verifyClaimAdapterOutput({ claim: member, result: results[0], github: host.github, adapter })).verified, false);
            });
          }
          assert.equal(host.refs.size, count);
          assert.equal(new Set(results.map(result => result.branch)).size, count);
          assert.equal(host.pulls.size, 0);
        } finally {
          fs.rmSync(root, { recursive: true, force: true });
        }
      });
    }

    it("verifies exact complete tree, bytes, parent, branch and PR fields without a shared writer checkout", async () => {
      const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code");
      const assignment = assigned();
      const adapter = configured();
      const host = native();
      const trustedContext = { repo: { owner: "owner", repo: "repo" } };
      const authorize = async request => {
        if (request.message.type === "work_queue_resource_verification") {
          assert.equal(request.context, trustedContext);
          assert.equal(request.github, host.github);
        }
        return { claim_handle: request.claim_handle, authorized: true };
      };
      const effects = [];
      const message = {
        type: "code",
        claim_handle: "h1",
        files: [
          { path: "src/code.js", content: "new code\n", mode: "100755" },
          { path: "binary.bin", content: Buffer.from([0, 255, 170]).toString("base64"), encoding: "base64" },
          { path: "README.md", delete: true },
        ],
        title: "Claim code",
        body: "Exact delivered code",
      };
      assignment.claims[0].work.effect_contract.outputs[0].verification = { verifier_id: "code", expected: { files: message.files, title: message.title, body: message.body } };
      try {
        await withClaimExecution({ assignment, claim_handle: "h1", authorize, effects }, async () => {
          const handler = await prepare(root, assignment, "h1", adapter, message, host.github);
          const result = await handler(message);
          assert.deepEqual(host.writes, ["blob", "blob", "tree", "commit", "ref", "pull"]);
          const options = {
            assignment,
            claim_handle: "h1",
            messages: [message],
            results: [{ messageIndex: 0, success: true, result }],
            effects,
            github: host.github,
            authorize,
            context: trustedContext,
            verifyOutput: input => verifyClaimAdapterOutput({ ...input, adapter }),
            verifyDeclaredOutput: createDeclaredAdapterVerifier({ code: adapter }),
          };
          assert.equal((await verifyClaimDelivery(options)).verification, "verified");
          assert.equal((await verifyClaimAdapterOutput({ claim: assignment.claims[0], result: structuredClone(result), github: host.github, adapter })).verified, false);
          host.state.mutateReadback = tree => tree.tree.push({ path: "undeclared.txt", mode: "100644", type: "blob", sha: blobHash("unexpected") });
          assert.equal((await verifyClaimDelivery(options)).verification, "unknown");
          host.state.mutateReadback = null;
          host.blobs.set(blobHash("new code\n"), "incorrect content");
          assert.equal((await verifyClaimDelivery(options)).verification, "unknown");
          host.blobs.set(blobHash("new code\n"), "new code\n");
          host.pulls.get(result.number).head.repo.full_name = "foreign/repo";
          assert.equal((await verifyClaimDelivery(options)).verification, "unknown");
          await assert.rejects(handler(message), /only one immutable delivery/);
        });
      } finally {
        fs.rmSync(root, { recursive: true, force: true });
      }
    });

    it("isolates failed and valid sibling code effects and persistent branch-only deliveries", async () => {
      const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code");
      const assignment = assigned();
      const host = native();
      let cancelled = false;
      const authorize = async request => ({ claim_handle: request.claim_handle, authorized: request.claim_handle !== "h1" || !cancelled });
      const adapter = configured(false);
      try {
        for (const handle of ["h1", "h2"]) {
          const message = { type: "code", claim_handle: handle, files: [{ path: "state.json", content: `{"claim":"${handle}"}` }] };
          const effects = [];
          await withClaimExecution({ assignment, claim_handle: handle, authorize, effects }, async () => {
            const handler = await prepare(root, assignment, handle, adapter, message, host.github);
            host.state.afterBlob = () => {
              if (handle === "h1") cancelled = true;
            };
            if (handle === "h1") {
              await assert.rejects(handler(message), /same-Claim/);
              assert.deepEqual(host.writes, ["blob"]);
            } else {
              const result = await handler(message);
              const delivery = await verifyClaimDelivery({
                assignment,
                claim_handle: handle,
                messages: [message],
                results: [{ messageIndex: 0, success: true, result }],
                effects,
                github: host.github,
                authorize,
                verifyOutput: input => verifyClaimAdapterOutput({ ...input, adapter }),
              });
              assert.equal(delivery.verification, "verified");
              assert.equal(delivery.descriptor.outputs[0].resource.kind, "git_commit");
            }
          });
        }
        assert.equal(host.refs.size, 1);
        assert.equal(host.pulls.size, 0);
      } finally {
        fs.rmSync(root, { recursive: true, force: true });
      }
    });

    it("rejects malformed, conflicting and unauthorized file paths before any native mutation", async () => {
      const root = require("./work_queue_effect_test_helpers.cjs").temporaryDirectory("claim-code");
      const assignment = assigned();
      const host = native();
      const adapter = configured(false);
      const authorize = async request => ({ claim_handle: request.claim_handle, authorized: request.resource?.path !== "forbidden.txt" });
      try {
        await withClaimExecution({ assignment, claim_handle: "h1", authorize }, async () => {
          for (const files of [
            [{ path: ".git/config", content: "bad" }],
            [{ path: "README.md/child", content: "conflict" }],
            [{ path: "new", delete: true }],
            [{ path: "forbidden.txt", content: "bad" }],
            [{ path: "x", content: "x", executable: true }],
          ]) {
            const message = { type: "code", claim_handle: "h1", files };
            const handler = await prepare(root, assignment, "h1", adapter, message, host.github);
            await assert.rejects(handler(message));
          }
        });
        assert.deepEqual(host.writes, []);
        const invalid = configured(false);
        invalid["git-tree"]["base-revision"] = "main";
        assert.throws(() => validateAdapter(invalid), /immutable base revision/);
      } finally {
        fs.rmSync(root, { recursive: true, force: true });
      }
    });
  });
}

module.exports = { registerTests };
if (require.main === module) registerTests(require("node:test"));
