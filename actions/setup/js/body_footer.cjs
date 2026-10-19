// @ts-check
/// <reference types="@actions/github-script" />

const { getBodyFooterMessage } = require("./messages_footer.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { fitMarkdownBody } = require("./report_size.cjs");

/**
 * Append a configured deterministic body footer using the current workflow context.
 * When `maxLength` is provided, fit agent content to the remaining budget without
 * shortening or omitting the configured footer.
 * @param {string} body
 * @param {string|undefined} template
 * @param {{workflowRepo?: any, maxLength?: number}} [options]
 * @returns {string}
 */
function appendConfiguredBodyFooter(body, template, options = {}) {
  const { workflowRepo, maxLength } = options || {};
  if (!template) return maxLength === undefined ? body : fitMarkdownBody(body, { maxLength });
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
  const runUrl = buildWorkflowRunUrl(context, workflowRepo || context.repo);
  const bodyFooter = getBodyFooterMessage(template, { workflowName, runUrl });
  if (!bodyFooter) return maxLength === undefined ? body : fitMarkdownBody(body, { maxLength });
  const trimmedBody = body.trimEnd();
  const composed = trimmedBody ? `${trimmedBody}\n\n${bodyFooter.trimEnd()}` : bodyFooter.trimEnd();
  if (typeof maxLength === "number" && maxLength > 0 && composed.length > maxLength) {
    return fitMarkdownBody(trimmedBody, { suffix: trimmedBody ? `\n\n${bodyFooter.trimEnd()}` : bodyFooter.trimEnd(), maxLength });
  }
  return composed;
}

module.exports = { appendConfiguredBodyFooter };
