// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Returns true when the given file path should be excluded from the PR.
 * .github/workflows/*.yml files (including .lock.yml) are excluded because
 * the github-actions bot cannot modify workflow files directly.
 *
 * @param {string} file - Relative path of the file
 * @returns {boolean}
 */
function isExcludedWorkflowFile(file) {
  return /^\.github\/workflows\/[^/]+\.yml$/.test(file);
}

/**
 * Format a UTC Date as YYYY-MM-DD-HH-MM-SS for use in branch names.
 * Colons are not allowed in artifact filenames or branch names on some systems.
 *
 * @param {Date} date
 * @returns {string}
 */
function formatTimestamp(date) {
  /** @param {number} n */
  const pad = n => String(n).padStart(2, "0");
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}-${pad(date.getUTCHours())}-${pad(date.getUTCMinutes())}-${pad(date.getUTCSeconds())}`;
}

/**
 * Run 'gh aw update' or 'gh aw upgrade', then create a pull request with the
 * resulting non-workflow-YAML changes if any exist.
 *
 * .github/workflows/*.yml files (including compiled .lock.yml files) are
 * excluded from the PR because the github-actions bot cannot modify workflow
 * files directly. The PR body instructs reviewers to recompile lock files
 * after merging.
 *
 * Required environment variables:
 *   GH_TOKEN           - GitHub token for gh CLI auth and git push
 *   GH_AW_OPERATION    - 'update' or 'upgrade'
 *   GH_AW_CMD_PREFIX   - Command prefix: './gh-aw' (dev) or 'gh aw' (release)
 *
 * @returns {Promise<void>}
 */
async function main() {
  const operation = process.env.GH_AW_OPERATION;
  if (operation !== "update" && operation !== "upgrade") {
    core.info(`Skipping: operation '${operation}' is not 'update' or 'upgrade'`);
    return;
  }

  const isUpgrade = operation === "upgrade";
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);

  // Run gh aw update or gh aw upgrade
  const fullCmd = [bin, ...prefixArgs, operation].join(" ");
  core.info(`Running: ${fullCmd}`);
  await exec.exec(bin, [...prefixArgs, operation]);

  // Check for changed files
  const { stdout: statusOutput } = await exec.getExecOutput("git", ["status", "--porcelain"]);

  // Parse changed files - filter out .github/workflows/*.yml (including .lock.yml)
  // git status --porcelain format: "XY path" (X and Y are 1-char each at positions 0-1,
  // position 2 is a space, filename starts at position 3). Do NOT trim the full line
  // before slicing or the positional indices shift.
  const changedFiles = statusOutput
    .split("\n")
    .filter(line => line.length > 2)
    .map(line => {
      // "XY path" or "XY old -> new" for renames
      const path = line.slice(3).trim();
      return path.includes(" -> ") ? (path.split(" -> ").at(-1) ?? path) : path;
    })
    .filter(file => file.length > 0 && !isExcludedWorkflowFile(file));

  if (changedFiles.length === 0) {
    core.info("✓ No changes detected (excluding compiled workflow files) - nothing to create a PR for");
    return;
  }

  core.info(`Found ${changedFiles.length} changed file(s) to include in PR:`);
  for (const f of changedFiles) {
    core.info(`  ${f}`);
  }

  // Configure git identity
  await exec.exec("git", ["config", "user.email", "github-actions[bot]@users.noreply.github.com"]);
  await exec.exec("git", ["config", "user.name", "github-actions[bot]"]);

  // Create a new branch with a filesystem-safe timestamp (no colons)
  const branchName = `aw/${operation}-${formatTimestamp(new Date())}`;
  core.info(`Creating branch: ${branchName}`);
  await exec.exec("git", ["checkout", "-b", branchName]);

  // Stage only the non-yml files
  for (const file of changedFiles) {
    try {
      await exec.exec("git", ["add", "--", file]);
    } catch (error) {
      core.warning(`Failed to stage '${file}': ${getErrorMessage(error)}`);
    }
  }

  // Verify staged content
  const { stdout: stagedOutput } = await exec.getExecOutput("git", ["diff", "--cached", "--name-only"]);
  if (!stagedOutput.trim()) {
    core.info("✓ No staged changes after filtering workflow files - nothing to commit");
    return;
  }

  const stagedFiles = stagedOutput
    .split("\n")
    .map(f => f.trim())
    .filter(Boolean);

  // Commit the changes
  const commitMessage = isUpgrade ? "chore: upgrade agentic workflows" : "chore: update agentic workflows";
  await exec.exec("git", ["commit", "-m", commitMessage]);

  // Push to the new branch using a token-authenticated remote
  const owner = context.repo.owner;
  const repo = context.repo.repo;
  const token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN || "";
  const remoteUrl = `https://x-access-token:${token}@github.com/${owner}/${repo}.git`;

  try {
    await exec.exec("git", ["remote", "remove", "aw-push"]);
  } catch {
    // Remote doesn't exist yet - that's fine
  }
  await exec.exec("git", ["remote", "add", "aw-push", remoteUrl]);

  try {
    await exec.exec("git", ["push", "aw-push", branchName]);
  } finally {
    // Always clean up the temporary remote
    try {
      await exec.exec("git", ["remote", "remove", "aw-push"]);
    } catch {
      // Non-fatal
    }
  }

  // Build PR title and body
  const prTitle = isUpgrade ? "[aw] Upgrade available" : "[aw] Updates available";
  const fileList = stagedFiles.map(f => `- \`${f}\``).join("\n");
  const operationLabel = isUpgrade ? "Upgrade" : "Update";
  const prBody = `## Agentic Workflows ${operationLabel}

The \`gh aw ${operation}\` command was run automatically and produced the following changes:

${fileList}

### ⚠️ Lock Files Need Recompilation

The compiled workflow files (\`.github/workflows/*.yml\`) were **not included** in this PR because the \`github-actions\` bot cannot modify workflow files directly.

After merging this PR, **recompile the lock files** using one of these methods:

1. **Via @copilot**: Add a comment \`@copilot compile agentic workflows\` on this PR
2. **Via CLI**: Run \`gh aw compile --validate\` in your local checkout after merging
`;

  // Create the PR using gh CLI
  core.info(`Creating PR: "${prTitle}"`);
  const { stdout: prOutput } = await exec.getExecOutput("gh", ["pr", "create", "--title", prTitle, "--body", prBody, "--head", branchName], {
    env: { ...process.env, GH_TOKEN: token },
  });

  const prUrl = prOutput.trim();
  core.info(`✓ Created PR: ${prUrl}`);
  core.notice(`Created PR: ${prUrl}`);

  await core.summary
    .addHeading(prTitle, 2)
    .addRaw(`Pull request created: [${prUrl}](${prUrl})\n\n`)
    .addRaw(`**Changed files included in PR:**\n\n${fileList}\n\n`)
    .addRaw(`> **Note**: The \`.github/workflows/*.yml\` lock files were excluded. Recompile them after merging via \`@copilot compile agentic workflows\` or \`gh aw compile\`.`)
    .write();
}

module.exports = { main, isExcludedWorkflowFile, formatTimestamp };
