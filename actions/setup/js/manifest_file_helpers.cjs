// @ts-check

/** @typedef {import('./types/handler-factory').HandlerConfig} HandlerConfig */

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
  if (!patchContent || !patchContent.trim()) {
    return [];
  }
  const fileSet = new Set();
  const matches = patchContent.matchAll(/^diff --git a\/(.+) b\/(.+)$/gm);
  for (const match of matches) {
    for (const filePath of [match[1], match[2]]) {
      // "dev/null" is the sentinel used when a file is created or deleted; skip it
      if (filePath && filePath !== "dev/null") {
        const parts = filePath.split("/");
        const basename = parts[parts.length - 1];
        if (basename) {
          fileSet.add(basename);
        }
      }
    }
  }
  return Array.from(fileSet);
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
  const matches = patchContent.matchAll(/^diff --git a\/(.+) b\/(.+)$/gm);
  for (const match of matches) {
    for (const filePath of [match[1], match[2]]) {
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
  const changedFiles = extractFilenamesFromPatch(patchContent);
  const manifestFileSet = new Set(manifestFiles);
  const manifestFilesFound = changedFiles.filter(f => manifestFileSet.has(f));
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
  const found = changedPaths.filter(p => pathPrefixes.some(prefix => p.startsWith(prefix)));
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
  const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
  const compiledPatterns = allowedFilePatterns.map(p => globPatternToRegex(p));
  const disallowedFiles = allPaths.filter(p => !compiledPatterns.some(re => re.test(p)));
  return { hasDisallowedFiles: disallowedFiles.length > 0, disallowedFiles };
}

/**
 * Identifies which files in a patch match the given list of ignored-file glob patterns.
 * Matching is done against the full file path (e.g. `.github/workflows/ci.yml`).
 *
 * Glob matching supports `*` (matches any characters except `/`) and `**` (matches
 * any characters including `/`).
 *
 * @param {string} patchContent - The git patch content
 * @param {string[]} ignoredFilePatterns - Glob patterns for files to ignore
 * @returns {{ ignoredFiles: string[] }}
 */
function checkIgnoredFiles(patchContent, ignoredFilePatterns) {
  if (!ignoredFilePatterns || ignoredFilePatterns.length === 0) {
    return { ignoredFiles: [] };
  }
  const allPaths = extractPathsFromPatch(patchContent);
  if (allPaths.length === 0) {
    return { ignoredFiles: [] };
  }
  const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
  const compiledPatterns = ignoredFilePatterns.map(p => globPatternToRegex(p));
  const ignoredFiles = allPaths.filter(p => compiledPatterns.some(re => re.test(p)));
  return { ignoredFiles };
}

/**
 * Filters a git patch to remove entire diff sections for files matching the given glob patterns.
 * This implements `.gitignore`-style exclusion: matching files are completely stripped from the
 * patch so they will not appear in the resulting commit when the patch is applied with `git am`.
 *
 * Both the `a/<path>` (original) and `b/<path>` (new) sides of each diff section are checked
 * against the patterns so that renames are handled correctly: if either side matches, the
 * entire diff section is removed.
 *
 * Glob matching supports `*` (matches any characters except `/`) and `**` (matches
 * any characters including `/`).
 *
 * @param {string} patchContent - The git patch content (from git format-patch)
 * @param {string[]} ignoredFilePatterns - Glob patterns for files to exclude from the patch
 * @returns {string} The filtered patch with ignored file sections removed, or empty string if all files were ignored
 */
function filterPatchByIgnoredFiles(patchContent, ignoredFilePatterns) {
  if (!ignoredFilePatterns || ignoredFilePatterns.length === 0) {
    return patchContent;
  }
  if (!patchContent || !patchContent.trim()) {
    return patchContent;
  }
  const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
  const compiledPatterns = ignoredFilePatterns.map(p => globPatternToRegex(p));

  /**
   * @param {string} filePath
   * @returns {boolean}
   */
  function isIgnored(filePath) {
    return compiledPatterns.some(re => re.test(filePath));
  }

  // Find the start of the first diff section
  const firstDiffMatch = /^diff --git /m.exec(patchContent);
  if (!firstDiffMatch) {
    return patchContent; // No diff sections found
  }

  const header = patchContent.slice(0, firstDiffMatch.index);
  const diffContent = patchContent.slice(firstDiffMatch.index);

  // Split into individual diff sections; lookahead keeps "diff --git" prefix with each section
  const sections = diffContent.split(/(?=^diff --git )/m);

  const keptSections = sections.filter(section => {
    if (!section.startsWith("diff --git ")) {
      return true; // keep non-diff content (e.g. trailing version signature)
    }
    // Extract paths from "diff --git a/<aPath> b/<bPath>"
    const headerMatch = /^diff --git a\/(.+) b\/(.+)$/m.exec(section);
    if (!headerMatch) {
      return true; // can't parse, keep it
    }
    const aPath = headerMatch[1].trimEnd();
    const bPath = headerMatch[2].trimEnd();
    // Remove this section if either path matches an ignored pattern
    return !isIgnored(aPath) && !isIgnored(bPath);
  });

  // If no diff sections remain, return empty string
  const hasRemainingDiffs = keptSections.some(s => s.startsWith("diff --git "));
  if (!hasRemainingDiffs) {
    return "";
  }

  return header + keptSections.join("");
}

/**
 * Evaluates a patch against the configured file-protection policy and returns a
 * single structured result, eliminating nested branching in callers.
 *
 * The checks are applied in order and all must pass:
 * 1. If `allowed_files` is set → every file in the patch must match at least one pattern (deny if not).
 * 2. `protected-files` policy applies independently: "allowed" = skip,
 *    "fallback-to-issue" = create review issue, default ("blocked") = deny.
 *
 * To allow an agent to write protected files, set both `allowed-files` (strict scope) and
 * `protected-files: allowed` (explicit permission) — neither overrides the other implicitly.
 *
 * Note: `ignored-files` filtering must be applied to the patch **before** calling this function
 * (see `filterPatchByIgnoredFiles`). The ignored files will already be absent from the patch
 * by the time protection checks run.
 *
 * @param {string} patchContent - The git patch content (after ignored-files have been filtered out)
 * @param {HandlerConfig} config
 * @returns {{ action: 'allow' } | { action: 'deny', source: 'allowlist'|'protected', files: string[] } | { action: 'fallback', files: string[] }}
 */
function checkFileProtection(patchContent, config) {
  // Step 1: allowlist check (if configured)
  const allowedFilePatterns = Array.isArray(config.allowed_files) ? config.allowed_files : [];
  if (allowedFilePatterns.length > 0) {
    const { disallowedFiles } = checkAllowedFiles(patchContent, allowedFilePatterns);
    if (disallowedFiles.length > 0) {
      return { action: "deny", source: "allowlist", files: disallowedFiles };
    }
  }

  // Step 2: protected-files check (independent of allowlist)
  if (config.protected_files_policy === "allowed") {
    return { action: "allow" };
  }

  const manifestFiles = Array.isArray(config.protected_files) ? config.protected_files : [];
  const prefixes = Array.isArray(config.protected_path_prefixes) ? config.protected_path_prefixes : [];
  const { manifestFilesFound } = checkForManifestFiles(patchContent, manifestFiles);
  const { protectedPathsFound } = checkForProtectedPaths(patchContent, prefixes);
  const allFound = [...manifestFilesFound, ...protectedPathsFound];

  if (allFound.length === 0) {
    return { action: "allow" };
  }

  return config.protected_files_policy === "fallback-to-issue" ? { action: "fallback", files: allFound } : { action: "deny", source: "protected", files: allFound };
}

module.exports = { extractFilenamesFromPatch, extractPathsFromPatch, checkForManifestFiles, checkForProtectedPaths, checkAllowedFiles, checkIgnoredFiles, filterPatchByIgnoredFiles, checkFileProtection };
