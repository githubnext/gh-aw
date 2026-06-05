// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Footer Message Module
 *
 * This module provides footer and installation instructions generation
 * for safe-output workflows.
 */

const { getMessages, renderTemplate, renderTemplateFromFile, toSnakeCase, getPromptPath } = require("./messages_core.cjs");
const { getMissingInfoSections } = require("./missing_messages_helper.cjs");
const { getBlockedDomains, generateBlockedDomainsSection } = require("./firewall_blocked_domains.cjs");
const { getDifcFilteredEvents, generateDifcFilteredSection } = require("./gateway_difc_filtered.cjs");
const { formatET, reduceModelNameToIdentifier, resolveActualModelName } = require("./effective_tokens.cjs");
const { formatAIC } = require("./model_costs.cjs");
const { getDetectionWarningMessage } = require("./messages_run_status.cjs");

/**
 * Get the detection caution alert if the detection job found a potential issue.
 * Reads GH_AW_DETECTION_CONCLUSION and GH_AW_DETECTION_REASON from environment variables.
 * Returns the caution alert markdown when conclusion is "warning", or empty string otherwise.
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {string} Caution alert markdown or empty string
 */
function getDetectionCautionAlert(workflowName, runUrl) {
  const detectionConclusion = process.env.GH_AW_DETECTION_CONCLUSION;
  if (detectionConclusion !== "warning") {
    return "";
  }
  const detectionReason = process.env.GH_AW_DETECTION_REASON || "";
  return getDetectionWarningMessage({ workflowName, runUrl, reason: detectionReason });
}

/**
 * Read effective tokens from the GH_AW_EFFECTIVE_TOKENS environment variable and return
 * both the raw count, compact formatted string, and a pre-formatted suffix.
 * Returns undefined/empty for all fields when the variable is absent or the parsed value
 * is not a positive integer.
 * @param {string} modelName
 * @returns {{ effectiveTokens: number|undefined, effectiveTokensFormatted: string|undefined, effectiveTokensSuffix: string }}
 */
function getEffectiveTokensFromEnv(modelName) {
  const raw = process.env.GH_AW_EFFECTIVE_TOKENS;
  const parsed = raw ? parseInt(raw, 10) : NaN;
  if (!isNaN(parsed) && parsed > 0) {
    const modelPrefix = buildModelPrefix(modelName);
    const effectiveTokensFormatted = formatET(parsed);
    return { effectiveTokens: parsed, effectiveTokensFormatted, effectiveTokensSuffix: ` · ${modelPrefix}${effectiveTokensFormatted}` };
  }
  return { effectiveTokens: undefined, effectiveTokensFormatted: undefined, effectiveTokensSuffix: "" };
}

/**
 * Read AI Credits from GH_AW_AIC and return the raw value, formatted value, and suffix.
 * @returns {{ aiCredits: number|undefined, aiCreditsFormatted: string|undefined, aiCreditsSuffix: string }}
 */
function getAICFromEnv() {
  const raw = process.env.GH_AW_AIC;
  const parsed = raw ? Number.parseFloat(raw) : NaN;
  if (Number.isFinite(parsed) && parsed > 0) {
    const aiCreditsFormatted = formatAIC(parsed);
    return { aiCredits: parsed, aiCreditsFormatted, aiCreditsSuffix: aiCreditsFormatted ? ` · ${aiCreditsFormatted} AIC` : "" };
  }
  return { aiCredits: undefined, aiCreditsFormatted: undefined, aiCreditsSuffix: "" };
}

/**
 * @param {string} modelName
 * @returns {string}
 */
function buildModelPrefix(modelName) {
  const reducedModel = reduceModelNameToIdentifier(modelName);
  return reducedModel ? `${reducedModel} ` : "";
}

/**
 * Determine whether to append AI Credits to effective_tokens_suffix in custom
 * templates for backward compatibility.
 * @param {string} footerTemplate
 * @param {string} effectiveTokensSuffix
 * @param {string} aiCreditsSuffix
 * @returns {boolean}
 */
