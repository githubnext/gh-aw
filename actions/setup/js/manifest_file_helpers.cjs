// @ts-check

/**
 * Extracts the unique set of file basenames (filename without directory path) changed in a git patch.
 * Parses "diff --git a/<path> b/<path>" headers to determine which files were modified.
 *
 * @param {string} patchContent - The git patch content
 * @returns {string[]} Deduplicated list of file basenames changed in the patch
 */
function extractFilenamesFromPatch(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return [];
  }
  const fileSet = new Set();
  const matches = patchContent.matchAll(/^diff --git a\/.+ b\/(.+)$/gm);
  for (const match of matches) {
    const filePath = match[1];
    const parts = filePath.split("/");
    const basename = parts[parts.length - 1];
    if (basename) {
      fileSet.add(basename);
    }
  }
  return Array.from(fileSet);
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

module.exports = { extractFilenamesFromPatch, checkForManifestFiles };
