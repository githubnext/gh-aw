// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("path")} */
const path = require("path");
const { ERR_API } = require("./error_codes.cjs");

/**
 * Unescape a C-quoted path returned by `git diff-tree --raw`.
 *
 * git wraps paths that contain special characters (spaces, non-ASCII bytes,
 * control characters, etc.) in double-quotes and encodes each "unusual" byte
 * as a C-style escape sequence.  This function strips the surrounding quotes
 * and decodes the escape sequences back to the original byte sequence, then
 * interprets the result as UTF-8.
 *
 * Supported escape sequences: `\\`, `\"`, `\a`, `\b`, `\f`, `\n`, `\r`,
 * `\t`, `\v`, and octal `\NNN` (1–3 octal digits).
 *
 * @param {string} s - Raw path token from git output (may or may not be quoted)
 * @returns {string} Unescaped path
 */
function unquoteCPath(s) {
  if (!s.startsWith('"')) return s;
  // Strip surrounding double-quotes
  const inner = s.slice(1, s.endsWith('"') ? s.length - 1 : s.length);
  const bytes = [];
  let i = 0;
  while (i < inner.length) {
    if (inner[i] === "\\") {
      i++;
      if (i < inner.length && inner[i] >= "0" && inner[i] <= "7") {
        // Octal sequence – collect up to 3 octal digits
        let oct = "";
        while (i < inner.length && inner[i] >= "0" && inner[i] <= "7" && oct.length < 3) {
          oct += inner[i++];
        }
        bytes.push(parseInt(oct, 8));
      } else {
        const esc = inner[i++];
        switch (esc) {
          case "\\":
            bytes.push(0x5c);
            break;
          case '"':
            bytes.push(0x22);
            break;
          case "a":
            bytes.push(0x07);
            break;
          case "b":
            bytes.push(0x08);
            break;
          case "f":
            bytes.push(0x0c);
            break;
          case "n":
            bytes.push(0x0a);
            break;
          case "r":
            bytes.push(0x0d);
            break;
          case "t":
            bytes.push(0x09);
            break;
          case "v":
            bytes.push(0x0b);
            break;
          default:
            // Unknown escape: preserve backslash and the character as-is
            bytes.push(0x5c, esc.charCodeAt(0));
        }
      }
    } else {
      bytes.push(inner.charCodeAt(i++));
    }
  }
  return Buffer.from(bytes).toString("utf8");
}

/**
 * @fileoverview Signed Commit Push Helper
 *
 * Pushes local git commits to a remote branch using the GitHub GraphQL
 * `createCommitOnBranch` mutation, so commits are cryptographically signed
 * (verified) by GitHub.  Falls back to a plain `git push` when the GraphQL
 * approach is unavailable (e.g. GitHub Enterprise Server instances that do
 * not support the mutation, or when branch-protection policies reject it).
 *
 * Both `create_pull_request.cjs` and `push_to_pull_request_branch.cjs` use
 * this helper so the signed-commit logic lives in exactly one place.
 */

/**
 * Pushes local commits to a remote branch using the GitHub GraphQL
 * `createCommitOnBranch` mutation so commits are cryptographically signed.
 * Falls back to `git push` if the GraphQL approach fails (e.g. on GHES).
 *
 * @param {object} opts
 * @param {any} opts.githubClient - Authenticated Octokit client with `.graphql()` and `.rest.git.createRef()`
 * @param {string} opts.owner - Repository owner
 * @param {string} opts.repo - Repository name
 * @param {string} opts.branch - Target branch name
 * @param {string} opts.baseRef - Git ref of the remote head before commits were applied (used for rev-list)
 * @param {string} opts.cwd - Working directory of the local git checkout
 * @param {object} [opts.gitAuthEnv] - Environment variables for git push fallback auth
 * @returns {Promise<void>}
 */
