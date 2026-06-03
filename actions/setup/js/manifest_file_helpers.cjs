// @ts-check

/** @typedef {import('./types/handler-factory').HandlerConfig} HandlerConfig */
const { extractDiffGitHeaderEntries } = require("./patch_path_helpers.cjs");

const UNPARSEABLE_DIFF_GIT_HEADER_ERROR_CODE = "ERR_UNPARSEABLE_DIFF_GIT_HEADER";

/**
 * @param {{ headerLine: string }} entry
 * @returns {Error}
 */
function createUnparseableDiffHeaderError(entry) {
  const error = new Error(`Patch contains an unparseable diff --git header: ${entry.headerLine}`);
  error.code = UNPARSEABLE_DIFF_GIT_HEADER_ERROR_CODE;
  return error;
}

/**
 * @param {string[]} changedPaths
 * @returns {string[]}
 */
function extractBasenamesFromPaths(changedPaths) {
  return Array.from(
    new Set(
      changedPaths
        .map(filePath => {
          const parts = filePath.split("/");
          return parts[parts.length - 1] || "";
        })
        .filter(Boolean)
    )
  );
}

/**
 * @param {string[]} changedPaths
 * @param {string[]} allowedFilePatterns
 * @returns {string[]}
 */
function findDisallowedPaths(changedPaths, allowedFilePatterns) {
  const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
  const compiledPatterns = allowedFilePatterns.map(p => globPatternToRegex(p));
  return changedPaths.filter(p => !compiledPatterns.some(re => re.test(p)));
}

/**
 * @param {string[]} changedPaths
 * @param {string[]} pathPrefixes
 * @returns {string[]}
 */
function findProtectedPaths(changedPaths, pathPrefixes) {
  return changedPaths.filter(p => pathPrefixes.some(prefix => p.startsWith(prefix)));
}

/**
 * @param {string[]} changedPaths
 * @param {string[]} excludedPatterns
 * @returns {string[]}
 */
function findMatchingPaths(changedPaths, excludedPatterns) {
  const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
  const compiledPatterns = excludedPatterns.map(p => globPatternToRegex(p));
  return changedPaths.filter(p => compiledPatterns.some(re => re.test(p)));
}

/**
 * @param {string[]} changedPaths
 * @param {string[]} protectedFiles
 * @returns {string[]}
 */
function findManifestBasenames(changedPaths, protectedFiles) {
  const manifestFileSet = new Set(protectedFiles);
  return extractBasenamesFromPaths(changedPaths).filter(f => manifestFileSet.has(f));
}

/**
 * @param {string[]} changedPaths
 * @param {string[]} [excludes]
 * @returns {string[]}
 */
function findTopLevelDotFolderPaths(changedPaths, excludes) {
  const excludeSet = new Set(Array.isArray(excludes) ? excludes : []);
  return changedPaths.filter(p => {
    const slashIdx = p.indexOf("/");
    if (slashIdx === -1) return false;
    const firstComponent = p.substring(0, slashIdx);
    if (!firstComponent.startsWith(".") || firstComponent.length < 2) return false;
    return !excludeSet.has(firstComponent + "/");
  });
}

/**
 * Extracts the unique set of file basenames (filename without directory path) changed in a git patch.
 * Parses "diff --git a/<path> b/<path>" headers to determine which files were modified.
 * Both the a/<path> (original) and b/<path> (new) sides are captured so that renames and copies
 * are detected even when only the original filename matches a manifest file pattern.
 * The special sentinel "dev/null" (used for new-file/deleted-file diffs) is ignored.
 *
 * @param {string} patchContent - The git patch content
 * @returns {string[]} Deduplicated list of file basenames changed in the patch
 */
function extractFilenamesFromPatch(patchContent) {
  return extractBasenamesFromPaths(extractPathsFromPatch(patchContent));
}

/**
 * Extracts the unique set of full file paths changed in a git patch.
 * Parses "diff --git a/<path> b/<path>" headers and returns both sides
 * (excluding the "dev/null" sentinel).  Full paths are needed for
 * prefix-based protection (e.g. ".github/").
 *
 * Both the `a/<path>` (original) and `b/<path>` (new) sides are captured so
 * that renames are fully detected — e.g. renaming `.github/old.yml` to
 * `.github/new.yml` adds both paths to the returned set.
 *
 * @param {string} patchContent - The git patch content
 * @returns {string[]} Deduplicated list of full file paths changed in the patch
 */