function shouldAppendAICToEffectiveTokensSuffix(footerTemplate, effectiveTokensSuffix, aiCreditsSuffix) {
  const includesEffectiveTokensSuffix = footerTemplate.includes("{effective_tokens_suffix}");
  const includesAICSuffix = footerTemplate.includes("{ai_credits_suffix}");
  return includesEffectiveTokensSuffix && !includesAICSuffix && effectiveTokensSuffix.length > 0 && aiCreditsSuffix.length > 0;
}

/**
 * @typedef {Object} FooterContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [agenticWorkflowUrl] - Direct URL to the agentic workflow page ({run_url}/agentic_workflow)
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {number|string} [triggeringNumber] - Issue, PR, or discussion number that triggered this workflow
 * @property {string} [historyUrl] - GitHub search URL for items created by this workflow (for the history link)
 * @property {string} [historyLink] - Pre-formatted markdown history link (e.g. " · [◷](url)"), or "" if unavailable
 * @property {number} [effectiveTokens] - Total effective token count for the run (shown as N when > 0, in compact format)
 * @property {number} [aiCredits] - Total AI Credits cost for the run (1 AIC == 0.01 USD)
 * @property {string} [model] - Model name used for the run, used to build a compact model identifier in ET suffixes
 * @property {string} [emoji] - Optional emoji representing the workflow (from frontmatter)
 * @property {string} [slashCommand] - Slash command name (without leading slash) for the run-again hint, when applicable
 * @property {string} [slashCommandPlaceholder] - Custom hint text appended after the command name (replaces default "to run again")
 */

/**
 * Get the footer message, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer message
 */