async function pushSignedCommits({ githubClient, owner, repo, branch, baseRef, cwd, gitAuthEnv }) {
  // Collect the commits introduced (oldest-first)
  const { stdout: revListOut } = await exec.getExecOutput("git", ["rev-list", "--reverse", `${baseRef}..HEAD`], { cwd });
  const shas = revListOut.trim().split("\n").filter(Boolean);

  if (shas.length === 0) {
    core.info("pushSignedCommits: no new commits to push via GraphQL");
    return;
  }

  core.info(`pushSignedCommits: replaying ${shas.length} commit(s) via GraphQL createCommitOnBranch (branch: ${branch}, repo: ${owner}/${repo})`);

  try {
    // Pre-scan ALL commits: collect file changes and check for unsupported file modes
    // BEFORE starting any GraphQL mutations. If a symlink is found mid-loop after some
    // commits have already been signed, the remote branch diverges and the git push
    // fallback would be rejected as non-fast-forward.
    //
    // The GitHub GraphQL createCommitOnBranch mutation only supports regular file mode 100644:
    //   - Symlinks (120000) would be silently converted to regular files containing the link target path
    //   - Executable bits (100755) are silently dropped
    /** @type {Map<string, Array<{path: string, contents: string}>>} */
    const additionsMap = new Map();
    /** @type {Map<string, Array<{path: string}>>} */
    const deletionsMap = new Map();

    for (const sha of shas) {
      /** @type {Array<{path: string, contents: string}>} */
      const additions = [];
      /** @type {Array<{path: string}>} */
      const deletions = [];

      // Use git diff-tree --raw to obtain file mode information per changed file.
      // Format: :<srcMode> <dstMode> <srcHash> <dstHash> <status>[score]\t<path>[<\t><newPath>]
      // Fields: [0]=srcMode, [1]=dstMode, [2]=srcHash, [3]=dstHash, [4]=status
      const { stdout: rawDiffOut } = await exec.getExecOutput("git", ["diff-tree", "-r", "--raw", sha], { cwd });

      for (const line of rawDiffOut.trim().split("\n").filter(Boolean)) {
        // Raw format lines start with ':'; skip the commit SHA header line and any other non-raw lines
        if (!line.startsWith(":")) continue;

        const tabIdx = line.indexOf("\t");
        if (tabIdx === -1) continue;

        const modeFields = line.slice(1, tabIdx).split(" "); // strip leading ':'
        if (modeFields.length < 5) {
          core.warning(`pushSignedCommits: unexpected diff-tree output format, skipping line: ${line}`);
          continue;
        }
        const dstMode = modeFields[1]; // destination file mode (e.g. 100644, 100755, 120000)
        const status = modeFields[4]; // A=Added, M=Modified, D=Deleted, R=Renamed, C=Copied

        const paths = line.slice(tabIdx + 1).split("\t");
        const filePath = unquoteCPath(paths[0]);

        if (status === "D") {
          deletions.push({ path: filePath });
        } else if (status && status.startsWith("R")) {
          // Rename: source path is deleted, destination path is added
          const renamedPath = unquoteCPath(paths[1]);
          if (!renamedPath) {
            core.warning(`pushSignedCommits: rename entry missing destination path, skipping: ${line}`);
            continue;
          }
          deletions.push({ path: filePath });
          if (dstMode === "120000") {
            core.warning(`pushSignedCommits: symlink ${renamedPath} cannot be pushed as a signed commit, falling back to git push`);
            throw new Error("symlink file mode requires git push fallback");
          }
          if (dstMode === "100755") {
            core.warning(`pushSignedCommits: executable bit on ${renamedPath} will be lost in signed commit (GitHub GraphQL does not support mode 100755)`);
          }
          const content = fs.readFileSync(path.join(cwd, renamedPath));
          additions.push({ path: renamedPath, contents: content.toString("base64") });
        } else if (status && status.startsWith("C")) {
          // Copy: source path is kept (no deletion), only the destination path is added
          const copiedPath = unquoteCPath(paths[1]);
          if (!copiedPath) {
            core.warning(`pushSignedCommits: copy entry missing destination path, skipping: ${line}`);
            continue;
          }
          if (dstMode === "120000") {
            core.warning(`pushSignedCommits: symlink ${copiedPath} cannot be pushed as a signed commit, falling back to git push`);
            throw new Error("symlink file mode requires git push fallback");
          }
          if (dstMode === "100755") {
            core.warning(`pushSignedCommits: executable bit on ${copiedPath} will be lost in signed commit (GitHub GraphQL does not support mode 100755)`);
          }
          const content = fs.readFileSync(path.join(cwd, copiedPath));
          additions.push({ path: copiedPath, contents: content.toString("base64") });
        } else {
          // Added or Modified
          if (dstMode === "120000") {
            core.warning(`pushSignedCommits: symlink ${filePath} cannot be pushed as a signed commit, falling back to git push`);
            throw new Error("symlink file mode requires git push fallback");
          }
          if (dstMode === "100755") {
            core.warning(`pushSignedCommits: executable bit on ${filePath} will be lost in signed commit (GitHub GraphQL does not support mode 100755)`);
          }
          const content = fs.readFileSync(path.join(cwd, filePath));
          additions.push({ path: filePath, contents: content.toString("base64") });
        }
      }

      additionsMap.set(sha, additions);
      deletionsMap.set(sha, deletions);
    }

    // All commits passed the mode checks. Replay via GraphQL.
    /** @type {string | undefined} */
    let lastOid;
    for (let i = 0; i < shas.length; i++) {
      const sha = shas[i];
      core.info(`pushSignedCommits: processing commit ${i + 1}/${shas.length} sha=${sha}`);

      // Determine the expected HEAD OID for this commit.
      // After the first signed commit, reuse the OID returned by the previous GraphQL
      // mutation instead of re-querying ls-remote (works even if the branch is new).
      let expectedHeadOid;
      if (lastOid) {
        expectedHeadOid = lastOid;
        core.info(`pushSignedCommits: using chained OID from previous mutation: ${expectedHeadOid}`);
      } else {
        // First commit: check whether the branch already exists on the remote.
        const { stdout: oidOut } = await exec.getExecOutput("git", ["ls-remote", "origin", `refs/heads/${branch}`], { cwd });
        expectedHeadOid = oidOut.trim().split(/\s+/)[0];
        if (!expectedHeadOid) {
          // Branch does not exist on the remote yet.
          // createCommitOnBranch requires the branch to already exist – it does NOT auto-create branches.
          // Resolve the parent OID, create the branch on the remote via the REST API,
          // then proceed with the signed-commit mutation as normal.
          core.info(`pushSignedCommits: branch ${branch} not yet on the remote, resolving parent OID for first commit`);
          const { stdout: parentOut } = await exec.getExecOutput("git", ["rev-parse", `${sha}^`], { cwd });
          expectedHeadOid = parentOut.trim();
          if (!expectedHeadOid) {
            throw new Error(`${ERR_API}: Could not resolve OID for new branch ${branch}`);
          }
          core.info(`pushSignedCommits: creating remote branch ${branch} at parent OID ${expectedHeadOid}`);
          try {
            await githubClient.rest.git.createRef({
              owner,
              repo,
              ref: `refs/heads/${branch}`,
              sha: expectedHeadOid,
            });
            core.info(`pushSignedCommits: remote branch ${branch} created successfully`);
          } catch (createRefError) {
            /** @type {any} */
            const err = createRefError;
            const status = err && typeof err === "object" ? err.status : undefined;
            const message = err && typeof err === "object" ? String(err.message || "") : "";
            // If the branch was created concurrently between our ls-remote check and this call,
            // GitHub returns 422 "Reference refs/heads/<branch> already exists". Treat that as success and continue.
            if (status === 422 && /reference.*already exists/i.test(message)) {
              core.info(`pushSignedCommits: remote branch ${branch} was created concurrently (422 Reference already exists); continuing with signed commits`);
            } else {
              throw createRefError;
            }
          }
        } else {
          core.info(`pushSignedCommits: using remote HEAD OID from ls-remote: ${expectedHeadOid}`);
        }
      }

      // Full commit message (subject + body)
      const { stdout: msgOut } = await exec.getExecOutput("git", ["log", "-1", "--format=%B", sha], { cwd });
      const message = msgOut.trim();
      const headline = message.split("\n")[0];
      const body = message.split("\n").slice(1).join("\n").trim();
      core.info(`pushSignedCommits: commit message headline: "${headline}"`);

      const additions = additionsMap.get(sha) || [];
      const deletions = deletionsMap.get(sha) || [];
      core.info(`pushSignedCommits: file changes: ${additions.length} addition(s), ${deletions.length} deletion(s)`);

      /** @type {any} */
      const input = {
        branch: { repositoryNameWithOwner: `${owner}/${repo}`, branchName: branch },
        message: { headline, ...(body ? { body } : {}) },
        fileChanges: { additions, deletions },
        expectedHeadOid,
      };

      core.info(`pushSignedCommits: calling createCommitOnBranch mutation (expectedHeadOid=${expectedHeadOid})`);
      const result = await githubClient.graphql(
        `mutation($input: CreateCommitOnBranchInput!) {
          createCommitOnBranch(input: $input) { commit { oid } }
        }`,
        { input }
      );
      const newOid = result && result.createCommitOnBranch && result.createCommitOnBranch.commit ? result.createCommitOnBranch.commit.oid : undefined;
      if (typeof newOid !== "string" || newOid.length === 0) {
        throw new Error(`${ERR_API}: GraphQL createCommitOnBranch did not return a valid commit OID`);
      }
      lastOid = newOid;
      core.info(`pushSignedCommits: signed commit created: ${lastOid}`);
    }
    core.info(`pushSignedCommits: all ${shas.length} commit(s) pushed as signed commits`);
  } catch (graphqlError) {
    core.warning(`pushSignedCommits: GraphQL signed push failed, falling back to git push: ${graphqlError instanceof Error ? graphqlError.message : String(graphqlError)}`);
    await exec.exec("git", ["push", "origin", branch], {
      cwd,
      env: { ...process.env, ...(gitAuthEnv || {}) },
    });
  }
}

module.exports = { pushSignedCommits, unquoteCPath };
