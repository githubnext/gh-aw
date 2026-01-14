#!/usr/bin/env node

/**
 * Changeset CLI - A minimalistic implementation for managing version releases
 * Inspired by @changesets/cli
 *
 * Usage:
 *   node changeset.js version    - Preview next version from changesets
 *   node changeset.js release    - Create release and update CHANGELOG
 *   GH_AW_CURRENT_VERSION=v1.2.3 node changeset.js release to force current version
 */

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

// ANSI color codes for terminal output
const colors = {
  info: "\x1b[36m", // Cyan
  success: "\x1b[32m", // Green
  error: "\x1b[31m", // Red
  reset: "\x1b[0m",
};

function formatInfoMessage(msg) {
  return `${colors.info}ℹ ${msg}${colors.reset}`;
}

function formatSuccessMessage(msg) {
  return `${colors.success}✓ ${msg}${colors.reset}`;
}

function formatErrorMessage(msg) {
  return `${colors.error}✗ ${msg}${colors.reset}`;
}

/**
 * Parse a changeset markdown file
 * @param {string} filePath - Path to the changeset file
 * @returns {Object} Parsed changeset entry
 */
function parseChangesetFile(filePath) {
  const content = fs.readFileSync(filePath, "utf8");
  const lines = content.split("\n");

  // Check for frontmatter
  if (lines[0] !== "---") {
    throw new Error(`Invalid changeset format in ${filePath}: missing frontmatter`);
  }

  // Find end of frontmatter
  let frontmatterEnd = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === "---") {
      frontmatterEnd = i;
      break;
    }
  }

  if (frontmatterEnd === -1) {
    throw new Error(`Invalid changeset format in ${filePath}: unclosed frontmatter`);
  }

  // Parse frontmatter (simple YAML parsing for our use case)
  const frontmatterLines = lines.slice(1, frontmatterEnd);
  let bumpType = null;

  for (const line of frontmatterLines) {
    const match = line.match(/^"(githubnext\/)?gh-aw":\s*(patch|minor|major)/);
    if (match) {
      bumpType = match[2];
      break;
    }
  }

  if (!bumpType) {
    throw new Error(`Invalid changeset format in ${filePath}: missing or invalid 'gh-aw' field`);
  }

  // Get body content (everything after frontmatter)
  const bodyContent = lines
    .slice(frontmatterEnd + 1)
    .join("\n")
    .trim();

  // Check for codemod section (## Codemod)
  const codemodMatch = bodyContent.match(/^([\s\S]*?)(?:^|\n)## Codemod\s*\n([\s\S]*)$/m);

  let description = bodyContent;
  let codemod = null;

  if (codemodMatch) {
    // Split into description (before codemod) and codemod section
    description = codemodMatch[1].trim();
    codemod = codemodMatch[2].trim();
  }

  return {
    package: "gh-aw",
    bumpType: bumpType,
    description: description,
    codemod: codemod,
    filePath: filePath,
  };
}

/**
 * Read all changeset files from .changeset/ directory
 * @returns {Array} Array of changeset entries
 */
function readChangesets() {
  const changesetDir = ".changeset";

  // Try to read directory without checking existence first (avoids TOCTOU)
  let entries;
  try {
    entries = fs.readdirSync(changesetDir);
  } catch (error) {
    if (error.code === "ENOENT") {
      throw new Error("Changeset directory not found: .changeset/");
    }
    throw error;
  }

  const changesets = [];

  for (const entry of entries) {
    if (!entry.endsWith(".md")) {
      continue;
    }

    const filePath = path.join(changesetDir, entry);
    try {
      const changeset = parseChangesetFile(filePath);
      changesets.push(changeset);
    } catch (error) {
      console.error(formatErrorMessage(`Skipping ${entry}: ${error.message}`));
    }
  }

  return changesets;
}

/**
 * Determine the highest priority version bump from changesets
 * @param {Array} changesets - Array of changeset entries
 * @returns {string} Version bump type (major, minor, or patch)
 */
function determineVersionBump(changesets) {
  if (changesets.length === 0) {
    return "";
  }

  // Priority: major > minor > patch
  let hasMajor = false;
  let hasMinor = false;
  let hasPatch = false;

  for (const cs of changesets) {
    switch (cs.bumpType) {
      case "major":
        hasMajor = true;
        break;
      case "minor":
        hasMinor = true;
        break;
      case "patch":
        hasPatch = true;
        break;
    }
  }

  if (hasMajor) return "major";
  if (hasMinor) return "minor";
  if (hasPatch) return "patch";

  return "";
}