function getFooterMessage(ctx) {
  const messages = getMessages();

  // Use effectiveTokens from context if provided, otherwise fall back to env var.
  // This ensures callers that don't pass effectiveTokens (e.g. update_activation_comment.cjs)
  // still get the effective token count in the footer when GH_AW_EFFECTIVE_TOKENS is set.
  // Prefer the actual model name from token-usage data (primary_model in agent_usage.json)
  // over GH_AW_ENGINE_MODEL, which may be a user-supplied alias (e.g. "agent").
  const resolvedModelName = ctx.model || resolveActualModelName();
  const { effectiveTokens: envEffectiveTokens, effectiveTokensFormatted: envEffectiveTokensFormatted, effectiveTokensSuffix: envEffectiveTokensSuffix } = getEffectiveTokensFromEnv(resolvedModelName);
  const { aiCredits: envAIC, aiCreditsFormatted: envAICFormatted, aiCreditsSuffix: envAICSuffix } = getAICFromEnv();
  const effectiveTokens = ctx.effectiveTokens ?? envEffectiveTokens;
  const aiCredits = ctx.aiCredits ?? envAIC;

  // Pre-compute history_link as a ready-to-use markdown suffix (empty string when unavailable)
  const historyLink = ctx.historyUrl ? ` · [◷](${ctx.historyUrl})` : "";

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Pre-compute effective_tokens_formatted and effective_tokens_suffix for use in custom templates
  const hasExplicitContextEffectiveTokens = ctx.effectiveTokens !== undefined && ctx.effectiveTokens !== null;
  let effectiveTokensFormatted = envEffectiveTokensFormatted;
  // effective_tokens_suffix is always a string: either " · 1.2K" or "" (for safe use in templates)
  let effectiveTokensSuffix = envEffectiveTokensSuffix;
  if (hasExplicitContextEffectiveTokens) {
    effectiveTokensFormatted = effectiveTokens ? formatET(effectiveTokens) : undefined;
    if (effectiveTokensFormatted) {
      const modelPrefix = buildModelPrefix(resolvedModelName);
      effectiveTokensSuffix = ` · ${modelPrefix}${effectiveTokensFormatted}`;
    } else {
      effectiveTokensSuffix = "";
    }
  }
  const hasExplicitContextAIC = ctx.aiCredits !== undefined && ctx.aiCredits !== null;
  let aiCreditsFormatted = envAICFormatted;
  let aiCreditsSuffix = envAICSuffix;
  if (hasExplicitContextAIC) {
    aiCreditsFormatted = aiCredits ? formatAIC(aiCredits) : undefined;
    aiCreditsSuffix = aiCreditsFormatted ? ` · ${aiCreditsFormatted} AIC` : "";
  }

  // Backward compatibility for generated custom footers that currently include only
  // {effective_tokens_suffix}: when AIC is available, append it immediately after ET
  // unless the template already opts in via {ai_credits_suffix}.
  const customFooterTemplate = messages?.footer || "";
  const shouldAppendAIC = shouldAppendAICToEffectiveTokensSuffix(customFooterTemplate, effectiveTokensSuffix, aiCreditsSuffix);
  const finalEffectiveTokensSuffix = shouldAppendAIC ? `${effectiveTokensSuffix}${aiCreditsSuffix}` : effectiveTokensSuffix;

  // Create context with both camelCase and snake_case keys, including computed history_link and agentic_workflow_url
  const templateContext = toSnakeCase({
    ...ctx,
    effectiveTokens,
    aiCredits,
    historyLink,
    agenticWorkflowUrl,
    effectiveTokensFormatted,
    effectiveTokensSuffix: finalEffectiveTokensSuffix,
    aiCreditsFormatted,
    aiCreditsSuffix,
  });

  // Use custom footer template if configured (no automatic suffix appended)
  if (messages?.footer) {
    return renderTemplate(messages.footer, templateContext);
  }

  // Default footer template - includes emoji prefix when available
  const workflowLabel = ctx.emoji ? `${ctx.emoji} {workflow_name}` : "{workflow_name}";
  let defaultFooter = `> Generated by [${workflowLabel}]({run_url})`;
  if (ctx.triggeringNumber) {
    defaultFooter += " for issue #{triggering_number}";
  }
  if (aiCredits) {
    defaultFooter += aiCreditsSuffix;
  }
  // Append history link when available
  if (ctx.historyUrl) {
    defaultFooter += " · [◷]({history_url})";
  }
  // Append slash command hint when applicable (workflow has a slash command trigger)
  if (ctx.slashCommand) {
    const hintText = ctx.slashCommandPlaceholder || "to run again";
    defaultFooter += `\n> <sub>Comment <em>/{slash_command}</em> ${hintText}</sub>`;
  }
  return renderTemplate(defaultFooter, templateContext);
}

/**
 * Get the footer installation instructions, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer installation message or empty string if no source
 */
function getFooterInstallMessage(ctx) {
  if (!ctx.workflowSource || !ctx.workflowSourceUrl) {
    return "";
  }

  const messages = getMessages();

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Create context with both camelCase and snake_case keys, including computed agentic_workflow_url
  const templateContext = toSnakeCase({ ...ctx, agenticWorkflowUrl });

  const defaultInstallTemplatePath = getPromptPath("workflow_install_note.md");

  // Use custom installation message if configured
  return messages?.footerInstall ? renderTemplate(messages.footerInstall, templateContext) : renderTemplateFromFile(defaultInstallTemplatePath, templateContext);
}

/**
 * @typedef {Object} WorkflowRecompileContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [agenticWorkflowUrl] - Direct URL to the agentic workflow page ({run_url}/agentic_workflow)
 * @property {string} repository - Repository name (owner/repo)
 */

/**
 * Get the footer message for workflow recompile issues, using custom template if configured.
 * @param {WorkflowRecompileContext} ctx - Context for footer generation
 * @returns {string} Footer message for workflow recompile issues
 */
