// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

const AGENT_OUTPUT_PATH = "/tmp/gh-aw/agent_output.json";
const SAFE_BRANCH_NAME_REGEX = /^[a-zA-Z0-9/_.-]+$/;

/**
 * @param {string} itemRepo
 * @param {string} workflowRepo
 * @returns {boolean}
 */
function isSameWorkflowRepo(itemRepo, workflowRepo) {
  if (!itemRepo) return true;
  if (!workflowRepo) return false;
  if (itemRepo === workflowRepo) return true;

  // Safe-output repo values may be a bare repo name and get qualified at runtime.
  // Match bare names against the repository suffix from owner/repo.
  if (!itemRepo.includes("/")) {
    return workflowRepo.endsWith(`/${itemRepo}`);
  }

  return false;
}

/**
 * @param {{ agentOutputPath?: string, workflowRepo?: string }} [opts]
 * @returns {string}
 */
function extractBaseBranchFromAgentOutput(opts = {}) {
  const agentOutputPath = opts.agentOutputPath || AGENT_OUTPUT_PATH;
  const workflowRepo = (opts.workflowRepo || process.env.GITHUB_REPOSITORY || "").trim();

  try {
    const data = JSON.parse(fs.readFileSync(agentOutputPath, "utf8"));
    const item = (data.items || []).find(i => {
      const itemRepo = (i.repo || "").trim();
      const sameRepo = isSameWorkflowRepo(itemRepo, workflowRepo);
      return (i.type === "create_pull_request" || i.type === "push_to_pull_request_branch") && i.base_branch && sameRepo;
    });
    return typeof item?.base_branch === "string" ? item.base_branch : "";
  } catch {
    return "";
  }
}

async function main() {
  const baseBranch = extractBaseBranchFromAgentOutput();
  if (!baseBranch) return;
  if (!SAFE_BRANCH_NAME_REGEX.test(baseBranch) || baseBranch.length > 255) return;
  core.setOutput("base-branch", baseBranch);
  core.info(`Extracted base branch from safe output: ${baseBranch}`);
}

module.exports = { extractBaseBranchFromAgentOutput, isSameWorkflowRepo, main };