/**
 * Get current version from git tags
 * @returns {Object} Version info {major, minor, patch}
 */
function getCurrentVersion() {
  try {
    const output = process.env.GH_AW_CURRENT_VERSION || execSync("git describe --tags --abbrev=0", { encoding: "utf8" });
    const versionStr = output.trim().replace(/^v/, "");
    const parts = versionStr.split(".");

    if (parts.length !== 3) {
      throw new Error(`Invalid version format: ${versionStr}`);
    }

    return {
      major: parseInt(parts[0], 10),
      minor: parseInt(parts[1], 10),
      patch: parseInt(parts[2], 10),
    };
  } catch (error) {
    // No tags exist, start from v0.0.0
    return { major: 0, minor: 0, patch: 0 };
  }
}

/**
 * Bump version based on bump type
 * @param {Object} current - Current version
 * @param {string} bumpType - Type of bump (major, minor, patch)
 * @returns {Object} New version
 */
function bumpVersion(current, bumpType) {
  const next = {
    major: current.major,
    minor: current.minor,
    patch: current.patch,
  };

  switch (bumpType) {
    case "major":
      next.major++;
      next.minor = 0;
      next.patch = 0;
      break;
    case "minor":
      next.minor++;
      next.patch = 0;
      break;
    case "patch":
      next.patch++;
      break;
  }

  return next;
}

/**
 * Format version as string
 * @param {Object} version - Version object
 * @returns {string} Formatted version string
 */
function formatVersion(version) {
  return `v${version.major}.${version.minor}.${version.patch}`;
}

/**
 * Extract first non-empty line from text
 * @param {string} text - Text to extract from
 * @returns {string} First line
 */
function extractFirstLine(text) {
  const lines = text.split("\n");
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed !== "") {
      return trimmed;
    }
  }
  return text;
}

/**
 * Extract and consolidate all codemod entries from changesets
 * @param {Array} changesets - Array of changeset entries
 * @returns {string|null} Consolidated codemod prompt or null if no codemods
 */
function extractCodemods(changesets) {
  const codemodEntries = changesets.filter(cs => cs.codemod);

  if (codemodEntries.length === 0) {
    return null;
  }

  let prompt = "The following breaking changes require code updates:\n\n";

  for (const cs of codemodEntries) {
    // Add the description as context
    const firstLine = extractFirstLine(cs.description);
    prompt += `### ${firstLine}\n\n`;
    prompt += cs.codemod + "\n\n";
  }

  return prompt.trim();
}

/**
 * Format changeset body for changelog entry
 * Converts the first line to a header 4 and includes the rest of the body
 * @param {string} text - Changeset description text
 * @returns {string} Formatted text with first line as h4
 */
function formatChangesetBody(text) {
  const lines = text.split("\n");

  // Find first non-empty line for header
  let firstLineIndex = -1;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].trim() !== "") {
      firstLineIndex = i;
      break;
    }
  }

  if (firstLineIndex === -1) {
    return text + "\n\n";
  }

  // Format first line as header 4
  const firstLine = lines[firstLineIndex].trim();
  const remainingLines = lines.slice(firstLineIndex + 1);

  // Build formatted output
  let formatted = `#### ${firstLine}\n\n`;

  // Add remaining content if present
  const remainingText = remainingLines.join("\n").trim();
  if (remainingText) {
    formatted += remainingText + "\n\n";
  }

  return formatted;
}

/**
 * Check if git working tree is clean
 * @returns {boolean} True if tree is clean
 */
function isGitTreeClean() {
  try {
    const output = execSync("git status --porcelain --untracked-files=no", { encoding: "utf8" });
    return output.trim() === "";
  } catch (error) {
    throw new Error("Failed to check git status. Are you in a git repository?");
  }
}

/**
 * Get current git branch name
 * @returns {string} Branch name
 */
function getCurrentBranch() {
  try {
    const output = execSync("git branch --show-current", { encoding: "utf8" });
    return output.trim();
  } catch (error) {
    throw new Error("Failed to get current branch. Are you in a git repository?");
  }
}

/**
 * Check git prerequisites for release
 */