function extractPathsFromPatch(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return [];
  }
  const pathSet = new Set();
  const entries = extractDiffGitHeaderEntries(patchContent);
  for (const entry of entries) {
    if (!entry.parseable) {
      throw createUnparseableDiffHeaderError(entry);
    }
    for (const filePath of [entry.oldPath, entry.newPath]) {
      if (filePath && filePath !== "dev/null") {
        pathSet.add(filePath);
      }
    }
  }
  return Array.from(pathSet);
}

/**
 * Checks whether any files modified in the patch match the given list of manifest file names.
 * Matching is done by file basename only (no path comparison).
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} manifestFiles - List of manifest file names to check against (e.g. ["package.json", "go.mod"])
 * @returns {{ hasManifestFiles: boolean, manifestFilesFound: string[] }}
 */
function checkForManifestFiles(patchContent, manifestFiles) {
  if (!manifestFiles || manifestFiles.length === 0) {
    return { hasManifestFiles: false, manifestFilesFound: [] };
  }
  const changedPaths = extractPathsFromPatch(patchContent);
  const manifestFilesFound = findManifestBasenames(changedPaths, manifestFiles);
  return { hasManifestFiles: manifestFilesFound.length > 0, manifestFilesFound };
}

/**
 * Checks whether any files modified in the patch have a path that starts with one of the
 * given protected path prefixes (e.g. ".github/").  This catches arbitrary files under a
 * protected directory, regardless of their filename.
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} pathPrefixes - List of path prefixes to check (e.g. [".github/"])
 * @returns {{ hasProtectedPaths: boolean, protectedPathsFound: string[] }}
 */
function checkForProtectedPaths(patchContent, pathPrefixes) {
  if (!pathPrefixes || pathPrefixes.length === 0) {
    return { hasProtectedPaths: false, protectedPathsFound: [] };
  }
  const changedPaths = extractPathsFromPatch(patchContent);
  const found = findProtectedPaths(changedPaths, pathPrefixes);
  return { hasProtectedPaths: found.length > 0, protectedPathsFound: found };
}

/**
 * Checks all files in a patch against an allowlist of glob patterns.
 * When `allowed-files` is configured, it acts as a strict allowlist: every file
 * touched by the patch must match at least one pattern; files that do not match
 * are returned as disallowed.
 *
 * Glob matching supports `*` (matches any characters except `/`) and `**` (matches
 * any characters including `/`).  Each changed file is tested as its full path
 * (e.g. `.github/workflows/ci.yml`) against the provided patterns.
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} allowedFilePatterns - Glob patterns for files permitted by the allowlist
 * @returns {{ hasDisallowedFiles: boolean, disallowedFiles: string[] }}
 */
function checkAllowedFiles(patchContent, allowedFilePatterns) {
  if (!allowedFilePatterns || allowedFilePatterns.length === 0) {
    return { hasDisallowedFiles: false, disallowedFiles: [] };
  }
  const allPaths = extractPathsFromPatch(patchContent);
  if (allPaths.length === 0) {
    return { hasDisallowedFiles: false, disallowedFiles: [] };
  }
  const disallowedFiles = findDisallowedPaths(allPaths, allowedFilePatterns);
  return { hasDisallowedFiles: disallowedFiles.length > 0, disallowedFiles };
}

/**
 * Identifies which files in a patch match the given list of excluded-file glob patterns.
 * Matching is done against the full file path (e.g. `.github/workflows/ci.yml`).
 *
 * Glob matching supports `*` (matches any characters except `/`) and `**` (matches
 * any characters including `/`).
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} excludedFilePatterns - Glob patterns for files to exclude
 * @returns {{ excludedFiles: string[] }}
 */
function checkExcludedFiles(patchContent, excludedFilePatterns) {
  if (!excludedFilePatterns || excludedFilePatterns.length === 0) {
    return { excludedFiles: [] };
  }
  const allPaths = extractPathsFromPatch(patchContent);
  if (allPaths.length === 0) {
    return { excludedFiles: [] };
  }
  const excludedFiles = findMatchingPaths(allPaths, excludedFilePatterns);
  return { excludedFiles };
}

/**
 * Checks whether any files modified in the patch reside inside a top-level
 * directory whose name starts with ".".  For example, `.cursor/settings.json`
 * or `.vscode/extensions.json` would both match.
 *
 * Root-level dot-files (e.g. `.env`) are NOT matched — only paths that have at
 * least two components (i.e. the file is *inside* a dot-directory) are flagged.
 *
 * Specific dot-folder prefixes can be opted out via the `excludes` parameter
 * (e.g. `[".agents/"]` to allow an agent to write its own instruction files).
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} [excludes] - Optional list of dot-folder prefixes to skip (e.g. [".agents/"])
 * @returns {{ hasTopLevelDotFolders: boolean, topLevelDotFoldersFound: string[] }}
 */
