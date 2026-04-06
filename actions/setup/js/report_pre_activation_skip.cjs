// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Reports why the workflow was not activated (skipped by pre-activation gate).
 * Called from the pre_activation job with `if: always()` condition.
 * Only writes to job summary when activation was denied by one or more checks.
 *
 * Environment variables (all optional — empty means "check was not configured"):
 *   GH_AW_IS_TEAM_MEMBER            - "true"/"false" from check_membership step
 *   GH_AW_MEMBERSHIP_RESULT         - result code from check_membership step
 *   GH_AW_MEMBERSHIP_ERROR_MESSAGE  - human-readable error from check_membership step
 *   GH_AW_STOP_TIME_OK              - "true"/"false" from check_stop_time step
 *   GH_AW_RATE_LIMIT_OK             - "true"/"false" from check_rate_limit step
 *   GH_AW_SKIP_CHECK_OK             - "true"/"false" from check_skip_if_match step
 *   GH_AW_SKIP_NO_MATCH_OK          - "true"/"false" from check_skip_if_no_match step
 *   GH_AW_SKIP_IF_CHECK_FAILING_OK  - "true"/"false" from check_skip_if_check_failing step
 *   GH_AW_SKIP_ROLES_OK             - "true"/"false" from check_skip_roles step
 *   GH_AW_SKIP_ROLES_ERROR_MESSAGE  - human-readable error from check_skip_roles step
 *   GH_AW_SKIP_BOTS_OK              - "true"/"false" from check_skip_bots step
 *   GH_AW_SKIP_BOTS_ERROR_MESSAGE   - human-readable error from check_skip_bots step
 *   GH_AW_COMMAND_POSITION_OK       - "true"/"false" from check_command_position step
 */

/**
 * @typedef {{ check: string, message: string, result: string, remediation: string }} SkipReason
 */

async function main() {
  const reasons = collectSkipReasons();

  if (reasons.length === 0) {
    // All checks passed (or no checks were configured) — nothing to report.
    core.info("✅ All pre-activation checks passed, no skip reason to report.");
    return;
  }

  core.info(`⏭️ Workflow activation denied by ${reasons.length} check(s). Writing skip reason to job summary...`);
  await writeSkipSummary(reasons);
}

/**
 * Collects skip reasons from the environment variables set by each check step.
 * A check is considered failing when its ok-output is exactly "false".
 * An empty / undefined value means the check was not configured — skip it.
 *
 * @returns {SkipReason[]}
 */