function checkGitPrerequisites() {
  // Check if on main branch
  const currentBranch = getCurrentBranch();
  if (currentBranch !== "main") {
    throw new Error(`Must be on 'main' branch to create a release (currently on '${currentBranch}')`);
  }

  // Check if working tree is clean
  if (!isGitTreeClean()) {
    throw new Error("Working tree is not clean. Commit or stash your changes before creating a release.");
  }
}

/**
 * Prompt for user confirmation
 * @param {string} message - Message to display
 * @returns {boolean} True if user confirmed
 */
function promptConfirmation(message) {
  const readline = require("readline");
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  return new Promise(resolve => {
    rl.question(`${message} (y/N): `, answer => {
      rl.close();
      const confirmed = answer.toLowerCase() === "y" || answer.toLowerCase() === "yes";
      resolve(confirmed);
    });
  });
}

/**
 * Update CHANGELOG.md with new version and changes
 * @param {string} version - Version string
 * @param {Array} changesets - Array of changesets
 * @param {boolean} dryRun - If true, preview changes without writing
 * @returns {string} The new changelog entry or full content
 */
function updateChangelog(version, changesets, dryRun = false) {
  const changelogPath = "CHANGELOG.md";

  // Read existing changelog or create header
  // Use file descriptor to avoid TOCTOU vulnerability
  let existingContent = "";
  let fd;
  try {
    // Try to open existing file for reading
    fd = fs.openSync(changelogPath, fs.constants.O_RDONLY);
    existingContent = fs.readFileSync(fd, "utf8");
    fs.closeSync(fd);
  } catch (error) {
    // File doesn't exist or can't be read, use default header
    if (error.code === "ENOENT") {
      existingContent = "# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n";
    } else {
      throw error;
    }
  }

  // Build new entry
  const date = new Date().toISOString().split("T")[0];
  let newEntry = `## ${version} - ${date}\n\n`;

  // If no changesets, add a minimal entry
  if (changesets.length === 0) {
    newEntry += "Maintenance release with dependency updates and minor improvements.\n\n";
  } else {
    // Group changes by type
    const majorChanges = changesets.filter(cs => cs.bumpType === "major");
    const minorChanges = changesets.filter(cs => cs.bumpType === "minor");
    const patchChanges = changesets.filter(cs => cs.bumpType === "patch");

    // Write changes by category
    if (majorChanges.length > 0) {
      newEntry += "### Breaking Changes\n\n";
      for (const cs of majorChanges) {
        newEntry += formatChangesetBody(cs.description);
      }
      newEntry += "\n";
    }

    if (minorChanges.length > 0) {
      newEntry += "### Features\n\n";
      for (const cs of minorChanges) {
        newEntry += formatChangesetBody(cs.description);
      }
      newEntry += "\n";
    }

    if (patchChanges.length > 0) {
      newEntry += "### Bug Fixes\n\n";
      for (const cs of patchChanges) {
        newEntry += formatChangesetBody(cs.description);
      }
      newEntry += "\n";
    }

    // Add consolidated codemods as a markdown code region if any exist
    const codemodPrompt = extractCodemods(changesets);
    if (codemodPrompt) {
      newEntry += "### Migration Guide\n\n";
      newEntry += "`````markdown\n";
      newEntry += codemodPrompt + "\n";
      newEntry += "`````\n\n";
    }
  }

  // Insert new entry after header
  const headerEnd = existingContent.indexOf("\n## ");
  let updatedContent;
  if (headerEnd === -1) {
    // No existing entries, append to end
    updatedContent = existingContent + newEntry;
  } else {
    // Insert before first existing entry
    updatedContent = existingContent.substring(0, headerEnd + 1) + newEntry + existingContent.substring(headerEnd + 1);
  }

  if (dryRun) {
    // Return the new entry for preview
    return newEntry;
  }

  // Write updated changelog using file descriptor to avoid TOCTOU vulnerability
  try {
    fd = fs.openSync(changelogPath, fs.constants.O_WRONLY | fs.constants.O_CREAT | fs.constants.O_TRUNC, 0o644);
    fs.writeFileSync(fd, updatedContent, "utf8");
    fs.closeSync(fd);
  } catch (error) {
    throw new Error(`Failed to write CHANGELOG.md: ${error.message}`);
  }
  return newEntry;
}

