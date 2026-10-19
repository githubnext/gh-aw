// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { ERR_API, ERR_CONFIG, ERR_PERMISSION } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

function identityId(value) {
  if (typeof value === "number" && Number.isSafeInteger(value) && value > 0) return String(value);
  if (typeof value === "string" && /^[1-9][0-9]{0,19}$/.test(value)) return value;
  return null;
}

/** @returns {never} */
function reject(message) {
  throw new Error(`${ERR_PERMISSION}: CAO gateway provenance rejected: ${message}`);
}

/**
 * Authenticate GitHub's dispatch principal, never identities inside client_payload.
 * @param {{action?: unknown, sender?: {type?: unknown, id?: unknown, login?: unknown} | null, repository?: {id?: unknown, full_name?: unknown} | null} | null | undefined} event
 * @param {typeof github} client - github-script's client uses the runner's GitHub API host.
 * @param {NodeJS.ProcessEnv} env - Compiler-owned step environment.
 */
async function verifyGatewayProvenance(event, client, env) {
  const appId = identityId(env.GH_AW_GATEWAY_APP_ID);
  const botId = identityId(env.GH_AW_GATEWAY_APP_BOT_ID);
  if (!appId || !botId) {
    throw new Error(`${ERR_CONFIG}: Set the central GH_AW_GATEWAY_APP_ID and GH_AW_GATEWAY_APP_BOT_ID Actions secrets or variables to the approved App and bot-account numeric IDs.`);
  }
  if (env.GH_AW_GATEWAY_EVENT_NAME !== "repository_dispatch" || !/^gh-aw-webhook-[a-f0-9]{24}$/.test(env.GH_AW_GATEWAY_EVENT_TYPE || "") || event?.action !== env.GH_AW_GATEWAY_EVENT_TYPE) {
    reject("the event must use this workflow's compiled repository_dispatch route");
  }
  const sender = event?.sender;
  if (sender?.type !== "Bot" || identityId(sender.id) !== botId || identityId(env.GH_AW_GATEWAY_ACTOR_ID) !== botId) {
    reject("the top-level GitHub sender and original actor must be the approved CAO App bot");
  }
  const repositoryId = identityId(env.GH_AW_GATEWAY_REPOSITORY_ID);
  const repository = env.GH_AW_GATEWAY_REPOSITORY;
  if (
    !repositoryId ||
    identityId(event?.repository?.id) !== repositoryId ||
    typeof repository !== "string" ||
    !/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(repository) ||
    typeof event?.repository?.full_name !== "string" ||
    event.repository.full_name.toLowerCase() !== repository.toLowerCase()
  ) {
    reject("the GitHub event repository must match the workflow's destination");
  }
  const login = typeof sender.login === "string" ? sender.login : "";
  const slug = /^([A-Za-z0-9][A-Za-z0-9-]{0,99})\[bot\]$/.exec(login)?.[1];
  if (!slug) reject("the approved sender must have a GitHub App bot login");

  // Never follow payload URLs or API redirects with the workflow credential.
  /** @type {{retries: number, timeout: number, redirect: "error"}} */
  const request = { retries: 0, timeout: 15000, redirect: "error" };
  let app;
  let bot;
  try {
    app = await client.rest.apps.getBySlug({ app_slug: slug, request });
    bot = await client.rest.users.getByUsername({ username: login, request });
  } catch {
    throw new Error(`${ERR_API}: Unable to verify the CAO App-to-bot identity binding with GitHub. Check the central identity configuration and GitHub API access.`);
  }
  if (
    app?.status !== 200 ||
    identityId(app.data?.id) !== appId ||
    typeof app.data?.slug !== "string" ||
    app.data.slug.toLowerCase() !== slug.toLowerCase() ||
    bot?.status !== 200 ||
    bot.data?.type !== "Bot" ||
    identityId(bot.data.id) !== botId ||
    typeof bot.data.login !== "string" ||
    bot.data.login.toLowerCase() !== login.toLowerCase()
  ) {
    reject("GitHub's App and bot metadata do not match the centrally approved identity");
  }
  return { verified: true };
}

function readEvent() {
  try {
    if (!process.env.GITHUB_EVENT_PATH) reject("GITHUB_EVENT_PATH is required");
    return JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, "utf8"));
  } catch {
    return reject("GITHUB_EVENT_PATH must contain the readable original GitHub JSON event");
  }
}

async function main() {
  core.setOutput("verified", "false");
  try {
    await verifyGatewayProvenance(readEvent(), github, process.env);
    core.setOutput("verified", "true");
    core.info("Verified the approved CAO App dispatch principal.");
    return;
  } catch (error) {
    const message = getErrorMessage(error);
    core.setFailed(message);
    return core.summary
      .addRaw(
        `## CAO gateway provenance rejected\n\n${message}\n\nConfigure the central CAO App identity and dispatch using its approved repository-scoped installation token. Human roles, allowed bots, and fallback credentials cannot authorize gateway dispatches.\n\nSee the \`gateway_provenance\` job log for details.\n`
      )
      .write();
  }
}

module.exports = { main, verifyGatewayProvenance };
