const { loadAgentOutput } = require("./load_agent_output.cjs"),
  { generateFooter } = require("./generate_footer.cjs"),
  { getTrackerID } = require("./get_tracker_id.cjs"),
  { getRepositoryUrl } = require("./get_repository_url.cjs");
function buildRunUrl() {
  const runId = context.runId,
    githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
  return context.payload.repository ? `${context.payload.repository.html_url}/actions/runs/${runId}` : `${githubServer}/${context.repo.owner}/${context.repo.repo}/actions/runs/${runId}`;
}
function buildCommentBody(body, triggeringIssueNumber, triggeringPRNumber) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow",
    workflowSource = process.env.GH_AW_WORKFLOW_SOURCE || "",
    workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL || "",
    runUrl = buildRunUrl();
  let commentBody = body.trim();
  return ((commentBody += getTrackerID("markdown")), (commentBody += generateFooter(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, void 0)), commentBody);
}
function checkLabelFilter(entityLabels, requiredLabels) {
  if (0 === requiredLabels.length) return !0;
  const labelNames = entityLabels.map(l => l.name);
  return requiredLabels.some(required => labelNames.includes(required));
}
function checkTitlePrefixFilter(title, requiredTitlePrefix) {
  return !requiredTitlePrefix || title.startsWith(requiredTitlePrefix);
}
async function generateCloseEntityStagedPreview(config, items, requiredLabels, requiredTitlePrefix) {
  let summaryContent = `## 🎭 Staged Mode: Close ${config.displayNameCapitalizedPlural} Preview\n\n`;
  summaryContent += `The following ${config.displayNamePlural} would be closed if staged mode was disabled:\n\n`;
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    summaryContent += `### ${config.displayNameCapitalized} ${i + 1}\n`;
    const entityNumber = item[config.numberField];
    if (entityNumber) {
      const entityUrl = `${getRepositoryUrl()}/${config.urlPath}/${entityNumber}`;
      summaryContent += `**Target ${config.displayNameCapitalized}:** [#${entityNumber}](${entityUrl})\n\n`;
    } else summaryContent += `**Target:** Current ${config.displayName}\n\n`;
    ((summaryContent += `**Comment:**\n${item.body || "No content provided"}\n\n`),
      requiredLabels.length > 0 && (summaryContent += `**Required Labels:** ${requiredLabels.join(", ")}\n\n`),
      requiredTitlePrefix && (summaryContent += `**Required Title Prefix:** ${requiredTitlePrefix}\n\n`),
      (summaryContent += "---\n\n"));
  }
  (await core.summary.addRaw(summaryContent).write(), core.info(`📝 ${config.displayNameCapitalized} close preview written to step summary`));
}
function parseEntityConfig(envVarPrefix) {
  const labelsEnvVar = `${envVarPrefix}_REQUIRED_LABELS`,
    titlePrefixEnvVar = `${envVarPrefix}_REQUIRED_TITLE_PREFIX`,
    targetEnvVar = `${envVarPrefix}_TARGET`;
  return { requiredLabels: process.env[labelsEnvVar] ? process.env[labelsEnvVar].split(",").map(l => l.trim()) : [], requiredTitlePrefix: process.env[titlePrefixEnvVar] || "", target: process.env[targetEnvVar] || "triggering" };
}
function resolveEntityNumber(config, target, item, isEntityContext) {
  if ("*" === target) {
    const targetNumber = item[config.numberField];
    if (targetNumber) {
      const parsed = parseInt(targetNumber, 10);
      return isNaN(parsed) || parsed <= 0 ? { success: !1, message: `Invalid ${config.displayName} number specified: ${targetNumber}` } : { success: !0, number: parsed };
    }
    return { success: !1, message: `Target is "*" but no ${config.numberField} specified in ${config.itemTypeDisplay} item` };
  }
  if ("triggering" !== target) {
    const parsed = parseInt(target, 10);
    return isNaN(parsed) || parsed <= 0 ? { success: !1, message: `Invalid ${config.displayName} number in target configuration: ${target}` } : { success: !0, number: parsed };
  }
  if (isEntityContext) {
    const number = context.payload[config.contextPayloadField]?.number;
    return number ? { success: !0, number } : { success: !1, message: `${config.displayNameCapitalized} context detected but no ${config.displayName} found in payload` };
  }
  return { success: !1, message: `Not in ${config.displayName} context and no explicit target specified` };
}
function escapeMarkdownTitle(title) {
  return title.replace(/[[\]()]/g, "\\$&");
}
async function processCloseEntityItems(config, callbacks) {
  const isStaged = "true" === process.env.GH_AW_SAFE_OUTPUTS_STAGED,
    result = loadAgentOutput();
  if (!result.success) return;
  const items = result.items.filter(item => item.type === config.itemType);
  if (0 === items.length) return void core.info(`No ${config.itemTypeDisplay} items found in agent output`);
  core.info(`Found ${items.length} ${config.itemTypeDisplay} item(s)`);
  const { requiredLabels, requiredTitlePrefix, target } = parseEntityConfig(config.envVarPrefix);
  core.info(`Configuration: requiredLabels=${requiredLabels.join(",")}, requiredTitlePrefix=${requiredTitlePrefix}, target=${target}`);
  const isEntityContext = config.contextEvents.some(event => context.eventName === event);
  if (isStaged) return void (await generateCloseEntityStagedPreview(config, items, requiredLabels, requiredTitlePrefix));
  if ("triggering" === target && !isEntityContext) return void core.info(`Target is "triggering" but not running in ${config.displayName} context, skipping ${config.displayName} close`);
  const triggeringIssueNumber = context.payload?.issue?.number,
    triggeringPRNumber = context.payload?.pull_request?.number,
    closedEntities = [];
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    core.info(`Processing ${config.itemTypeDisplay} item ${i + 1}/${items.length}: bodyLength=${item.body.length}`);
    const resolved = resolveEntityNumber(config, target, item, isEntityContext);
    if (!resolved.success) {
      core.info(resolved.message);
      continue;
    }
    const entityNumber = resolved.number;
    try {
      const entity = await callbacks.getDetails(github, context.repo.owner, context.repo.repo, entityNumber);
      if (!checkLabelFilter(entity.labels, requiredLabels)) {
        core.info(`${config.displayNameCapitalized} #${entityNumber} does not have required labels: ${requiredLabels.join(", ")}`);
        continue;
      }
      if (!checkTitlePrefixFilter(entity.title, requiredTitlePrefix)) {
        core.info(`${config.displayNameCapitalized} #${entityNumber} does not have required title prefix: ${requiredTitlePrefix}`);
        continue;
      }
      if ("closed" === entity.state) {
        core.info(`${config.displayNameCapitalized} #${entityNumber} is already closed, skipping`);
        continue;
      }
      const commentBody = buildCommentBody(item.body, triggeringIssueNumber, triggeringPRNumber),
        comment = await callbacks.addComment(github, context.repo.owner, context.repo.repo, entityNumber, commentBody);
      core.info(`✓ Added comment to ${config.displayName} #${entityNumber}: ${comment.html_url}`);
      const closedEntity = await callbacks.closeEntity(github, context.repo.owner, context.repo.repo, entityNumber);
      if ((core.info(`✓ Closed ${config.displayName} #${entityNumber}: ${closedEntity.html_url}`), closedEntities.push({ entity: closedEntity, comment }), i === items.length - 1)) {
        const numberOutputName = "issue" === config.entityType ? "issue_number" : "pull_request_number",
          urlOutputName = "issue" === config.entityType ? "issue_url" : "pull_request_url";
        (core.setOutput(numberOutputName, closedEntity.number), core.setOutput(urlOutputName, closedEntity.html_url), core.setOutput("comment_url", comment.html_url));
      }
    } catch (error) {
      throw (core.error(`✗ Failed to close ${config.displayName} #${entityNumber}: ${error instanceof Error ? error.message : String(error)}`), error);
    }
  }
  if (closedEntities.length > 0) {
    let summaryContent = `\n\n## Closed ${config.displayNameCapitalizedPlural}\n`;
    for (const { entity, comment } of closedEntities) {
      const escapedTitle = escapeMarkdownTitle(entity.title);
      summaryContent += `- ${config.displayNameCapitalized} #${entity.number}: [${escapedTitle}](${entity.html_url}) ([comment](${comment.html_url}))\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }
  return (core.info(`Successfully closed ${closedEntities.length} ${config.displayName}(s)`), closedEntities);
}
const ISSUE_CONFIG = {
    entityType: "issue",
    itemType: "close_issue",
    itemTypeDisplay: "close-issue",
    numberField: "issue_number",
    envVarPrefix: "GH_AW_CLOSE_ISSUE",
    contextEvents: ["issues", "issue_comment"],
    contextPayloadField: "issue",
    urlPath: "issues",
    displayName: "issue",
    displayNamePlural: "issues",
    displayNameCapitalized: "Issue",
    displayNameCapitalizedPlural: "Issues",
  },
  PULL_REQUEST_CONFIG = {
    entityType: "pull_request",
    itemType: "close_pull_request",
    itemTypeDisplay: "close-pull-request",
    numberField: "pull_request_number",
    envVarPrefix: "GH_AW_CLOSE_PR",
    contextEvents: ["pull_request", "pull_request_review_comment"],
    contextPayloadField: "pull_request",
    urlPath: "pull",
    displayName: "pull request",
    displayNamePlural: "pull requests",
    displayNameCapitalized: "Pull Request",
    displayNameCapitalizedPlural: "Pull Requests",
  };
module.exports = { processCloseEntityItems, generateCloseEntityStagedPreview, checkLabelFilter, checkTitlePrefixFilter, parseEntityConfig, resolveEntityNumber, buildCommentBody, escapeMarkdownTitle, ISSUE_CONFIG, PULL_REQUEST_CONFIG };
