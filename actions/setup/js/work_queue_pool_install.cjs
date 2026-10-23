// @ts-check
"use strict";

const { canonical, identity, queueError } = require("./work_queue_codec.cjs");
const store = require("./work_queue_store.cjs");
const { policyProposalFor, loadQueue } = require("./work_queue_binding.cjs");
const { requestForIntent } = require("./work_queue_intents.cjs");
const { actorFromContext } = require("./work_queue_policy.cjs");
const { verifyWorkerRoute } = require("./work_queue_provisioning.cjs");

function policyWithoutPools(policy) {
  const { pools: _pools, ...rest } = policy;
  return rest;
}

function poolFromProposal(state, proposal, poolName, config) {
  if (state.policy.authorization !== "aw" || proposal.authorization !== "aw") throw queueError("work_queue_pool_not_installed", "only AW-managed pools can be installed automatically");
  if (canonical(policyWithoutPools(state.policy)) !== canonical(policyWithoutPools(proposal))) throw queueError("work_queue_policy_proposal_mismatch", "automatic pool installation cannot change installed queue policy");
  for (const [name, pool] of Object.entries(state.policy.pools)) {
    if (canonical(proposal.pools[name]) !== canonical(pool)) throw queueError("work_queue_policy_proposal_mismatch", "automatic pool installation cannot change an installed pool");
  }
  const pool = proposal.pools[poolName];
  if (!pool) throw queueError("work_queue_pool_not_installed", "the requested pool is absent from the compiled queue policy");
  const workflows = new Set(config.work_queue_workflows ?? []);
  const contexts = new Set(config.aw_context_workflows ?? []);
  for (const profile of Object.values(pool.profiles)) {
    const name = profile.workflow.replace(/^\.github\/workflows\//, "").replace(/\.(?:lock\.yml|yml|yaml)$/, "");
    if (profile.effect_scope !== state.repository || Object.hasOwn(profile, "principal") || !workflows.has(name) || !contexts.has(name))
      throw queueError("work_queue_worker_not_compiler_approved", "new pool routes must be compiler-approved and preserve repository-scoped AW authority");
  }
  return pool;
}

async function installWorkQueuePools(options, trustedContext, poolNames) {
  if (!["administrator", "producer", "dispatcher"].includes(trustedContext.role)) return;
  poolNames = [...new Set(poolNames.map(name => identity(name, "pool")))];
  const configured = { ...options, githubClient: options.githubClient || options.queueClient };
  let latest = await loadQueue({ ...configured, policyProposal: undefined, initializationContext: undefined });
  if (!latest.projection.policy) return latest;
  const missing = [...new Set(poolNames)].filter(name => !Object.hasOwn(latest.projection.policy.pools, name));
  if (!missing.length) return latest;

  const proposal = policyProposalFor(configured);
  if (!proposal) throw queueError("work_queue_pool_not_installed", "the requested pool is absent from the installed Policy and no compiled pool proposal is available");
  const candidates = missing.map(poolName => [poolName, poolFromProposal(latest.projection, proposal, poolName, options.config ?? {})]);
  const provisioning = { githubClient: configured.githubClient, owner: options.context.repo.owner, repo: options.context.repo.repo };
  const verified = new Set();
  for (const [, pool] of candidates) {
    for (const profile of Object.values(pool.profiles)) {
      const route = `${profile.workflow}@${profile.ref}`;
      if (verified.has(route)) continue;
      await verifyWorkerRoute({ ...provisioning, profile });
      verified.add(route);
    }
  }

  for (const [poolName, pool] of candidates) {
    const operation = { kind: "PoolInstall", pool: poolName, config: pool };
    const request = requestForIntent(trustedContext, `pool-install:${poolName}`, "pool_install", { operations: [operation] });
    await (configured.publishWorkQueueRequest || store.publishWorkQueueRequest)({
      githubClient: configured.queueClient || configured.githubClient,
      owner: options.context.repo.owner,
      repo: options.context.repo.repo,
      ...(options.branch === undefined ? {} : { branch: options.branch }),
      context: trustedContext,
      actor: actorFromContext(trustedContext),
      request,
      generateOperations: state => {
        if (!state.policy || state.policy.authorization !== "aw") throw queueError("work_queue_pool_not_installed", "automatic installation requires an AW-managed queue");
        if (Object.hasOwn(state.policy.pools, poolName)) {
          if (canonical(state.policy.pools[poolName]) !== canonical(pool)) throw queueError("work_queue_policy_proposal_mismatch", "the pool was installed with a different definition");
          return { operations: [], idempotent: true };
        }
        const installedPool = poolFromProposal(state, proposal, poolName, options.config ?? {});
        if (canonical(installedPool) !== canonical(pool)) throw queueError("work_queue_policy_proposal_mismatch", "the pool proposal changed during installation");
        return { operations: [operation] };
      },
      core: options.core,
    });
    latest = await loadQueue({ ...configured, policyProposal: undefined, initializationContext: undefined });
  }
  return latest;
}

module.exports = { installWorkQueuePools };
