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

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// ANSI color codes for terminal output
const colors = {
  info: '\x1b[36m',    // Cyan
  success: '\x1b[32m', // Green
  error: '\x1b[31m',   // Red
  reset: '\x1b[0m'
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
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');
  
  // Check for frontmatter
  if (lines[0] !== '---') {
    throw new Error(`Invalid changeset format in ${filePath}: missing frontmatter`);
  }
  
  // Find end of frontmatter
  let frontmatterEnd = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i] === '---') {
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
  
  // Get description (everything after frontmatter)
  const description = lines.slice(frontmatterEnd + 1).join('\n').trim();
  
  return {
    package: 'gh-aw',
    bumpType: bumpType,
    description: description,
    filePath: filePath
  };
}

/**
 * Read all changeset files from .changeset/ directory
 * @returns {Array} Array of changeset entries
 */
function readChangesets() {
  const changesetDir = '.changeset';
  
  if (!fs.existsSync(changesetDir)) {
    throw new Error('Changeset directory not found: .changeset/');
  }
  
  const entries = fs.readdirSync(changesetDir);
  const changesets = [];
  
  for (const entry of entries) {
    if (!entry.endsWith('.md')) {
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
    return '';
  }
  
  // Priority: major > minor > patch
  let hasMajor = false;
  let hasMinor = false;
  let hasPatch = false;
  
  for (const cs of changesets) {
    switch (cs.bumpType) {
      case 'major':
        hasMajor = true;
        break;
      case 'minor':
        hasMinor = true;
        break;
      case 'patch':
        hasPatch = true;
        break;
    }
  }
  
  if (hasMajor) return 'major';
  if (hasMinor) return 'minor';
  if (hasPatch) return 'patch';
  
  return '';
}

/**
 * Get current version from git tags
 * @returns {Object} Version info {major, minor, patch}
 */
function getCurrentVersion() {
  try {
    const output = process.env.GH_AW_CURRENT_VERSION || execSync('git describe --tags --abbrev=0', { encoding: 'utf8' });
    const versionStr = output.trim().replace(/^v/, '');
    const parts = versionStr.split('.');
    
    if (parts.length !== 3) {
      throw new Error(`Invalid version format: ${versionStr}`);
    }
    
    return {
      major: parseInt(parts[0], 10),
      minor: parseInt(parts[1], 10),
      patch: parseInt(parts[2], 10)
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
    patch: current.patch
  };
  
  switch (bumpType) {
    case 'major':
      next.major++;
      next.minor = 0;
      next.patch = 0;
      break;
    case 'minor':
      next.minor++;
      next.patch = 0;
      break;
    case 'patch':
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
  const lines = text.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed !== '') {
      return trimmed;
    }
  }
  return text;
}

/**
 * Check if git working tree is clean
 * @returns {boolean} True if tree is clean
 */
function isGitTreeClean() {
  try {
    const output = execSync('git status --porcelain', { encoding: 'utf8' });
    return output.trim() === '';
  } catch (error) {
    throw new Error('Failed to check git status. Are you in a git repository?');
  }
}

/**
 * Get current git branch name
 * @returns {string} Branch name
 */
function getCurrentBranch() {
  try {
    const output = execSync('git branch --show-current', { encoding: 'utf8' });
    return output.trim();
  } catch (error) {
    throw new Error('Failed to get current branch. Are you in a git repository?');
  }
}

/**
 * Check git prerequisites for release
 */
function checkGitPrerequisites() {
  // Check if on main branch
  const currentBranch = getCurrentBranch();
  if (currentBranch !== 'main') {
    throw new Error(`Must be on 'main' branch to create a release (currently on '${currentBranch}')`);
  }
  
  // Check if working tree is clean
  if (!isGitTreeClean()) {
    throw new Error('Working tree is not clean. Commit or stash your changes before creating a release.');
  }
}

/**
 * Update CHANGELOG.md with new version and changes
 * @param {string} version - Version string
 * @param {Array} changesets - Array of changesets
 * @param {boolean} dryRun - If true, preview changes without writing
 * @returns {string} The new changelog entry or full content
 */
function updateChangelog(version, changesets, dryRun = false) {
  const changelogPath = 'CHANGELOG.md';
  
  // Read existing changelog or create header
  let existingContent = '';
  if (fs.existsSync(changelogPath)) {
    existingContent = fs.readFileSync(changelogPath, 'utf8');
  } else {
    existingContent = '# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n';
  }
  
  // Build new entry
  const date = new Date().toISOString().split('T')[0];
  let newEntry = `## ${version} - ${date}\n\n`;
  
  // Group changes by type
  const majorChanges = changesets.filter(cs => cs.bumpType === 'major');
  const minorChanges = changesets.filter(cs => cs.bumpType === 'minor');
  const patchChanges = changesets.filter(cs => cs.bumpType === 'patch');
  
  // Write changes by category
  if (majorChanges.length > 0) {
    newEntry += '### Breaking Changes\n\n';
    for (const cs of majorChanges) {
      newEntry += `- ${extractFirstLine(cs.description)}\n`;
    }
    newEntry += '\n';
  }
  
  if (minorChanges.length > 0) {
    newEntry += '### Features\n\n';
    for (const cs of minorChanges) {
      newEntry += `- ${extractFirstLine(cs.description)}\n`;
    }
    newEntry += '\n';
  }
  
  if (patchChanges.length > 0) {
    newEntry += '### Bug Fixes\n\n';
    for (const cs of patchChanges) {
      newEntry += `- ${extractFirstLine(cs.description)}\n`;
    }
    newEntry += '\n';
  }
  
  // Insert new entry after header
  const headerEnd = existingContent.indexOf('\n## ');
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
  
  // Write updated changelog
  fs.writeFileSync(changelogPath, updatedContent, 'utf8');
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
    console.log(formatInfoMessage('No changesets found'));
    return;
  }
  
  const bumpType = determineVersionBump(changesets);
  const currentVersion = getCurrentVersion();
  const nextVersion = bumpVersion(currentVersion, bumpType);
  const versionString = formatVersion(nextVersion);
  
  console.log(formatInfoMessage(`Current version: ${formatVersion(currentVersion)}`));
  console.log(formatInfoMessage(`Bump type: ${bumpType}`));
  console.log(formatInfoMessage(`Next version: ${versionString}`));
  console.log(formatInfoMessage('\nChanges:'));
  
  for (const cs of changesets) {
    console.log(`  [${cs.bumpType}] ${extractFirstLine(cs.description)}`);
  }
  
  // Generate changelog preview (never write in version command)
  const changelogEntry = updateChangelog(versionString, changesets, true);
  
  console.log('');
  console.log(formatInfoMessage('Would add to CHANGELOG.md:'));
  console.log('---');
  console.log(changelogEntry);
  console.log('---');
}

