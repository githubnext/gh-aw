// @ts-check
"use strict";

const crypto = require("crypto");

/**
 * In-memory fake of the GitHub git data API used by the continue_workflow ledger.
 * Implements fast-forward-only `updateRef` (force: false) so compare-and-swap conflicts
 * behave like the real API.
 * @param {{onBeforeUpdateRef?: (store: any) => Promise<void> | void}} [options]
 */
function createLedgerFake(options = {}) {
  /** @type {Map<string, string>} */
  const blobs = new Map();
  /** @type {Map<string, Map<string, string>>} */
  const trees = new Map();
  /** @type {Map<string, {tree: string, parents: string[], message: string}>} */
  const commits = new Map();
  /** @type {Map<string, string>} */
  const refs = new Map();
  const calls = { updateRef: 0, createRef: 0, createWorkflowDispatch: /** @type {any[]} */ ([]) };
  let dispatchError = /** @type {any} */ (null);

  const sha = () => crypto.randomBytes(20).toString("hex");
  const notFound = () => Object.assign(new Error("Not Found"), { status: 404 });
  const unprocessable = (/** @type {string} */ message) => Object.assign(new Error(message), { status: 422 });

  /** @param {string} from @param {string} to */
  function isAncestor(from, to) {
    const queue = [to];
    const seen = new Set();
    while (queue.length > 0) {
      const current = /** @type {string} */ (queue.shift());
      if (current === from) return true;
      if (seen.has(current)) continue;
      seen.add(current);
      queue.push(...(commits.get(current)?.parents || []));
    }
    return false;
  }

  const store = {
    blobs,
    trees,
    commits,
    refs,
    calls,
    /** @param {any} error */
    failNextDispatch(error) {
      dispatchError = error;
    },
    /** Reads a chain document from the current ledger head. */
    readFile(/** @type {string} */ branch, /** @type {string} */ filePath) {
      const head = refs.get(branch);
      if (!head) return null;
      const tree = trees.get(/** @type {any} */ (commits.get(head)).tree);
      const blobSha = tree?.get(filePath);
      return blobSha ? JSON.parse(/** @type {string} */ (blobs.get(blobSha))) : null;
    },
  };

  const github = {
    rest: {
      git: {
        getRef: async (/** @type {any} */ { ref }) => {
          const name = ref.replace(/^heads\//, "");
          if (!refs.has(name)) throw notFound();
          return { data: { object: { sha: refs.get(name) } } };
        },
        getCommit: async (/** @type {any} */ { commit_sha }) => {
          const commit = commits.get(commit_sha);
          if (!commit) throw notFound();
          return { data: { sha: commit_sha, tree: { sha: commit.tree } } };
        },
        getTree: async (/** @type {any} */ { tree_sha }) => {
          const tree = trees.get(tree_sha);
          if (!tree) throw notFound();
          return { data: { tree: [...tree.keys()].map(path => ({ path, type: "blob" })) } };
        },
        createBlob: async (/** @type {any} */ { content }) => {
          const blobSha = sha();
          blobs.set(blobSha, content);
          return { data: { sha: blobSha } };
        },
        createTree: async (/** @type {any} */ { base_tree, tree }) => {
          const entries = new Map(base_tree ? trees.get(base_tree) : []);
          for (const entry of tree) entries.set(entry.path, entry.sha);
          const treeSha = sha();
          trees.set(treeSha, entries);
          return { data: { sha: treeSha } };
        },
        createCommit: async (/** @type {any} */ { tree, parents, message }) => {
          const commitSha = sha();
          commits.set(commitSha, { tree, parents, message });
          return { data: { sha: commitSha } };
        },
        updateRef: async (/** @type {any} */ { ref, sha: target, force }) => {
          calls.updateRef++;
          if (options.onBeforeUpdateRef) await options.onBeforeUpdateRef(store);
          const name = ref.replace(/^heads\//, "");
          const current = refs.get(name);
          if (!current) throw unprocessable("Reference does not exist");
          if (!force && !isAncestor(current, target)) throw unprocessable("Update is not a fast forward");
          refs.set(name, target);
          return { data: {} };
        },
        createRef: async (/** @type {any} */ { ref, sha: target }) => {
          calls.createRef++;
          const name = ref.replace(/^refs\/heads\//, "");
          if (refs.has(name)) throw unprocessable("Reference already exists");
          refs.set(name, target);
          return { data: {} };
        },
      },
      repos: {
        getContent: async (/** @type {any} */ { path: filePath, ref }) => {
          const commit = commits.get(ref);
          const blobSha = commit ? trees.get(commit.tree)?.get(filePath) : undefined;
          if (!blobSha) throw notFound();
          return { data: { content: Buffer.from(/** @type {string} */ (blobs.get(blobSha))).toString("base64"), encoding: "base64" } };
        },
        get: async () => ({ data: { default_branch: "main" } }),
      },
      actions: {
        createWorkflowDispatch: async (/** @type {any} */ params) => {
          if (dispatchError) {
            const error = dispatchError;
            dispatchError = null;
            throw error;
          }
          calls.createWorkflowDispatch.push(params);
          return { data: {} };
        },
      },
    },
  };

  return { github, store };
}

module.exports = { createLedgerFake };