/**
 * Delete changeset files
 * @param {Array} changesets - Array of changesets to delete
 * @param {boolean} dryRun - If true, preview what would be deleted
 */
function deleteChangesetFiles(changesets, dryRun = false) {
  if (dryRun) {
    // Just return the list of files that would be deleted
    return changesets.map(cs => cs.filePath);
  }

  for (const cs of changesets) {
    fs.unlinkSync(cs.filePath);
  }
  return [];
}

/**
 * Run the version command
 */
function runVersion() {
  const changesets = readChangesets();

  if (changesets.length === 0) {
    console.log(formatInfoMessage("No changesets found"));
    return;
  }

  const bumpType = determineVersionBump(changesets);
  const currentVersion = getCurrentVersion();
  const nextVersion = bumpVersion(currentVersion, bumpType);
  const versionString = formatVersion(nextVersion);

  console.log(formatInfoMessage(`Current version: ${formatVersion(currentVersion)}`));
  console.log(formatInfoMessage(`Bump type: ${bumpType}`));
  console.log(formatInfoMessage(`Next version: ${versionString}`));
  console.log(formatInfoMessage("\nChanges:"));

  for (const cs of changesets) {
    console.log(`  [${cs.bumpType}] ${extractFirstLine(cs.description)}`);
  }

  // Generate changelog preview (never write in version command)
  const changelogEntry = updateChangelog(versionString, changesets, true);

  console.log("");
  console.log(formatInfoMessage("Would add to CHANGELOG.md:"));
  console.log("---");
  console.log(changelogEntry);
  console.log("---");

  // Extract and display consolidated codemods
  const codemodPrompt = extractCodemods(changesets);
  if (codemodPrompt) {
    console.log("");
    console.log(formatInfoMessage("Consolidated Codemod Instructions (copy for Copilot agent task):"));
    console.log("---");
    console.log(codemodPrompt);
    console.log("---");
  }
}

/**
 * Run the release command
 * @param {string} releaseType - Optional release type (patch, minor, major)
 * @param {boolean} skipConfirmation - If true, skip confirmation prompt
 */
