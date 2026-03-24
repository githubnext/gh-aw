// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("fs")} */
const fs = require("fs");
/** @type {typeof import("path")} */
const path = require("path");
const { ERR_API } = require("./error_codes.cjs");

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

      // File changes for this commit (supports Add/Modify/Delete/Rename/Copy)
      const { stdout: nameStatusOut } = await exec.getExecOutput("git", ["diff", "--name-status", `${sha}^`, sha], { cwd });
      /** @type {Array<{path: string, contents: string}>} */
      const additions = [];
      /** @type {Array<{path: string}>} */
      const deletions = [];

      for (const line of nameStatusOut.trim().split("\n").filter(Boolean)) {
        const parts = line.split("\t");
        const status = parts[0];
        if (status === "D") {
          deletions.push({ path: parts[1] });
        } else if (status.startsWith("R") || status.startsWith("C")) {
          // Rename or Copy: parts[1] = old path, parts[2] = new path
          deletions.push({ path: parts[1] });
          const content = fs.readFileSync(path.join(cwd, parts[2]));
          additions.push({ path: parts[2], contents: content.toString("base64") });
        } else {
          // Added or Modified
          const content = fs.readFileSync(path.join(cwd, parts[1]));
          additions.push({ path: parts[1], contents: content.toString("base64") });
        }
      }

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

module.exports = { pushSignedCommits };
