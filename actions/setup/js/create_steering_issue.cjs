// @ts-check
/// <reference types="@actions/github-script" />

const { getPromptPath, renderTemplateFromFile } = require("./messages_core.cjs");
const { sanitizeTitle } = require("./sanitize_title.cjs");

const WIP_TITLE_MARKER = "[WIP] ";
const MAX_ISSUE_TITLE_LENGTH = 256;
const MAX_STEERING_TITLE_BODY_LENGTH = MAX_ISSUE_TITLE_LENGTH - WIP_TITLE_MARKER.length;

async function main() {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || context.workflow || "Agentic workflow";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;
  const coreTitle = sanitizeTitle(`${workflowName}: work in progress`, "", MAX_STEERING_TITLE_BODY_LENGTH);
  const title = `${WIP_TITLE_MARKER}${coreTitle}`;
  const body = renderTemplateFromFile(getPromptPath("steering_issue_body.md"), {
    run_url: runUrl,
    workflow_name: workflowName,
  }).trimEnd();

  const { data: issue } = await github.rest.issues.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title,
    body,
  });

  core.setOutput("issue_number", issue.number);
  core.setOutput("issue_url", issue.html_url);
}

module.exports = { main };
