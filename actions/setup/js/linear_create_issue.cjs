// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeTitle } = require("./sanitize_title.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { linearGraphQL } = require("./linear_graphql.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { ERR_API, ERR_CONFIG, ERR_VALIDATION } = require("./error_codes.cjs");

const LINEAR_CREATE_ISSUE = `mutation LinearCreateIssue($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue {
      id
      identifier
      title
    }
  }
}`;

async function main(config = {}) {
  const teamId = config.team_id;
  if (typeof teamId !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(teamId)) {
    throw new Error(`${ERR_CONFIG}: linear_create_issue requires a valid configured team ID`);
  }

  return async function handleLinearCreateIssue(item) {
    if (typeof item?.title !== "string" || !item.title.trim()) {
      throw new Error(`${ERR_VALIDATION}: linear_create_issue title is required`);
    }
    if (typeof item?.body !== "string" || !item.body.trim()) {
      throw new Error(`${ERR_VALIDATION}: linear_create_issue body is required`);
    }
    if (item.title.length > 128 || item.body.length > 65000 || item.body.length < 20) {
      throw new Error(`${ERR_VALIDATION}: linear_create_issue content exceeds the configured field limits`);
    }

    const title = sanitizeTitle(item.title);
    const description = sanitizeContent(item.body);
    if (!title) {
      throw new Error(`${ERR_VALIDATION}: linear_create_issue title is empty after sanitization`);
    }

    if (isStagedMode(config)) {
      logStagedPreviewInfo(`Would create Linear issue "${title}"`);
      return { success: true, staged: true, title };
    }

    const data = await linearGraphQL(LINEAR_CREATE_ISSUE, {
      input: { teamId, title, description },
    });
    const payload = data?.issueCreate;
    if (payload?.success !== true || !payload.issue) {
      throw new Error(`${ERR_API}: Linear issueCreate did not return a successful issue`);
    }
    return {
      success: true,
      id: payload.issue.id,
      identifier: payload.issue.identifier,
      title: payload.issue.title,
    };
  };
}

module.exports = { LINEAR_CREATE_ISSUE, main };