/**
 * Run the release command
 * @param {string} releaseType - Optional release type (patch, minor, major)
 */
function runRelease(releaseType) {
  // Check git prerequisites (clean tree, main branch)
  checkGitPrerequisites();
  
  const changesets = readChangesets();
  
  if (changesets.length === 0) {
    console.error(formatErrorMessage('No changesets found to release'));
    process.exit(1);
  }
  
  // Determine bump type
  let bumpType = releaseType;
  if (!bumpType) {
    bumpType = determineVersionBump(changesets);
  }
  
  // Safety check for major releases
  if (bumpType === 'major' && !releaseType) {
    console.error(formatErrorMessage("Major releases must be explicitly specified with 'node changeset.js release major' for safety"));
    process.exit(1);
  }
  
  const currentVersion = getCurrentVersion();
  const nextVersion = bumpVersion(currentVersion, bumpType);
  const versionString = formatVersion(nextVersion);
  
  console.log(formatInfoMessage(`Creating ${bumpType} release: ${versionString}`));
  
  // Update changelog
  updateChangelog(versionString, changesets, false);
  
  // Delete changeset files
  deleteChangesetFiles(changesets, false);
  
  console.log('');
  console.log(formatSuccessMessage('Updated CHANGELOG.md'));
  console.log(formatSuccessMessage(`Removed ${changesets.length} changeset file(s)`));
  
  // Execute git operations automatically
  console.log('');
  console.log(formatInfoMessage('Executing git operations...'));
  
  try {
    // Stage changes
    console.log(formatInfoMessage('Staging changes...'));
    execSync('git add CHANGELOG.md .changeset/', { encoding: 'utf8' });
    
    // Commit changes
    console.log(formatInfoMessage('Committing changes...'));
    execSync(`git commit -m "Release ${versionString}"`, { encoding: 'utf8' });
    
    // Create tag
    console.log(formatInfoMessage('Creating tag...'));
    execSync(`git tag -a ${versionString} -m "Release ${versionString}"`, { encoding: 'utf8' });
    
    // Push commit to remote
    console.log(formatInfoMessage('Pushing commit...'));
    execSync('git push', { encoding: 'utf8' });
    
    // Push tag
    console.log(formatInfoMessage('Pushing tag...'));
    execSync(`git push origin ${versionString}`, { encoding: 'utf8' });
    
    console.log('');
    console.log(formatSuccessMessage(`Successfully released ${versionString}`));
    console.log(formatSuccessMessage('Commit and tag pushed to remote'));
  } catch (error) {
    console.log('');
    console.error(formatErrorMessage('Git operation failed: ' + error.message));
    console.log('');
    console.log(formatInfoMessage('You can complete the release manually with:'));
    console.log(`  git add CHANGELOG.md .changeset/`);
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
  console.log('Changeset CLI - Manage version releases');
  console.log('');
  console.log('Usage:');
  console.log('  node scripts/changeset.js version      - Preview next version from changesets');
  console.log('  node scripts/changeset.js release [type] - Create release and update CHANGELOG');
  console.log('');
  console.log('Release types: patch, minor, major');
  console.log('');
  console.log('Examples:');
  console.log('  node scripts/changeset.js version');
  console.log('  node scripts/changeset.js release');
  console.log('  node scripts/changeset.js release patch');
  console.log('  node scripts/changeset.js release minor');
  console.log('  node scripts/changeset.js release major');
}

// Main entry point
function main() {
  const args = process.argv.slice(2);
  
  if (args.length === 0 || args[0] === '--help' || args[0] === '-h') {
    showHelp();
    return;
  }
  
  const command = args[0];
  
  try {
    switch (command) {
      case 'version':
        runVersion();
        break;
      case 'release':
        runRelease(args[1]);
        break;
      default:
        console.error(formatErrorMessage(`Unknown command: ${command}`));
        console.log('');
        showHelp();
        process.exit(1);
    }
  } catch (error) {
    console.error(formatErrorMessage(error.message));
    process.exit(1);
  }
}

main();
