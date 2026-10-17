// @ts-check
"use strict";
// @safe-outputs-exempt SEC-005 — creation receipts are private, Claim-bound records; delivery performs independently scoped native readback only.

const { canonical, digest } = require("./work_queue_codec.cjs");
const { currentClaimHandle, claimIdentity, receiptMatchesClaim, createClaimResourceVerification } = require("./work_queue_claim_scope.cjs");
const { resolveRepositoryTarget, resolveParentResourceTarget, nativeDecimalIdentity } = require("./work_queue_effect_resource.cjs");
const log = require("./work_queue_logging.cjs").createWorkQueueLogger("native_delivery");

const receipts = new WeakMap();
const REVISION = /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/;
const names = values => [...new Set(values.map(value => String(typeof value === "string" ? value : value.name || value.login).toLowerCase()))].sort();

function captureDiscussionDelivery(result, message, expected) {
  if (currentClaimHandle()) receipts.set(result, { ...claimIdentity(currentClaimHandle()), type: "create_discussion", message: canonical({ ...message, repo: expected.repository }), ...structuredClone(expected) });
  return result;
}

async function preparePullRequestDelivery(exec, baseBranch) {
  if (!currentClaimHandle()) return null;
  const read = async revision => {
    const result = await exec.getExecOutput("git", ["rev-parse", "--verify", revision], { silent: true });
    const sha = result.stdout.trim();
    if (result.exitCode !== 0 || !REVISION.test(sha)) throw new Error("Claim pull request preparation requires exact local commit/tree evidence");
    return sha;
  };
  return { tree: await read("HEAD^{tree}"), base: await read(`refs/remotes/origin/${baseBranch}^{commit}`) };
}

async function readAncestry(github, repository, base, head) {
  const [owner, repo] = repository.split("/");
  const { data } = await github.rest.repos.compareCommits({ owner, repo, base, head });
  return data?.base_commit?.sha === base && data.merge_base_commit?.sha === base && data.status === "ahead" && Number.isSafeInteger(data.ahead_by) && data.ahead_by > 0 && data.behind_by === 0;
}

async function readCode(github, receipt, pinnedCommit) {
  const [owner, repo] = receipt.headRepository.split("/");
  const { data: ref } = await github.rest.git.getRef({ owner, repo, ref: `heads/${receipt.branch}` });
  const sha = ref?.object?.sha;
  if (ref?.ref !== `refs/heads/${receipt.branch}` || ref.object.type !== "commit" || !REVISION.test(sha) || (pinnedCommit !== undefined && sha !== pinnedCommit)) return null;
  const { data: commit } = await github.rest.git.getCommit({ owner, repo, commit_sha: sha });
  if (commit?.sha !== sha || commit.tree?.sha !== receipt.tree || !Array.isArray(commit.parents) || !commit.parents.length || commit.parents.some(parent => !REVISION.test(parent.sha))) return null;
  const { data: tree } = await github.rest.git.getTree({ owner, repo, tree_sha: receipt.tree });
  if (tree?.sha !== receipt.tree || tree.truncated !== false || !Array.isArray(tree.tree)) return null;
  // Cross-repository PR comparisons are read from the base repository using a qualified head.
  const head = receipt.headRepository === receipt.repository ? sha : `${owner}:${sha}`;
  if (!(await readAncestry(github, receipt.repository, receipt.base, head))) return null;
  return sha;
}

async function capturePullRequestDelivery(result, message, expected, github) {
  if (!currentClaimHandle() || !expected.prepared) return result;
  const receipt = { ...claimIdentity(currentClaimHandle()), type: "create_pull_request", message: canonical({ ...message, repo: expected.repository }), ...structuredClone(expected), ...expected.prepared };
  delete receipt.prepared;
  try {
    const commit = await readCode(github, receipt);
    const repository = await resolveRepositoryTarget(github, { repository: receipt.repository });
    const headRepository = await resolveRepositoryTarget(github, { repository: receipt.headRepository });
    if (commit && repository.repository_id && headRepository.repository_id) receipts.set(result, { ...receipt, commit, repositoryId: repository.repository_id, headRepositoryId: headRepository.repository_id });
  } catch (error) {
    log.failure("creation.capture.unverified", error);
    // An unavailable readback must leave the successful write unverified, not create a fallback resource.
  }
  return result;
}

async function verifyDiscussion(options, receipt) {
  const { github, claim } = options;
  const response = await github.graphql(
    `query ClaimDiscussionDelivery($id: ID!) {
      node(id: $id) {
        __typename id
        ... on Discussion {
          number title body url
          repository { id databaseId nameWithOwner }
          category { id }
          labels(first: 100) { nodes { name } pageInfo { hasNextPage } }
        }
      }
    }`,
    { id: receipt.id }
  );
  const node = response?.node;
  if (
    typeof receipt.id !== "string" ||
    !receipt.id ||
    Buffer.byteLength(receipt.id) > 256 ||
    !Number.isSafeInteger(receipt.number) ||
    receipt.number < 1 ||
    node?.__typename !== "Discussion" ||
    node.id !== receipt.id ||
    node.number !== receipt.number ||
    node.url !== receipt.url ||
    node.repository?.id !== receipt.repositoryNodeId ||
    node.repository.nameWithOwner !== receipt.repository ||
    node.category?.id !== receipt.category ||
    node.title !== receipt.title ||
    node.body !== receipt.body ||
    node.labels?.pageInfo?.hasNextPage !== false ||
    !Array.isArray(node.labels.nodes) ||
    canonical(names(node.labels.nodes)) !== canonical(names(receipt.labels))
  )
    return { verified: false };
  const authority = await resolveRepositoryTarget(github, { repository: receipt.repository, repository_id: nativeDecimalIdentity(node.repository.databaseId), number: node.number });
  return createClaimResourceVerification({
    verified: true,
    claim_handle: claim.handle,
    resource: { kind: "discussion", repository: receipt.repository, id: node.id, number: node.number, url: node.url },
    authority_resource: authority,
    effect_resources: receipt.effectIds.map(id => ({ kind: "graphql", repository: receipt.repository, id })),
    evidence: { source: "independent_native_discussion_readback", id: node.id, category: node.category.id, digest: digest(node) },
  });
}