async function runRelease(releaseType, skipConfirmation = false) {
  // Check git prerequisites (clean tree, main branch)
  checkGitPrerequisites();

  const changesets = readChangesets();

  if (changesets.length === 0) {
    // If no changesets exist, default to patch release
    if (!releaseType) {
      releaseType = "patch";
      console.log(formatInfoMessage("No changesets found - defaulting to patch release"));
    } else {
      console.log(formatInfoMessage("No changesets found - creating release without changeset entries"));
    }
  }

  // Determine bump type
  let bumpType = releaseType;
  if (!bumpType) {
    bumpType = determineVersionBump(changesets);
  }

  // Safety check for major releases
  if (bumpType === "major" && !releaseType) {
    console.error(formatErrorMessage("Major releases must be explicitly specified with 'node changeset.js release major' for safety"));
    process.exit(1);
  }

  const currentVersion = getCurrentVersion();
  const nextVersion = bumpVersion(currentVersion, bumpType);
  const versionString = formatVersion(nextVersion);

  console.log(formatInfoMessage(`Current version: ${formatVersion(currentVersion)}`));
  console.log(formatInfoMessage(`Bump type: ${bumpType}`));
  console.log(formatInfoMessage(`Next version: ${versionString}`));
  console.log(formatInfoMessage(`Creating ${bumpType} release: ${versionString}`));

  // Show what will be included in the release
  if (changesets.length > 0) {
    console.log("");
    console.log(formatInfoMessage("Changes to be included:"));
    for (const cs of changesets) {
      console.log(`  [${cs.bumpType}] ${extractFirstLine(cs.description)}`);
    }
  }

  // Ask for confirmation before making any changes (unless --yes flag is used)
  if (!skipConfirmation) {
    console.log("");
    const confirmed = await promptConfirmation(formatInfoMessage("Proceed with creating the release (update files, commit, tag, and push)?"));

    if (!confirmed) {
      console.log(formatInfoMessage("Release cancelled. No changes have been made."));
      return;
    }
  } else {
    console.log("");
    console.log(formatInfoMessage("Skipping confirmation (--yes flag provided)"));
  }

  // Update changelog
  updateChangelog(versionString, changesets, false);

  // Delete changeset files only if there are any
  if (changesets.length > 0) {
    deleteChangesetFiles(changesets, false);
  }

  console.log("");
  console.log(formatSuccessMessage("Updated CHANGELOG.md"));
  if (changesets.length > 0) {
    console.log(formatSuccessMessage(`Removed ${changesets.length} changeset file(s)`));
  }

  // Extract and display consolidated codemods if any
  const codemodPrompt = extractCodemods(changesets);
  if (codemodPrompt) {
    console.log("");
    console.log(formatInfoMessage("Consolidated Codemod Instructions (copy for Copilot agent task):"));
    console.log("---");
    console.log(codemodPrompt);
    console.log("---");
  }

  // Execute git operations
  console.log("");
  console.log(formatInfoMessage("Executing git operations..."));

  try {
    // Stage changes
    console.log(formatInfoMessage("Staging changes..."));
    if (changesets.length > 0) {
      execSync("git add CHANGELOG.md .changeset/", { encoding: "utf8" });
    } else {
      execSync("git add CHANGELOG.md", { encoding: "utf8" });
    }

    // Commit changes
    console.log(formatInfoMessage("Committing changes..."));
    execSync(`git commit -m "Release ${versionString}"`, { encoding: "utf8" });

    // Create tag
    console.log(formatInfoMessage("Creating tag..."));
    execSync(`git tag -a ${versionString} -m "Release ${versionString}"`, { encoding: "utf8" });

    // Push commit to remote
    console.log(formatInfoMessage("Pushing commit..."));
    execSync("git push", { encoding: "utf8" });

    // Push tag
    console.log(formatInfoMessage("Pushing tag..."));
    execSync(`git push origin ${versionString}`, { encoding: "utf8" });

    console.log("");
    console.log(formatSuccessMessage(`Successfully released ${versionString}`));
    console.log(formatSuccessMessage("Commit and tag pushed to remote"));
  } catch (error) {
    console.log("");
    console.error(formatErrorMessage("Git operation failed: " + error.message));
    console.log("");
    console.log(formatInfoMessage("You can complete the release manually with:"));
    if (changesets.length > 0) {
      console.log(`  git add CHANGELOG.md .changeset/`);
    } else {
      console.log(`  git add CHANGELOG.md`);
    }
    console.log(`  git commit -m "Release ${versionString}"`);
    console.log(`  git tag -a ${versionString} -m "Release ${versionString}"`);
    console.log(`  git push`);
    console.log(`  git push origin ${versionString}`);
    process.exit(1);
  }
}

/**
 * Show help message
 */
function showHelp() {
  console.log("Changeset CLI - Manage version releases");
  console.log("");
  console.log("Usage:");
  console.log("  node scripts/changeset.js version      - Preview next version from changesets");
  console.log("  node scripts/changeset.js release [type] [--yes] - Create release and update CHANGELOG");
  console.log("");
  console.log("Release types: patch, minor, major");
  console.log("");
  console.log("Flags:");
  console.log("  --yes, -y    Skip confirmation prompt and proceed automatically");
  console.log("");
  console.log("Examples:");
  console.log("  node scripts/changeset.js version");
  console.log("  node scripts/changeset.js release");
  console.log("  node scripts/changeset.js release patch");
  console.log("  node scripts/changeset.js release minor");
  console.log("  node scripts/changeset.js release major");
  console.log("  node scripts/changeset.js release --yes");
  console.log("  node scripts/changeset.js release patch --yes");
}

// Main entry point
async function main() {
  const args = process.argv.slice(2);

  if (args.length === 0 || args[0] === "--help" || args[0] === "-h") {
    showHelp();
    return;
  }

  const command = args[0];

  try {
    switch (command) {
      case "version":
        runVersion();
        break;
      case "release":
        // Parse release type and flags
        let releaseType = null;
        let skipConfirmation = false;

        for (let i = 1; i < args.length; i++) {
          const arg = args[i];
          if (arg === "--yes" || arg === "-y") {
            skipConfirmation = true;
          } else if (!releaseType && ["patch", "minor", "major"].includes(arg)) {
            releaseType = arg;
          }
        }

        await runRelease(releaseType, skipConfirmation);
        break;
      default:
        console.error(formatErrorMessage(`Unknown command: ${command}`));
        console.log("");
        showHelp();
        process.exit(1);
    }
  } catch (error) {
    console.error(formatErrorMessage(error.message));
    process.exit(1);
  }
}

main();