function collectSkipReasons() {
  /** @type {SkipReason[]} */
  const reasons = [];

  // Role / bot check (check_membership step)
  const isTeamMember = process.env.GH_AW_IS_TEAM_MEMBER;
  if (isTeamMember === "false") {
    const result = process.env.GH_AW_MEMBERSHIP_RESULT || "insufficient_permissions";
    const errorMsg = process.env.GH_AW_MEMBERSHIP_ERROR_MESSAGE || "Actor does not have the required repository permissions.";
    reasons.push({
      check: "Role / bot check",
      message: errorMsg,
      result,
      remediation: buildMembershipRemediation(result),
    });
  }

  // Stop-time check (check_stop_time step)
  const stopTimeOk = process.env.GH_AW_STOP_TIME_OK;
  if (stopTimeOk === "false") {
    reasons.push({
      check: "Stop-time limit",
      message: "The workflow is past its configured stop-time limit.",
      result: "stop_time_exceeded",
      remediation: "Update or remove `on.stop-time:` in the workflow frontmatter to extend the active window.",
    });
  }

  // Rate-limit check (check_rate_limit step)
  const rateLimitOk = process.env.GH_AW_RATE_LIMIT_OK;
  if (rateLimitOk === "false") {
    reasons.push({
      check: "Rate-limit check",
      message: "The actor has exceeded the configured rate limit for this workflow.",
      result: "rate_limit_exceeded",
      remediation: "Adjust `on.rate-limit:` in the workflow frontmatter, or wait for the time window to expire.",
    });
  }

  // Skip-if-match check (check_skip_if_match step)
  const skipCheckOk = process.env.GH_AW_SKIP_CHECK_OK;
  if (skipCheckOk === "false") {
    reasons.push({
      check: "Skip-if-match check",
      message: "The skip-if-match query matched, so the workflow was intentionally skipped.",
      result: "skip_if_match",
      remediation: "Update `on.skip-if-match:` in the workflow frontmatter if this skip was unexpected.",
    });
  }

  // Skip-if-no-match check (check_skip_if_no_match step)
  const skipNoMatchOk = process.env.GH_AW_SKIP_NO_MATCH_OK;
  if (skipNoMatchOk === "false") {
    reasons.push({
      check: "Skip-if-no-match check",
      message: "The skip-if-no-match query returned no results, so the workflow was intentionally skipped.",
      result: "skip_if_no_match",
      remediation: "Update `on.skip-if-no-match:` in the workflow frontmatter if this skip was unexpected.",
    });
  }

  // Skip-if-check-failing check (check_skip_if_check_failing step)
  const skipIfCheckFailingOk = process.env.GH_AW_SKIP_IF_CHECK_FAILING_OK;
  if (skipIfCheckFailingOk === "false") {
    reasons.push({
      check: "Skip-if-check-failing",
      message: "A required check is currently failing, so the workflow was intentionally skipped.",
      result: "skip_if_check_failing",
      remediation: "Fix the failing check referenced in `on.skip-if-check-failing:`, or update the frontmatter configuration.",
    });
  }

  // Skip-roles check (check_skip_roles step)
  const skipRolesOk = process.env.GH_AW_SKIP_ROLES_OK;
  if (skipRolesOk === "false") {
    const errorMsg = process.env.GH_AW_SKIP_ROLES_ERROR_MESSAGE || "Actor has a role that is configured to skip this workflow.";
    reasons.push({
      check: "Skip-roles check",
      message: errorMsg,
      result: "skip_roles",
      remediation: "Update `on.skip-roles:` in the workflow frontmatter to change which roles are excluded.",
    });
  }

  // Skip-bots check (check_skip_bots step)
  const skipBotsOk = process.env.GH_AW_SKIP_BOTS_OK;
  if (skipBotsOk === "false") {
    const errorMsg = process.env.GH_AW_SKIP_BOTS_ERROR_MESSAGE || "Actor is in the skip-bots list for this workflow.";
    reasons.push({
      check: "Skip-bots check",
      message: errorMsg,
      result: "skip_bots",
      remediation: "Update `on.skip-bots:` in the workflow frontmatter to change which bots are excluded.",
    });
  }

  // Command-position check (check_command_position step)
  const commandPositionOk = process.env.GH_AW_COMMAND_POSITION_OK;
  if (commandPositionOk === "false") {
    reasons.push({
      check: "Command check",
      message: "The required trigger command was not found in the expected position.",
      result: "command_not_found",
      remediation: "Make sure the trigger comment starts with the required command defined in `on.command:` in the workflow frontmatter.",
    });
  }

  return reasons;
}

/**
 * Returns a remediation hint tailored to the membership check result code.
 *
 * @param {string} result - The result code from check_membership.cjs
 * @returns {string}
 */
function buildMembershipRemediation(result) {
  switch (result) {
    case "insufficient_permissions":
      return (
        "The actor does not have the required repository permission. " +
        "To allow a bot or GitHub App, add it to `on.bots:` in the workflow frontmatter. " +
        "To change the required human-actor roles, update `on.roles:` in the workflow frontmatter."
      );
    case "bot_not_active":
      return "The bot is in the allowed list but is not installed or active on this repository. Install the GitHub App and try again.";
    case "api_error":
      return "The permission check failed with a GitHub API error. Check the pre_activation job log for details.";
    case "config_error":
      return "The workflow has a permission configuration error. Contact the repository administrator.";
    default:
      return "To allow a bot or GitHub App actor, add it to `on.bots:` in the workflow frontmatter. " + "To change the required roles for human actors, update `on.roles:` in the workflow frontmatter.";
  }
}

/**
 * Writes the skip reasons to the GitHub Actions job summary.
 *
 * @param {SkipReason[]} reasons
 */
async function writeSkipSummary(reasons) {
  const lines = [];
  lines.push("## ⏭️ Workflow Activation Skipped\n");
  lines.push("This workflow run was not activated because one or more pre-activation checks denied execution.\n");

  for (const reason of reasons) {
    lines.push(`### ${reason.check}`);
    lines.push(`> ${reason.message}\n`);
    lines.push(`**Remediation:** ${reason.remediation}\n`);
  }

  lines.push("---");
  lines.push("_This summary is generated by [gh-aw](https://github.com/github/gh-aw). See the `pre_activation` job log for full details._");

  await core.summary.addRaw(lines.join("\n")).write();
  core.info("📝 Skip reason written to job summary.");
}

module.exports = { main, collectSkipReasons, buildMembershipRemediation };
