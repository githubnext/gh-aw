// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

const { getErrorMessage } = require("./error_helpers.cjs");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
const { execGitSync, getGitAuthEnv } = require("./git_helpers.cjs");
const { parseAllowedRepos, validateRepo } = require("./repo_helpers.cjs");
const { pushSignedCommits } = require("./push_signed_commits.cjs");

/**
 * Push repo-memory changes to git branch
 * Environment variables:
 *   ARTIFACT_DIR: Path to the downloaded artifact directory containing memory files
 *   MEMORY_ID: Memory identifier (used for subdirectory path)
 *   TARGET_REPO: Target repository (owner/name)
 *   BRANCH_NAME: Branch name to push to
 *   MAX_FILE_SIZE: Maximum file size in bytes
 *   MAX_FILE_COUNT: Maximum number of files per commit
 *   MAX_PATCH_SIZE: Maximum total patch size in bytes (default: 10240 = 10KB)
 *   ALLOWED_EXTENSIONS: JSON array of allowed file extensions (e.g., '[".json",".txt"]')
 *   FILE_GLOB_FILTER: Optional space-separated list of file patterns (e.g., "*.md metrics/** data/**")
 *                     Supports * (matches any chars except /) and ** (matches any chars including /)
 *
 *                     IMPORTANT: Patterns are matched against the RELATIVE FILE PATH from the artifact directory,
 *                     NOT against the branch path. Do NOT include the branch name in the patterns.
 *
 *                     Example:
 *                       BRANCH_NAME: memory/code-metrics
 *                       Artifact file: /tmp/gh-aw/repo-memory/default/history.jsonl
 *                       Relative path tested: "history.jsonl"
 *                       CORRECT pattern: "*.jsonl"
 *                       INCORRECT pattern: "memory/code-metrics/*.jsonl"  (includes branch name)
 *
 *                     The branch name is used for git operations (checkout, push) but not for pattern matching.
 *   GH_TOKEN: GitHub token for authentication
 *   GITHUB_RUN_ID: Workflow run ID for commit messages
 */