function getFooterWorkflowRecompileMessage(ctx) {
  const messages = getMessages();

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Read effective tokens from environment variable if available
  const modelName = resolveActualModelName();
  const { effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix } = getEffectiveTokensFromEnv(modelName);

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase({ ...ctx, agenticWorkflowUrl, effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix });

  // Default footer template
  const defaultFooter = "> Generated by [{workflow_name}]({run_url})";

  // Use custom workflow recompile footer if configured, otherwise use default footer
  let footer = messages?.footerWorkflowRecompile ? renderTemplate(messages.footerWorkflowRecompile, templateContext) : renderTemplate(defaultFooter, templateContext);

  return footer;
}

/**
 * Get the footer message for comments on workflow recompile issues, using custom template if configured.
 * @param {WorkflowRecompileContext} ctx - Context for footer generation
 * @returns {string} Footer message for comments on workflow recompile issues
 */
function getFooterWorkflowRecompileCommentMessage(ctx) {
  const messages = getMessages();

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Read effective tokens from environment variable if available
  const modelName = resolveActualModelName();
  const { effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix } = getEffectiveTokensFromEnv(modelName);

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase({ ...ctx, agenticWorkflowUrl, effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix });

  // Default footer template
  const defaultFooter = "> Updated by [{workflow_name}]({run_url})";

  // Use custom workflow recompile comment footer if configured, otherwise use default footer
  let footer = messages?.footerWorkflowRecompileComment ? renderTemplate(messages.footerWorkflowRecompileComment, templateContext) : renderTemplate(defaultFooter, templateContext);

  return footer;
}

/**
 * @typedef {Object} AgentFailureContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [agenticWorkflowUrl] - Direct URL to the agentic workflow page ({run_url}/agentic_workflow)
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {string} [historyUrl] - GitHub search URL for issues created by this workflow (for the history link)
 */

/**
 * Get the footer message for agent failure tracking issues, using custom template if configured.
 * @param {AgentFailureContext} ctx - Context for footer generation
 * @returns {string} Footer message for agent failure tracking issues
 */
function getFooterAgentFailureIssueMessage(ctx) {
  const messages = getMessages();

  // Pre-compute history_link as a ready-to-use markdown suffix (empty string when unavailable)
  const historyLink = ctx.historyUrl ? ` · [◷](${ctx.historyUrl})` : "";

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Read effective tokens from environment variable if available
  const modelName = resolveActualModelName();
  const { effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix } = getEffectiveTokensFromEnv(modelName);

  // Create context with both camelCase and snake_case keys, including computed history_link and agentic_workflow_url
  const templateContext = toSnakeCase({ ...ctx, historyLink, agenticWorkflowUrl, effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix });

  // Use custom agent failure issue footer if configured, otherwise use default footer
  let footer;
  if (messages?.agentFailureIssue) {
    footer = renderTemplate(messages.agentFailureIssue, templateContext);
  } else {
    // Default footer template with link to workflow run
    let defaultFooter = "> Generated from [{workflow_name}]({run_url})";
    // Append history link when available
    if (ctx.historyUrl) {
      defaultFooter += " · [◷]({history_url})";
    }
    footer = renderTemplate(defaultFooter, templateContext);
  }

  return footer;
}

/**
 * Get the footer message for comments on agent failure tracking issues, using custom template if configured.
 * @param {AgentFailureContext} ctx - Context for footer generation
 * @returns {string} Footer message for comments on agent failure tracking issues
 */
function getFooterAgentFailureCommentMessage(ctx) {
  const messages = getMessages();

  // Pre-compute history_link as a ready-to-use markdown suffix (empty string when unavailable)
  const historyLink = ctx.historyUrl ? ` · [◷](${ctx.historyUrl})` : "";

  // Pre-compute agentic_workflow_url as the direct link to the agentic workflow page
  const agenticWorkflowUrl = ctx.agenticWorkflowUrl || (ctx.runUrl ? `${ctx.runUrl}/agentic_workflow` : "");

  // Read effective tokens from environment variable if available
  const modelName = resolveActualModelName();
  const { effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix } = getEffectiveTokensFromEnv(modelName);

  // Create context with both camelCase and snake_case keys, including computed history_link and agentic_workflow_url
  const templateContext = toSnakeCase({ ...ctx, historyLink, agenticWorkflowUrl, effectiveTokens, effectiveTokensFormatted, effectiveTokensSuffix });

  // Use custom agent failure comment footer if configured, otherwise use default footer
  let footer;
  if (messages?.agentFailureComment) {
    footer = renderTemplate(messages.agentFailureComment, templateContext);
  } else {
    // Default footer template with link to workflow run
    let defaultFooter = "> Generated from [{workflow_name}]({run_url})";
    // Append history link when available
    if (ctx.historyUrl) {
      defaultFooter += " · [◷]({history_url})";
    }
    footer = renderTemplate(defaultFooter, templateContext);
  }

  return footer;
}

