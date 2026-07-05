// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_VALIDATION } = require("./error_codes.cjs");
const { resolveInvocationContext } = require("./invocation_context_helpers.cjs");

/** Valid GitHub reaction types */
const VALID_REACTIONS = Object.freeze(["+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"]);

/**
 * Resolve and validate reaction input plus invocation context.
 * @param {any} rawContext
 * @returns {{ reaction: string, invocationContext: ReturnType<typeof resolveInvocationContext> } | null }
 */
function resolveReactionSetup(rawContext) {
  const reaction = process.env.GH_AW_REACTION || "eyes";
  if (!VALID_REACTIONS.includes(reaction)) {
    core.setFailed(`${ERR_VALIDATION}: Invalid reaction type: ${reaction}. Valid reactions are: ${VALID_REACTIONS.join(", ")}`);
    return null;
  }
  const invocationContext = resolveInvocationContext(rawContext);
  return { reaction, invocationContext };
}

module.exports = { VALID_REACTIONS, resolveReactionSetup };