async function main() {
  const artifactDir = process.env.ARTIFACT_DIR;
  const memoryId = process.env.MEMORY_ID;
  const targetRepo = process.env.TARGET_REPO;
  const branchName = process.env.BRANCH_NAME;
  const maxFileSize = parseInt(process.env.MAX_FILE_SIZE || "10240", 10);
  const maxFileCount = parseInt(process.env.MAX_FILE_COUNT || "100", 10);
  const maxPatchSize = parseInt(process.env.MAX_PATCH_SIZE || "10240", 10);
  const fileGlobFilter = process.env.FILE_GLOB_FILTER || "";
  const formatJSON = process.env.FORMAT_JSON === "true";

  // Parse allowed extensions with error handling
  let allowedExtensions = [".json", ".jsonl", ".txt", ".md", ".csv"];
  if (process.env.ALLOWED_EXTENSIONS) {
    try {
      allowedExtensions = JSON.parse(process.env.ALLOWED_EXTENSIONS);
    } catch (/** @type {any} */ error) {
      core.setFailed(`Failed to parse ALLOWED_EXTENSIONS environment variable: ${getErrorMessage(error)}. Expected JSON array format.`);
      return;
    }
  }

  const ghToken = process.env.GH_TOKEN;
  const githubRunId = process.env.GITHUB_RUN_ID || "unknown";
  const githubServerUrl = process.env.GITHUB_SERVER_URL || "https://github.com";
  const serverHost = githubServerUrl.replace(/^https?:\/\//, "");

  // Log environment variable configuration for debugging
  core.info("Environment configuration:");
  core.info(`  MEMORY_ID: ${memoryId}`);
  core.info(`  MAX_FILE_SIZE: ${maxFileSize}`);
  core.info(`  MAX_FILE_COUNT: ${maxFileCount}`);
  core.info(`  MAX_PATCH_SIZE: ${maxPatchSize}`);
  core.info(`  ALLOWED_EXTENSIONS: ${JSON.stringify(allowedExtensions)}`);
  core.info(`  FILE_GLOB_FILTER: ${fileGlobFilter ? `"${fileGlobFilter}"` : "(empty - all files accepted)"}`);
  core.info(`  FILE_GLOB_FILTER length: ${fileGlobFilter.length}`);
  core.info(`  FORMAT_JSON: ${formatJSON}`);

  /** @param {unknown} value */
  function isPlainObject(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
  }

  /** @param {string} absPath */
  function tryParseJSONFile(absPath) {
    let raw;
    try {
      raw = fs.readFileSync(absPath, "utf8");
    } catch (err) {
      throw new Error(`Failed to read file ${absPath}: ${String(err)}`, { cause: err });
    }
    if (!raw.trim()) {
      throw new Error(`Empty JSON file: ${absPath}`);
    }
    try {
      return JSON.parse(raw);
    } catch (e) {
      throw new Error(`Invalid JSON in ${absPath}: ${getErrorMessage(e)}`, { cause: e });
    }
  }

  // Validate required environment variables
  if (!artifactDir || !memoryId || !targetRepo || !branchName || !ghToken) {
    core.setFailed("Missing required environment variables: ARTIFACT_DIR, MEMORY_ID, TARGET_REPO, BRANCH_NAME, GH_TOKEN");
    return;
  }

  // Validate branch name against the naming constraints enforced by the compiler:
  //
  //  Non-wiki memory (default): "memory/{id}" or "{custom-prefix}/{id}"
  //    - The branch prefix is validated at compile time: 4-32 alphanumeric/hyphen/underscore chars
  //    - generateDefaultBranchName() always produces "{prefix}/{id}" with a "/" separator
  //  Wiki memory: bare branch name, typically "master" or "main"
  //    - Wikis use the repository's default branch and never create an orphan
  //    - The target repo is already appended with ".wiki" by the compiler
  //
  // At runtime we enforce:
  //   1. Namespaced branches (contain "/") to prevent pushing to top-level branches like "main"
  //   2. Known wiki branch names ("master", "main", "gh-pages") are only valid when
  //      TARGET_REPO ends with ".wiki" – the compiler always appends ".wiki" for wiki memory.
  const isNamespaced = /^[a-zA-Z0-9_-]+\/.+/.test(branchName);
  const isKnownWikiBranch = branchName === "master" || branchName === "main" || branchName === "gh-pages";
  const isWikiRepo = targetRepo.endsWith(".wiki");
  if (!isNamespaced && !isKnownWikiBranch) {
    core.setFailed(`ERR_VALIDATION: Invalid branch name "${branchName}": branch name must be namespaced (e.g. "memory/default") or a known wiki branch ("master", "main", "gh-pages")`);
    return;
  }
  if (isKnownWikiBranch && !isWikiRepo) {
    core.setFailed(`ERR_VALIDATION: Branch name "${branchName}" is only valid for wiki repositories (TARGET_REPO must end with ".wiki", got "${targetRepo}")`);
    return;
  }

  // Validate target repository against allowlist
  const allowedReposEnv = process.env.REPO_MEMORY_ALLOWED_REPOS?.trim();
  const allowedRepos = parseAllowedRepos(allowedReposEnv);
  const defaultRepo = `${context.repo.owner}/${context.repo.repo}`;

  const repoValidation = validateRepo(targetRepo, defaultRepo, allowedRepos);
  if (!repoValidation.valid) {
    core.setFailed(`E004: ${repoValidation.error}`);
    return;
  }

  // Validate FILE_GLOB_FILTER patterns: absolute paths (starting with "/") are not supported.
  if (fileGlobFilter) {
    const allPatternStrs = fileGlobFilter.trim().split(/\s+/).filter(Boolean);
    for (const pat of allPatternStrs) {
      if (pat.startsWith("/")) {
        core.setFailed(`FILE_GLOB_FILTER contains an unsupported pattern: "${pat}". Patterns must not start with "/" (absolute paths are not allowed).`);
        return;
      }
    }
  }

  // Source directory with memory files (artifact location)
  // The artifactDir IS the memory directory (no nested structure needed)
  const sourceMemoryPath = artifactDir;

  // Check if artifact memory directory exists
  if (!fs.existsSync(sourceMemoryPath)) {
    core.info(`Memory directory not found in artifact: ${sourceMemoryPath}`);
    return;
  }

  // We're already in the checked out repository (from checkout step)
  const workspaceDir = process.env.GITHUB_WORKSPACE || process.cwd();
  core.info(`Working in repository: ${workspaceDir}`);

  // Split targetRepo into owner and repo name here so they are available for
  // the GitHub REST API seeding calls below (before the checkout block).
  const [targetOwner, targetRepoName] = targetRepo.split("/");

  // Checkout or create the memory branch
  // Note: we do NOT disable sparse checkout here. Disabling sparse checkout on a
  // large repository forces git to materialize all tracked files into the working
  // tree, which can exhaust pipe buffers (ENOBUFS) when thousands of files are
  // involved. The memory branch only holds a handful of small files, so sparse
  // checkout does not need to be altered for either case below.
  core.info(`Checking out branch: ${branchName}...`);

  // baseRef: the remote branch HEAD SHA before we make our local commit.
  // Used by pushSignedCommits to compute git rev-list baseRef..HEAD (i.e. only
  // our new commit) and as the OCC token for the GraphQL createCommitOnBranch
  // mutation.  Empty string when the branch is brand new (orphan).
  let baseRef = "";

  try {
    const repoUrl = `https://x-access-token:${ghToken}@${serverHost}/${targetRepo}.git`;

    // Try to fetch the branch
    try {
      execGitSync(["fetch", repoUrl, `${branchName}:${branchName}`], { stdio: "pipe", suppressLogs: true });
      execGitSync(["checkout", branchName], { stdio: "inherit" });
      core.info(`Checked out existing branch: ${branchName}`);
      // Capture the remote HEAD SHA so pushSignedCommits can compute which
      // local commits are new (rev-list range: baseRef..HEAD).
      baseRef = execGitSync(["rev-parse", "HEAD"]).trim();
      core.info(`Captured baseRef for signed commit push: ${baseRef}`);
    } catch (fetchError) {
      // Determine whether the fetch failed because the branch does not exist
      // (expected for new memory branches) or because of a network / auth
      // problem (unexpected – must surface as a real error and must NOT fall
      // through to orphan-branch creation).
      const fetchErrMsg = getErrorMessage(fetchError);
      const isMissingBranch = /couldn't find remote ref/i.test(fetchErrMsg) || /remote branch .* not found/i.test(fetchErrMsg);
      if (!isMissingBranch) {
        // Re-throw so the outer catch calls core.setFailed with the real cause.
        throw fetchError;
      }

      // Branch doesn't exist – attempt to seed it via the GitHub REST API so
      // the seed commit is server-signed, satisfying "Require signed commits"
      // branch protection rules.  Commits created via the REST API with
      // GITHUB_TOKEN are automatically signed by GitHub.
      const EMPTY_TREE_SHA = "4b825dc642cb6eb9a060e54bf8d69288fbee4904";
      try {
        core.info(`Branch ${branchName} does not exist, seeding via GitHub REST API...`);
        const { data: seedCommit } = await github.rest.git.createCommit({
          owner: targetOwner,
          repo: targetRepoName,
          message: `Initialize ${branchName}`,
          tree: EMPTY_TREE_SHA,
          parents: [],
        });
        let useApiSeedSha = true;
        try {
          await github.rest.git.createRef({
            owner: targetOwner,
            repo: targetRepoName,
            ref: `refs/heads/${branchName}`,
            sha: seedCommit.sha,
          });
        } catch (createRefError) {
          // GitHub returns HTTP 422 with "Reference already exists" when the
          // branch was created concurrently between our fetch-check and this
          // createRef call.  Check for either the status code or the message
          // text since different Octokit versions surface errors differently.
          // Treat as success and use the existing branch instead.
          const createRefErrMsg = getErrorMessage(createRefError);
          if (!/422|Reference already exists/i.test(createRefErrMsg)) {
            throw createRefError;
          }
          core.info(`Branch ${branchName} was created concurrently (422 Reference already exists); using existing branch.`);
          useApiSeedSha = false;
        }
        // Fetch the newly seeded (or concurrently created) branch and check it out.
        execGitSync(["fetch", repoUrl, `${branchName}:${branchName}`], { stdio: "pipe", suppressLogs: true });
        execGitSync(["checkout", branchName], { stdio: "inherit" });
        // Set baseRef to the seed commit SHA (or the existing branch HEAD for
        // the 422 concurrent-creation case) so pushSignedCommits can use the
        // GraphQL signed-commit path instead of the unsigned git push fallback.
        baseRef = useApiSeedSha ? seedCommit.sha : execGitSync(["rev-parse", "HEAD"]).trim();
        core.info(`Seeded and checked out new branch ${branchName} via GitHub API (baseRef: ${baseRef})`);
      } catch (seedError) {
        // Fallback: API seeding failed (e.g. insufficient token permissions).
        // Fall back to the original orphan-branch + git push path and emit a
        // warning so the operator knows signed commits may not be produced.
        core.warning(`Failed to seed branch ${branchName} via GitHub API, falling back to orphan branch: ${getErrorMessage(seedError)}`);
        // baseRef stays "" — pushSignedCommits will use git push for this
        // orphan-branch first push (unsigned, may be rejected by strict rulesets).
        core.info(`Branch ${branchName} does not exist, creating orphan branch...`);
        execGitSync(["checkout", "--orphan", branchName], { stdio: "inherit" });
        // Reset the index to an empty tree. This is O(1) regardless of how many
        // files the source branch contained, avoiding the ENOBUFS error that
        // "git rm -rf ." (with stdio:pipe) causes on large repos (10K+ files).
        execGitSync(["read-tree", "--empty"], { stdio: "pipe" });
        // Clean the working directory using Node.js so we never pipe large git
        // output back through spawnSync buffers.
        core.info("Cleaning working directory for orphan branch...");
        for (const entry of fs.readdirSync(workspaceDir)) {
          if (entry !== ".git") {
            fs.rmSync(path.join(workspaceDir, entry), { recursive: true, force: true });
          }
        }
        core.info(`Created orphan branch: ${branchName}`);
      }
    }
  } catch (error) {
    core.setFailed(`Failed to checkout branch: ${getErrorMessage(error)}`);
    return;
  }

  // Create destination directory in repo
  // Files are copied to the root of the checked-out branch (workspaceDir)
  // The branch name (e.g., "memory/campaigns") identifies the branch,
  // but files go at the branch root, not in a nested subdirectory
  const destMemoryPath = workspaceDir;
  core.info(`Destination directory: ${destMemoryPath}`);

  // Recursively scan and collect files from artifact directory
  let filesToCopy = [];
  /** @type {Array<{path: string, reason: string}>} */
  let filteredOutFiles = [];

  // Compile glob patterns once, outside the scan loop
  /** @type {string[]} */
  let patternStrs = [];
  /** @type {RegExp[]} */
  let compiledPatterns = [];
  if (fileGlobFilter) {
    patternStrs = fileGlobFilter.trim().split(/\s+/).filter(Boolean);
    // Slashless patterns (e.g. "*.json") are matched at the root of a single memory subfolder
    // (depth 1 only).  Patterns that already contain "/" are matched against the full relative
    // path unchanged.
    compiledPatterns = patternStrs.map(pattern => globPatternToRegex(pattern, { matchSubfolderRoot: !pattern.includes("/") }));
    core.info(`File glob filter enabled with ${patternStrs.length} pattern(s):`);
    patternStrs.forEach((pat, idx) => {
      core.info(`  [${idx + 1}] "${pat}" -> regex: ${compiledPatterns[idx].source}`);
    });
  } else {
    core.info("No file glob filter - all files will be accepted");
  }

  /**
   * Recursively scan directory and collect files
   * @param {string} dirPath - Directory to scan
   * @param {string} relativePath - Relative path from sourceMemoryPath (for nested files)
   */
  function scanDirectory(dirPath, relativePath = "") {
    const entries = fs.readdirSync(dirPath, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(dirPath, entry.name);
      const relativeFilePath = relativePath ? path.join(relativePath, entry.name) : entry.name;

      if (entry.isDirectory()) {
        core.info(`  [dir]  ${relativeFilePath}/`);
        // Recursively scan subdirectory
        scanDirectory(fullPath, relativeFilePath);
      } else if (entry.isFile()) {
        const stats = fs.statSync(fullPath);
        const normalizedRelPath = relativeFilePath.replace(/\\/g, "/");

        // Validate file name patterns if filter is set
        if (compiledPatterns.length > 0) {
          const matchResults = compiledPatterns.map((pattern, idx) => {
            const matches = pattern.test(normalizedRelPath);
            core.info(`  [test] ${normalizedRelPath}  pattern[${idx + 1}] "${patternStrs[idx]}" -> ${matches ? "✓ match" : "✗ no match"}`);
            return matches;
          });

          if (!matchResults.some(m => m)) {
            core.info(`  [skip] ${normalizedRelPath} (${stats.size} bytes) — no pattern matched`);
            filteredOutFiles.push({ path: normalizedRelPath, reason: "no pattern matched" });
            // Skip this file instead of failing - it may be from a previous run with different patterns
            return;
          }
        }

        // Validate file size
        if (stats.size > maxFileSize) {
          core.error(`File exceeds size limit: ${relativeFilePath} (${stats.size} bytes > ${maxFileSize} bytes)`);
          core.setFailed("File size validation failed");
          throw new Error("File size validation failed");
        }

        core.info(`  [keep] ${normalizedRelPath} (${stats.size} bytes)`);
        filesToCopy.push({
          relativePath: relativeFilePath,
          source: fullPath,
          size: stats.size,
        });
      }
    }
  }

  try {
    core.info(`Scanning artifact directory: ${sourceMemoryPath}`);
    scanDirectory(sourceMemoryPath);
    core.info(`Scan complete: ${filesToCopy.length} file(s) accepted, ${filteredOutFiles.length} file(s) filtered out`);
    if (filteredOutFiles.length > 0) {
      core.info(`Filtered-out files (${filteredOutFiles.length}):`);
      filteredOutFiles.forEach(f => core.info(`  - ${f.path} (${f.reason})`));
    }
    if (filesToCopy.length > 0 && filesToCopy.length <= 10) {
      core.info("Accepted files:");
      filesToCopy.forEach(f => core.info(`  - ${f.relativePath} (${f.size} bytes)`));
    } else if (filesToCopy.length > 10) {
      core.info(`Accepted files (first 10 of ${filesToCopy.length}):`);
      filesToCopy.slice(0, 10).forEach(f => core.info(`  - ${f.relativePath} (${f.size} bytes)`));
      core.info(`  ... and ${filesToCopy.length - 10} more`);
    }
  } catch (error) {
    core.setFailed(`Failed to scan artifact directory: ${getErrorMessage(error)}`);
    return;
  }

  if (filesToCopy.length === 0) {
    core.info("No files to copy from artifact");
    return;
  }

  // Validate file types before copying
  const { validateMemoryFiles } = require("./validate_memory_files.cjs");
  const validation = validateMemoryFiles(sourceMemoryPath, "repo", allowedExtensions);
  if (!validation.valid) {
    const errorMessage = `File type validation failed: Found ${validation.invalidFiles.length} file(s) with invalid extensions. Only ${allowedExtensions.join(", ")} are allowed. Invalid files: ${validation.invalidFiles.join(", ")}`;
    core.setOutput("validation_failed", "true");
    core.setOutput("validation_error", errorMessage);
    core.setFailed(errorMessage);
    return;
  }

  core.info(`Copying ${filesToCopy.length} validated file(s)...`);

  // Copy files to destination (preserving directory structure)
  for (const file of filesToCopy) {
    const destFilePath = path.join(destMemoryPath, file.relativePath);
    const destDir = path.dirname(destFilePath);

    try {
      // Path traversal protection
      const resolvedRoot = path.resolve(destMemoryPath) + path.sep;
      const resolvedDest = path.resolve(destFilePath);
      if (!resolvedDest.startsWith(resolvedRoot)) {
        core.setFailed(`Refusing to write outside repo-memory directory: ${file.relativePath}`);
        return;
      }

      // Ensure destination directory exists
      fs.mkdirSync(destDir, { recursive: true });

      // Copy file
      fs.copyFileSync(file.source, destFilePath);
      core.info(`Copied: ${file.relativePath} (${file.size} bytes)`);
    } catch (error) {
      core.setFailed(`Failed to copy file ${file.relativePath}: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Format JSON files if requested
  if (formatJSON) {
    core.info("FORMAT_JSON is enabled: formatting .json files as human-readable...");

    /**
     * Recursively find and format all .json files under a directory
     * @param {string} dirPath - Directory to scan
     */
    function formatJSONFilesInDir(dirPath) {
      const entries = fs.readdirSync(dirPath, { withFileTypes: true });
      for (const entry of entries) {
        const fullPath = path.join(dirPath, entry.name);
        if (entry.isDirectory()) {
          if (entry.name !== ".git") {
            formatJSONFilesInDir(fullPath);
          }
        } else if (entry.isFile() && entry.name.endsWith(".json")) {
          try {
            const raw = fs.readFileSync(fullPath, "utf8");
            if (!raw.trim()) {
              continue;
            }
            const parsed = JSON.parse(raw);
            const formatted = JSON.stringify(parsed, null, 2) + "\n";
            if (raw !== formatted) {
              const formattedSize = Buffer.byteLength(formatted, "utf8");
              if (formattedSize > maxFileSize) {
                const sizeError = new Error(`Formatted JSON exceeds MAX_FILE_SIZE: ${path.relative(destMemoryPath, fullPath)} (${formattedSize} bytes > ${maxFileSize} bytes)`);
                sizeError.name = "FormatJSONSizeLimitError";
                throw sizeError;
              }
              fs.writeFileSync(fullPath, formatted, "utf8");
              core.info(`Formatted JSON: ${path.relative(destMemoryPath, fullPath)}`);
            }
          } catch (/** @type {any} */ error) {
            if (error?.name === "FormatJSONSizeLimitError") {
              throw error;
            }
            core.warning(`Skipping JSON formatting for ${path.relative(destMemoryPath, fullPath)}: ${getErrorMessage(error)}`);
          }
        }
      }
    }

    try {
      formatJSONFilesInDir(destMemoryPath);
    } catch (error) {
      core.setFailed(`Failed to format JSON files: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Check if we have any changes to commit
  let changedFileCount = 0;
  try {
    const status = execGitSync(["status", "--porcelain"]);
    const changedEntries = status
      .split("\n")
      .map(line => line.trim())
      .filter(Boolean);
    changedFileCount = changedEntries.length;
  } catch (error) {
    core.setFailed(`Failed to check git status: ${getErrorMessage(error)}`);
    return;
  }

  if (changedFileCount === 0) {
    core.info("No changes detected after copying files");
    return;
  }

  if (changedFileCount > maxFileCount) {
    core.setFailed(`Too many changed files in working directory (${changedFileCount} > ${maxFileCount})`);
    return;
  }

  core.info(`Changed files detected after copying: ${changedFileCount}`);
  core.info("Changes detected, committing and pushing...");

  // Stage all changes.
  // --sparse: "Allow updating index entries outside of the sparse-checkout cone.
  // Normally, git add refuses to update index entries whose paths do not fit
  // within the sparse-checkout cone, since those files might be removed from the
  // working tree without warning." (git-add(1))
  // This is required because "git checkout --orphan" can re-activate
  // sparse-checkout, causing a plain "git add ." to silently skip or reject
  // files on the first run for a new memory branch.
  try {
    execGitSync(["add", "--sparse", "."], { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to stage changes: ${getErrorMessage(error)}`);
    return;
  }

  // Validate total patch size before committing
  // Only additions (new content) are counted toward the patch size limit.
  // Deletions are ignored since removing content is acceptable and does not
  // contribute to the size of the content being pushed.
  try {
    const patchContent = execGitSync(["diff", "--cached"], { stdio: "pipe" });
    // Count only added lines (starting with '+', excluding '+++' file-header lines)
    const addedSizeBytes = patchContent
      .split("\n")
      .filter(line => line.startsWith("+") && !line.startsWith("+++"))
      .reduce((sum, line) => sum + Buffer.byteLength(line + "\n", "utf8"), 0);
    const patchSizeBytes = addedSizeBytes;
    const patchSizeKb = Math.ceil(patchSizeBytes / 1024);
    const maxPatchSizeKb = Math.floor(maxPatchSize / 1024);
    // Allow 20% overhead to account for git diff format (headers, context lines, etc.)
    const effectiveMaxPatchSize = Math.floor(maxPatchSize * 1.2);
    const effectiveMaxPatchSizeKb = Math.floor(effectiveMaxPatchSize / 1024);
    const patchSizeMessage = `Patch additions size: ${patchSizeKb} KB (${patchSizeBytes} bytes) (configured limit: ${maxPatchSizeKb} KB (${maxPatchSize} bytes), effective with 20% overhead: ${effectiveMaxPatchSizeKb} KB (${effectiveMaxPatchSize} bytes))`;
    if (patchSizeBytes > effectiveMaxPatchSize) {
      // Warn at warning level so the size is visible even without verbose mode
      core.warning(patchSizeMessage);
      // Add per-file diff stats to diagnose what's causing the large patch
      // (e.g. a full rewrite of an accumulated history file shows old + new content in the diff)
      try {
        const diffStat = execGitSync(["diff", "--cached", "--stat"], { stdio: "pipe" });
        core.warning(`Patch content breakdown (git diff --stat):\n${diffStat}`);
      } catch (statError) {
        core.warning(`Could not retrieve diff stat: ${getErrorMessage(statError)}`);
      }
      core.setOutput("patch_size_exceeded", "true");
      core.setFailed(
        `Patch additions size (${patchSizeKb} KB, ${patchSizeBytes} bytes) exceeds maximum allowed size (${effectiveMaxPatchSizeKb} KB, ${effectiveMaxPatchSize} bytes, configured limit: ${maxPatchSizeKb} KB with 20% overhead allowance). Reduce the number or size of changes, or increase max-patch-size.`
      );
      return;
    } else if (patchSizeBytes > maxPatchSize) {
      // Within the 20% overhead window — still log as a warning so it's visible
      core.warning(patchSizeMessage);
    } else {
      core.info(patchSizeMessage);
    }
  } catch (error) {
    core.setFailed(`Failed to compute patch additions size: ${getErrorMessage(error)}`);
    return;
  }

  // Commit changes
  try {
    execGitSync(["commit", "-m", `Update repo memory from workflow run ${githubRunId}`], { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to commit changes: ${getErrorMessage(error)}`);
    return;
  }

  // Push using the GraphQL createCommitOnBranch mutation so commits are
  // server-signed (verified) by GitHub.  This satisfies "Require signed
  // commits" branch-protection rules that reject plain git push.
  //
  // pushSignedCommits falls back to a plain `git push` when the mutation
  // cannot be used (merge commits, symlinks, submodule entries).  Under a
  // strict signed-commits ruleset that fallback will also be rejected —
  // that is expected behaviour: remove the unsupported file types and
  // re-run.
  // URL with embedded token used for the pull-on-retry merge step only;
  // pushSignedCommits authenticates via the git extraheader set by
  // actions/checkout (and the gitAuthEnv fallback for the git-push path).
  const repoUrlWithToken = `https://x-access-token:${ghToken}@${serverHost}/${targetRepo}.git`;

  // Point origin at the memory target repo so pushSignedCommits can resolve
  // the remote branch HEAD (ls-remote origin) and the git-push fallback
  // pushes to the correct repository.
  execGitSync(["remote", "set-url", "origin", `https://${serverHost}/${targetRepo}.git`], { stdio: "pipe" });

  const MAX_RETRIES = 3;
  const BASE_DELAY_MS = 1000;
  let currentBaseRef = baseRef;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    core.info(`Pushing changes to ${branchName} (attempt ${attempt + 1}/${MAX_RETRIES + 1})...`);
    try {
      await pushSignedCommits({
        githubClient: github,
        owner: targetOwner,
        repo: targetRepoName,
        branch: branchName,
        baseRef: currentBaseRef,
        cwd: workspaceDir,
        gitAuthEnv: getGitAuthEnv(ghToken),
      });
      core.info(`Successfully pushed changes to ${branchName} branch`);
      return;
    } catch (error) {
      const errMsg = getErrorMessage(error);
      if (attempt < MAX_RETRIES) {
        const delay = BASE_DELAY_MS * Math.pow(2, attempt);
        core.warning(`Push failed (attempt ${attempt + 1}/${MAX_RETRIES + 1}), retrying in ${delay}ms: ${errMsg}`);
        await new Promise(resolve => setTimeout(resolve, delay));

        // Refresh currentBaseRef and merge concurrent remote changes before
        // retrying, in case another run pushed to the branch in the interim.
        try {
          const { stdout: lsOut } = await exec.getExecOutput("git", ["ls-remote", "origin", `refs/heads/${branchName}`], { cwd: workspaceDir });
          const remoteHead = lsOut.trim().split(/\s+/)[0] || "";
          if (remoteHead && remoteHead !== currentBaseRef) {
            currentBaseRef = remoteHead;
            core.info(`Refreshed baseRef for retry: ${currentBaseRef}`);
            // Merge the concurrent remote changes (ours wins on conflicts).
            // Note: this may produce a merge commit; if so, pushSignedCommits
            // will fall back to git push for this retry attempt.
            try {
              execGitSync(["pull", "--no-rebase", "-X", "ours", repoUrlWithToken, branchName], { stdio: "inherit", suppressLogs: true });
            } catch (pullError) {
              core.info(`Pull on retry failed (may be expected for new branches): ${getErrorMessage(pullError)}`);
            }
          }
        } catch (lsRemoteError) {
          // ls-remote failed; proceed with existing currentBaseRef
          core.info(`ls-remote on retry failed, keeping existing baseRef: ${getErrorMessage(lsRemoteError)}`);
        }
      } else {
        // Surface a helpful message when the repository's signed-commits
        // ruleset rejects the git-push fallback path.
        if (/GH013|must have verified signatures|Commits must have verified signatures/i.test(errMsg)) {
          core.setFailed(
            `repo-memory: push to branch ${branchName} was rejected because the repository requires verified (signed) commits. ` +
              `Commits pushed via the GitHub GraphQL API are signed automatically, but the signed-commit path could not be used for this push. ` +
              `If your memory files contain symlinks, executable files, or submodule references, remove them and use regular plain-text files (.json, .jsonl, .txt, .md, .csv). ` +
              `Original error: ${errMsg}`
          );
        } else {
          core.setFailed(`Failed to push changes after ${MAX_RETRIES + 1} attempts: ${errMsg}`);
        }
        return;
      }
    }
  }
}

module.exports = { main };
