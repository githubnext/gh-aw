// @ts-check
/// <reference types="@actions/github-script" />
require("./shim.cjs");

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

const COMMENT_MEMORY_TAG = "gh-aw-comment-memory";
const COMMENT_MEMORY_DIR = "/tmp/gh-aw/comment-memory";
const PROMPT_PATH = "/tmp/gh-aw/aw-prompts/prompt.txt";
const PROMPT_START_MARKER = "<!-- gh-aw-comment-memory-prompt:start -->";
const PROMPT_END_MARKER = "<!-- gh-aw-comment-memory-prompt:end -->";
const MAX_SCAN_PAGES = 50;

function extractCommentMemoryEntries(commentBody) {
  if (!commentBody || typeof commentBody !== "string") {
    return [];
  }

  const pattern = new RegExp(`<${COMMENT_MEMORY_TAG}\\s+id="([A-Za-z0-9_-]+)">([\\s\\S]*?)<\\/${COMMENT_MEMORY_TAG}>`, "g");
  const matches = [];
  let match = pattern.exec(commentBody);
  while (match) {
    matches.push({
      memoryId: match[1],
      content: String(match[2] || "").trim(),
    });
    match = pattern.exec(commentBody);
  }
  return matches;
}

function loadSafeOutputsConfig() {
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || `${process.env.RUNNER_TEMP}/gh-aw/safeoutputs/config.json`;
  if (!fs.existsSync(configPath)) {
    return {};
  }
  try {
    return JSON.parse(fs.readFileSync(configPath, "utf8"));
  } catch (error) {
    core.warning(`comment_memory setup: failed to parse config at ${configPath}: ${getErrorMessage(error)}`);
    return {};
  }
}

function getCommentMemoryConfig(config) {
  return config["comment-memory"] || config.comment_memory || null;
}

function resolveTargetNumber(commentMemoryConfig) {
  const target = String(commentMemoryConfig?.target || "triggering").trim();
  if (target === "triggering") {
    return context.payload.issue?.number || context.payload.pull_request?.number || null;
  }

  if (target === "*") {
    return context.payload.issue?.number || context.payload.pull_request?.number || null;
  }

  const parsed = parseInt(target, 10);
  if (Number.isInteger(parsed) && parsed > 0) {
    return parsed;
  }
  return null;
}

function resolveTargetRepo(commentMemoryConfig) {
  const configuredRepo = String(commentMemoryConfig?.["target-repo"] || "").trim();
  const repoSlug = configuredRepo || `${context.repo.owner}/${context.repo.repo}`;
  const [owner, repo] = repoSlug.split("/");
  if (!owner || !repo) {
    return null;
  }
  return { owner, repo, slug: `${owner}/${repo}` };
}

async function collectCommentMemoryFiles(githubClient, commentMemoryConfig) {
  const targetNumber = resolveTargetNumber(commentMemoryConfig);
  if (!targetNumber) {
    core.info("comment_memory setup: no resolvable target issue/PR number, skipping");
    return [];
  }

  const targetRepo = resolveTargetRepo(commentMemoryConfig);
  if (!targetRepo) {
    core.warning("comment_memory setup: invalid target repo configuration");
    return [];
  }

  core.info(`comment_memory setup: loading managed comment memory from ${targetRepo.slug}#${targetNumber}`);
  const memoryMap = new Map();

  for (let page = 1; page <= MAX_SCAN_PAGES; page++) {
    const { data } = await githubClient.rest.issues.listComments({
      owner: targetRepo.owner,
      repo: targetRepo.repo,
      issue_number: targetNumber,
      per_page: 100,
      page,
    });

    if (!Array.isArray(data) || data.length === 0) {
      break;
    }

    for (const comment of data) {
      const entries = extractCommentMemoryEntries(comment.body);
      for (const entry of entries) {
        memoryMap.set(entry.memoryId, entry.content);
      }
    }

    if (data.length < 100) {
      break;
    }
  }

  fs.mkdirSync(COMMENT_MEMORY_DIR, { recursive: true });
  const writtenFiles = [];
  for (const [memoryId, content] of memoryMap.entries()) {
    const filePath = path.join(COMMENT_MEMORY_DIR, `${memoryId}.md`);
    fs.writeFileSync(filePath, `${content}\n`);
    writtenFiles.push(filePath);
  }

  core.info(`comment_memory setup: wrote ${writtenFiles.length} memory file(s) to ${COMMENT_MEMORY_DIR}`);
  return writtenFiles;
}

function injectCommentMemoryPrompt(filePaths) {
  if (!fs.existsSync(PROMPT_PATH)) {
    core.info(`comment_memory setup: prompt file missing at ${PROMPT_PATH}, skipping prompt injection`);
    return;
  }

  const fileList = filePaths.length > 0 ? filePaths.map(file => `- ${file}`).join("\n") : "- (none yet; create new *.md files here when needed)";
  const injectedBlock = `${PROMPT_START_MARKER}
<comment-memory-files>
Comment memory files are editable markdown files under \`${COMMENT_MEMORY_DIR}\`.
Update existing files or create new \`<memory-id>.md\` files as needed, then persist updates by calling the \`comment_memory\` safe-output tool with:
- \`memory_id\` = the filename without \`.md\`
- \`body\` = the file contents only (no XML wrapper, no footer text)
Available files:
${fileList}
</comment-memory-files>
${PROMPT_END_MARKER}`;

  let promptContent = fs.readFileSync(PROMPT_PATH, "utf8");
  const start = promptContent.indexOf(PROMPT_START_MARKER);
  const end = promptContent.indexOf(PROMPT_END_MARKER);
  if (start >= 0 && end > start) {
    const suffixStart = end + PROMPT_END_MARKER.length;
    promptContent = `${promptContent.slice(0, start).trimEnd()}\n\n${injectedBlock}\n${promptContent.slice(suffixStart).trimStart()}`;
  } else {
    promptContent = `${promptContent.trimEnd()}\n\n${injectedBlock}\n`;
  }
  fs.writeFileSync(PROMPT_PATH, promptContent);
  core.info("comment_memory setup: injected comment-memory prompt guidance");
}

async function main() {
  const safeOutputsConfig = loadSafeOutputsConfig();
  const commentMemoryConfig = getCommentMemoryConfig(safeOutputsConfig);
  if (!commentMemoryConfig) {
    core.debug("comment_memory setup: comment-memory is not configured");
    return;
  }

  try {
    const files = await collectCommentMemoryFiles(github, commentMemoryConfig);
    injectCommentMemoryPrompt(files);
  } catch (error) {
    core.warning(`comment_memory setup: failed to prepare comment-memory files: ${getErrorMessage(error)}`);
  }
}

module.exports = {
  main,
  extractCommentMemoryEntries,
  resolveTargetNumber,
};