function checkForTopLevelDotFolders(patchContent, excludes) {
  const changedPaths = extractPathsFromPatch(patchContent);
  const found = findTopLevelDotFolderPaths(changedPaths, excludes);
  return { hasTopLevelDotFolders: found.length > 0, topLevelDotFoldersFound: found };
}

/**
 * Evaluates an explicit list of changed paths against the configured file-protection policy.
 * This is used after patch/bundle apply so enforcement is based on the actual git history that
 * will be pushed, rather than on patch parsing alone.
 *
 * @param {string[]} changedPaths
 * @param {HandlerConfig} config
 * @returns {{ action: 'allow' } | { action: 'deny', source: 'allowlist'|'protected', files: string[] } | { action: 'fallback', files: string[] } | { action: 'request_review', files: string[] }}
 */
function checkFileProtectionPaths(changedPaths, config) {
  const normalizedPaths = Array.from(new Set((Array.isArray(changedPaths) ? changedPaths : []).filter(Boolean)));

  const allowedFilePatterns = Array.isArray(config.allowed_files) ? config.allowed_files : [];
  if (allowedFilePatterns.length > 0) {
    const disallowedFiles = findDisallowedPaths(normalizedPaths, allowedFilePatterns);
    if (disallowedFiles.length > 0) {
      return { action: "deny", source: "allowlist", files: disallowedFiles };
    }
  }

  if (config.protected_files_policy === "allowed") {
    return { action: "allow" };
  }

  const manifestFiles = Array.isArray(config.protected_files) ? config.protected_files : [];
  const prefixes = Array.isArray(config.protected_path_prefixes) ? config.protected_path_prefixes : [];
  const manifestFilesFound = findManifestBasenames(normalizedPaths, manifestFiles);
  const protectedPathsFound = findProtectedPaths(normalizedPaths, prefixes);
  const dotFolderExcludes = Array.isArray(config.protected_dot_folder_excludes) ? config.protected_dot_folder_excludes : [];
  const topLevelDotFoldersFound = config.protect_top_level_dot_folders ? findTopLevelDotFolderPaths(normalizedPaths, dotFolderExcludes) : [];
  const allFound = [...new Set([...manifestFilesFound, ...protectedPathsFound, ...topLevelDotFoldersFound])];

  if (allFound.length === 0) {
    return { action: "allow" };
  }

  if (config.protected_files_policy === "fallback-to-issue") {
    return { action: "fallback", files: allFound };
  }
  if (config.protected_files_policy === "request_review") {
    return { action: "request_review", files: allFound };
  }
  return { action: "deny", source: "protected", files: allFound };
}

/**
 * Evaluates a patch against the configured file-protection policy and returns a
 * single structured result, eliminating nested branching in callers.
 *
 * The checks are applied in order and all must pass:
 * 1. If `allowed_files` is set → every file in the patch must match at least one pattern (deny if not).
 * 2. `protected-files` policy applies independently: "allowed" = skip,
 *    "fallback-to-issue" = create review issue, "request_review" = create PR with
 *    request-changes review, default ("blocked") = deny.
 *
 * To allow an agent to write protected files, set both `allowed-files` (strict scope) and
 * `protected-files: allowed` (explicit permission) — neither overrides the other implicitly.
 *
 * Note: `excluded-files` are excluded at patch generation time via `git format-patch`
 * `:(exclude)` pathspecs (see `generateGitPatch` options), so they will never appear in
 * the patch passed to this function.
 *
 * @param {string} patchContent - The git patch content
 * @param {HandlerConfig} config
 * @returns {{ action: 'allow' } | { action: 'deny', source: 'allowlist'|'protected', files: string[] } | { action: 'fallback', files: string[] } | { action: 'request_review', files: string[] }}
 */
function checkFileProtection(patchContent, config) {
  return checkFileProtectionPaths(extractPathsFromPatch(patchContent), config);
}

module.exports = {
  UNPARSEABLE_DIFF_GIT_HEADER_ERROR_CODE,
  extractFilenamesFromPatch,
  extractPathsFromPatch,
  checkForManifestFiles,
  checkForProtectedPaths,
  checkForTopLevelDotFolders,
  checkAllowedFiles,
  checkExcludedFiles,
  checkFileProtection,
  checkFileProtectionPaths,
};