/**
 * Generates an XML comment marker with agentic workflow metadata for traceability.
 * This marker enables searching and tracing back items generated by an agentic workflow.
 *
 * The marker format is:
 * <!-- gh-aw-agentic-workflow: workflow-name, gh-aw-tracker-id: id, engine: copilot, version: 1.0.0, model: gpt-5, run: https://github.com/... -->
 *
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {string} XML comment marker with workflow metadata
 */
function generateXMLMarker(workflowName, runUrl) {
  // Read engine metadata from environment variables
  const engineId = process.env.GH_AW_ENGINE_ID || "";
  const engineVersion = process.env.GH_AW_ENGINE_VERSION || "";
  const engineModel = process.env.GH_AW_ENGINE_MODEL || "";
  const trackerId = process.env.GH_AW_TRACKER_ID || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const workflowId = process.env.GH_AW_WORKFLOW_ID || "";

  // Build the key-value pairs for the marker
  const parts = [];

  // Always include agentic-workflow name
  parts.push(`gh-aw-agentic-workflow: ${workflowName}`);

  // Add tracker-id if available (for searchability and tracing)
  if (trackerId) {
    parts.push(`gh-aw-tracker-id: ${trackerId}`);
  }

  // Add engine ID if available
  if (engineId) {
    parts.push(`engine: ${engineId}`);
  }

  // Add version if available
  if (engineVersion) {
    parts.push(`version: ${engineVersion}`);
  }

  // Add model if available
  if (engineModel) {
    parts.push(`model: ${engineModel}`);
  }

  // Add numeric run ID if available
  if (runId) {
    parts.push(`id: ${runId}`);
  }

  // Add workflow identifier if available
  if (workflowId) {
    parts.push(`workflow_id: ${workflowId}`);
  }

  // Always include run URL
  parts.push(`run: ${runUrl}`);

  // Return the XML comment marker
  return `<!-- ${parts.join(", ")} -->`;
}

/**
 * @typedef {Object} GenerateFooterOptions
 * @property {boolean} [skipDetectionCaution=false] - When true, omit the threat detection caution alert
 *   from the footer. Use this when the caution alert has already been placed at the top of the body.
 */

/**
 * Generate the complete footer with AI attribution and optional installation instructions.
 * This is a drop-in replacement for the original generateFooter function.
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} workflowSource - Source of the workflow (owner/repo/path@ref)
 * @param {string} workflowSourceURL - GitHub URL for the workflow source
 * @param {number|undefined} triggeringIssueNumber - Issue number that triggered this workflow
 * @param {number|undefined} triggeringPRNumber - Pull request number that triggered this workflow
 * @param {number|undefined} triggeringDiscussionNumber - Discussion number that triggered this workflow
 * @param {string|null|undefined} [historyUrl] - GitHub search URL for items created by this workflow
 * @param {GenerateFooterOptions} [options] - Optional generation flags
 * @returns {string} Complete footer text
 */
function generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber, historyUrl, options) {
  // Determine triggering number (issue takes precedence, then PR, then discussion)
  let triggeringNumber;
  if (triggeringIssueNumber) {
    triggeringNumber = triggeringIssueNumber;
  } else if (triggeringPRNumber) {
    triggeringNumber = triggeringPRNumber;
  } else if (triggeringDiscussionNumber) {
    triggeringNumber = `discussion #${triggeringDiscussionNumber}`;
  }

  // Read effective tokens from environment variable if available.
  // GH_AW_EFFECTIVE_TOKENS is set by parse_mcp_gateway_log.cjs after computing ET
  // from the token-usage.jsonl produced by the firewall proxy.
  // Prefer the actual model name from token-usage data over GH_AW_ENGINE_MODEL alias.
  const modelName = resolveActualModelName();
  const { effectiveTokens } = getEffectiveTokensFromEnv(modelName);

  // Read workflow emoji from environment variable if available.
  const emoji = process.env.GH_AW_WORKFLOW_EMOJI || undefined;

  // Read slash command from GH_AW_COMMANDS (JSON array) when available.
  // Use the first command as the hint. This is only set when the workflow has a slash command trigger.
  let slashCommand;
  const commandsJSON = process.env.GH_AW_COMMANDS;
  if (commandsJSON) {
    try {
      const commands = JSON.parse(commandsJSON);
      if (Array.isArray(commands) && commands.length > 0 && typeof commands[0] === "string") {
        slashCommand = commands[0];
      }
    } catch {
      // Silently ignore malformed GH_AW_COMMANDS; the hint is a non-critical enhancement
      // and omitting it is always safe. The value is compiler-generated JSON, so this
      // path should not occur in practice.
    }
  }

  // Read optional footer hint placeholder from GH_AW_COMMAND_PLACEHOLDER.
  // When set, it replaces the default "to run again" suffix in the slash command hint.
  const slashCommandPlaceholder = process.env.GH_AW_COMMAND_PLACEHOLDER;

  const ctx = {
    workflowName,
    runUrl,
    workflowSource,
    workflowSourceUrl: workflowSourceURL,
    triggeringNumber,
    historyUrl: historyUrl || undefined,
    effectiveTokens,
    emoji,
    slashCommand,
    slashCommandPlaceholder,
  };

  const { skipDetectionCaution = false } = options || {};

  // Collect guard notices to show BEFORE the attribution footer
  let guardNotices = "";

  // Add detection caution alert if detection job found a potential issue.
  // Skip when the caller has already placed the caution alert at the top of the body.
  if (!skipDetectionCaution) {
    const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
    if (detectionCaution) {
      guardNotices += detectionCaution;
    }
  }

  // Add firewall blocked domains section if any domains were blocked
  const blockedDomains = getBlockedDomains();
  const blockedDomainsSection = generateBlockedDomainsSection(blockedDomains);
  if (blockedDomainsSection) {
    guardNotices += blockedDomainsSection;
  }

  // Add integrity filtering section if any items were filtered
  try {
    const difcFilteredEvents = getDifcFilteredEvents();
    const difcFilteredSection = generateDifcFilteredSection(difcFilteredEvents);
    if (difcFilteredSection) {
      guardNotices += difcFilteredSection;
    }
  } catch {
    // ignore errors so the rest of the footer is always preserved
  }

  // Attribution footer line comes after any guard notices
  let footer = guardNotices + "\n\n" + getFooterMessage(ctx);

  // Add installation instructions if source is available
  const installMessage = getFooterInstallMessage(ctx);
  if (installMessage) {
    footer += "\n>\n" + installMessage;
  }

  // Add missing tools and data sections if available
  const missingInfoSections = getMissingInfoSections();
  if (missingInfoSections) {
    footer += missingInfoSections;
  }

  // Add XML comment marker for traceability
  footer += "\n\n" + generateXMLMarker(workflowName, runUrl);

  footer += "\n";
  return footer;
}

module.exports = {
  getDetectionCautionAlert,
  getFooterMessage,
  getFooterInstallMessage,
  getFooterWorkflowRecompileMessage,
  getFooterWorkflowRecompileCommentMessage,
  getFooterAgentFailureIssueMessage,
  getFooterAgentFailureCommentMessage,
  generateFooterWithMessages,
  generateXMLMarker,
};
