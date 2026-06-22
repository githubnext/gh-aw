// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Symlink Guard Helper
 *
 * Centralises the lstatSync + isSymbolicLink security check that prevents
 * symlink-traversal attacks across staging and gateway code paths.
 *
 * All callers that need to reject symbolic links should call lstatGuard()
 * instead of calling fs.lstatSync() directly and checking isSymbolicLink()
 * themselves.  This ensures any future hardening (e.g., rejecting device
 * nodes or hard-links) can be applied in one place.
 */

const fs = require("fs");

/**
 * Calls `fs.lstatSync(filePath)` and returns the Stats object unless the path
 * is a symbolic link, in which case `null` is returned.
 *
 * By using `lstatSync` (not `statSync`) the symlink itself is inspected rather
 * than the target it points to, preventing symlink-traversal attacks.
 *
 * Throws if `filePath` does not exist — callers that need to tolerate
 * non-existent paths should wrap the call in a try/catch.
 *
 * @param {string} filePath
 * @returns {import('fs').Stats | null} Stats when not a symlink, null when it is
 */
function lstatGuard(filePath) {
  const stat = fs.lstatSync(filePath);
  return stat.isSymbolicLink() ? null : stat;
}

module.exports = { lstatGuard };