async function verifyPullRequest(options, receipt) {
  const { github, claim } = options;
  if (!Number.isSafeInteger(receipt.number) || receipt.number < 1 || nativeDecimalIdentity(receipt.id) !== receipt.id) return { verified: false };
  if (!(await readCode(github, receipt, receipt.commit))) return { verified: false };
  const [owner, repo] = receipt.repository.split("/");
  const { data } = await github.rest.pulls.get({ owner, repo, pull_number: receipt.number });
  if (
    !data ||
    nativeDecimalIdentity(data.id) !== receipt.id ||
    data.number !== receipt.number ||
    data.html_url !== receipt.url ||
    data.title !== receipt.title ||
    data.body !== receipt.body ||
    data.state !== "open" ||
    data.draft !== receipt.draft ||
    data.head?.sha !== receipt.commit ||
    data.head.ref !== receipt.branch ||
    data.head.repo?.full_name !== receipt.headRepository ||
    data.base?.ref !== receipt.baseBranch ||
    data.base.sha !== receipt.base ||
    data.base.repo?.full_name !== receipt.repository ||
    nativeDecimalIdentity(data.base.repo.id) !== receipt.repositoryId ||
    nativeDecimalIdentity(data.head.repo.id) !== receipt.headRepositoryId ||
    canonical(names(data.labels || [])) !== canonical(names(receipt.labels)) ||
    canonical(names(data.assignees || [])) !== canonical(names(receipt.assignees))
  )
    return { verified: false };
  const authority = await resolveParentResourceTarget(github, { repository: receipt.repository, kind: "pull_request", number: receipt.number }, data);
  const head = await resolveRepositoryTarget(github, { repository: receipt.headRepository, repository_id: nativeDecimalIdentity(data.head.repo.id), ref: `refs/heads/${receipt.branch}` });
  const effects = [];
  for (const effect of options.effects || []) {
    if (!receiptMatchesClaim(effect, claim) || effect.outcome !== "succeeded") continue;
    if (effect.repository === receipt.headRepository && effect.kind === "graphql" && effect.commit) {
      if (!REVISION.test(effect.commit) || (effect.commit !== receipt.commit && !(await readAncestry(github, receipt.headRepository, effect.commit, receipt.commit)))) return { verified: false };
      effects.push({ kind: "graphql", repository: receipt.headRepository, id: effect.id });
    }
    if (effect.repository === receipt.repository && effect.kind === "issue" && effect.number === receipt.number) {
      if (!effect.expected || !Object.keys(effect.expected).length || Object.keys(effect.expected).some(field => !["labels", "assignees"].includes(field))) continue;
      for (const field of Object.keys(effect.expected)) if (canonical(names(effect.expected[field])) !== canonical(names(receipt[field]))) return { verified: false };
      const { data: issue } = await github.rest.issues.get({ owner, repo, issue_number: receipt.number });
      const parent = await resolveParentResourceTarget(github, { repository: receipt.repository, kind: "issue", number: receipt.number }, issue);
      if (parent.kind !== "pull_request" || parent.resource_id !== receipt.id || (effect.id && effect.id !== String(issue.id))) return { verified: false };
      for (const field of Object.keys(effect.expected)) if (canonical(names(issue[field] || [])) !== canonical(names(receipt[field]))) return { verified: false };
      effects.push({ kind: "issue", repository: receipt.repository, id: nativeDecimalIdentity(issue.id), number: receipt.number });
    }
  }
  return createClaimResourceVerification({
    verified: true,
    claim_handle: claim.handle,
    resource: { kind: "pull_request", repository: receipt.repository, id: receipt.id, number: receipt.number, url: receipt.url },
    authority_resource: authority,
    authority_resources: [authority, head],
    effect_resources: effects,
    evidence: { source: "independent_native_pull_request_readback", commit: receipt.commit, tree: receipt.tree, base: receipt.base, digest: digest(data) },
  });
}

async function verifyNativeCreationDelivery(options) {
  const receipt = options.result && receipts.get(options.result);
  if (
    !receiptMatchesClaim(receipt, options.claim) ||
    receipt.type !== options.message.type ||
    (options.message.repo !== undefined && options.message.repo !== receipt.repository) ||
    receipt.message !== canonical({ ...options.message, repo: receipt.repository })
  ) {
    log.debug("creation.receipt.unverified", { receipt: !!receipt });
    return { verified: false };
  }
  const fields = new Set(["type", "claim_handle", "claim_id", "work_id", "repo", "temporary_id", "title", "body", "labels", ...(receipt.type === "create_discussion" ? ["category"] : ["branch", "base", "draft"])]);
  if (Object.keys(options.message).some(field => !fields.has(field)) || options.result.staged || options.result.fallback_used) return { verified: false };
  try {
    return receipt.type === "create_discussion" ? await verifyDiscussion(options, receipt) : await verifyPullRequest(options, receipt);
  } catch (error) {
    log.failure("creation.readback.unverified", error);
    return { verified: false };
  }
}

module.exports = { captureDiscussionDelivery, preparePullRequestDelivery, capturePullRequestDelivery, verifyNativeCreationDelivery };
