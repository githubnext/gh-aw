// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_SYSTEM } = require("./error_codes.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

/**
 * Generate a deterministic pastel hex color string from a label name.
 * Produces colors in the pastel range (128–191 per channel) for readability.
 *
 * @param {string} name
 * @returns {string} Six-character hex color (no leading #)
 */
function deterministicLabelColor(name) {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
  }
  // Map to pastel range: 128–191 per channel
  const r = 128 + (hash & 0x3f);
  const g = 128 + ((hash >> 6) & 0x3f);
  const b = 128 + ((hash >> 12) & 0x3f);
  return ((r << 16) | (g << 8) | b).toString(16).padStart(6, "0");
}

/**
 * @param {unknown} value
 * @returns {value is string}
 */
function isNonEmptyString(value) {
  return typeof value === "string" && value.trim().length > 0;
}

/**
 * Compile all agentic workflows, collect the labels referenced in safe-outputs
 * configurations, and create any labels that are missing from the repository.
 *
 * Required environment variables:
 *   GH_AW_CMD_PREFIX  - Command prefix: './gh-aw' (dev) or 'gh aw' (release)
 *
 * @returns {Promise<void>}
 */
async function main() {
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);

  // Run compile --json --no-emit to collect labels without writing lock files.
  // Use ignoreReturnCode because compile exits non-zero when some workflows have errors,
  // but still produces valid JSON output for all (valid and invalid) workflows.
  const compileArgs = [...prefixArgs, "compile", "--json", "--no-emit"];
  core.info(`Running: ${bin} ${compileArgs.join(" ")}`);

  let compileOutput;
  try {
    const result = await exec.getExecOutput(bin, compileArgs, { ignoreReturnCode: true });
    compileOutput = result.stdout;
    // Only treat as a fatal error when the exit is non-zero AND there is no stdout at all.
    // A non-zero exit with JSON on stdout means some workflows failed validation but we
    // still have label data from the successfully-parsed ones — continue processing.
    if (result.exitCode !== 0 && !compileOutput.trim()) {
      throw new Error(`${ERR_SYSTEM}: compile exited with code ${result.exitCode}: ${result.stderr}`);
    }
  } catch (err) {
    core.setFailed(`Failed to run compile: ${getErrorMessage(err)}`);
    return;
  }

  // Parse JSON output
  let validationResults;
  try {
    validationResults = JSON.parse(compileOutput);
  } catch (err) {
    core.setFailed(`Failed to parse compile JSON output: ${getErrorMessage(err)}`);
    return;
  }

  // Collect all unique labels across all workflows
  const allLabels = new Set(
    validationResults.flatMap(
      /** @param {{ labels?: unknown[] }} result */
      result => {
        if (!Array.isArray(result.labels)) return [];
        return result.labels.filter(isNonEmptyString).map(l => l.trim());
      }
    )
  );

  if (allLabels.size === 0) {
    core.info("No labels found in safe-outputs configurations — nothing to create");
    return;
  }

  core.info(`Found ${allLabels.size} unique label(s) in safe-outputs: ${[...allLabels].join(", ")}`);

  // Fetch all existing labels from the repository.
  // When GH_AW_TARGET_REPO_SLUG is set (SideRepoOps pattern), create labels in that
  // repository instead of the execution context repository.
  const { owner, repo } = resolveExecutionOwnerRepo();
  core.info(`Operating on repository: ${owner}/${repo}`);
  let existingLabels;
  try {
    existingLabels = await github.paginate(github.rest.issues.listLabelsForRepo, {
      owner,
      repo,
      per_page: 100,
    });
  } catch (err) {
    core.setFailed(`Failed to list repository labels: ${getErrorMessage(err)}`);
    return;
  }

  const existingLabelNames = new Set(existingLabels.map(l => l.name.toLowerCase()));

  // Create missing labels
  let created = 0;
  let skipped = 0;

  for (const labelName of allLabels) {
    if (existingLabelNames.has(labelName.toLowerCase())) {
      core.info(`ℹ️  Label already exists: ${labelName}`);
      skipped++;
    } else {
      try {
        await github.rest.issues.createLabel({
          owner,
          repo,
          name: labelName,
          color: deterministicLabelColor(labelName),
          description: "",
        });
        core.info(`✅ Created label: ${labelName}`);
        created++;
      } catch (err) {
        // 422 means label already exists (race condition) — treat as success
        if (err && typeof err === "object" && /** @type {any} */ err.status === 422) {
          core.info(`ℹ️  Label already exists (concurrent): ${labelName}`);
          skipped++;
        } else {
          core.warning(`Failed to create label '${labelName}': ${getErrorMessage(err)}`);
        }
      }
    }
  }

  core.info(`Done: ${created} label(s) created, ${skipped} already existed`);
}

module.exports = { main, deterministicLabelColor };
