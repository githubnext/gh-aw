// @ts-check
/// <reference types="@actions/github-script" />

const { createReportIssue } = require("./issue_report.cjs");

const { getPromptPath, renderTemplateFromFile } = require("./messages_core.cjs");
const { buildGeneratedFooter } = require("./markdown_body_helpers.cjs");
const { sanitizeTitle } = require("./sanitize_title.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { generateIssueClusteringHint } = require("./issue_clustering.cjs");

const WIP_TITLE_MARKER = "[WIP] ";
const MAX_ISSUE_TITLE_LENGTH = 256;
const MAX_STEERING_TITLE_BODY_LENGTH = MAX_ISSUE_TITLE_LENGTH - WIP_TITLE_MARKER.length;

async function main() {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || context.workflow || "Agentic workflow";
  const runUrl = buildWorkflowRunUrl(context, context.repo);
  const coreTitle = sanitizeTitle(`${workflowName}: work in progress`, "", MAX_STEERING_TITLE_BODY_LENGTH);
  const title = `${WIP_TITLE_MARKER}${coreTitle}`;
  const steeringBody = renderTemplateFromFile(getPromptPath("steering_issue_body.md"), {
    run_url: runUrl,
    workflow_name: workflowName,
  }).trimEnd();
  const footer = buildGeneratedFooter({
    workflowName,
    runUrl,
    workflowSource: process.env.GH_AW_WORKFLOW_SOURCE || "",
    workflowSourceURL: process.env.GH_AW_WORKFLOW_SOURCE_URL || "",
    triggeringIssueNumber: context.payload?.issue?.number,
    triggeringPRNumber: context.payload?.pull_request?.number,
    triggeringDiscussionNumber: context.payload?.discussion?.number,
  });
  const body = `${steeringBody}\n\n${footer}\n\n${generateIssueClusteringHint("tracking", ["steering"])}`;

  const { data: issue } = await createReportIssue(github, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    title,
    body,
  });

  core.setOutput("issue_number", issue.number);
  core.setOutput("issue_url", issue.html_url);
}

module.exports = { main };
