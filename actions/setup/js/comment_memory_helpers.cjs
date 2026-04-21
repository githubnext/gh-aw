// @ts-check

const fs = require("fs");
const path = require("path");

const COMMENT_MEMORY_TAG = "gh-aw-comment-memory";
const COMMENT_MEMORY_DIR = "/tmp/gh-aw/comment-memory";
const COMMENT_MEMORY_EXTENSION = ".md";
const MAX_MEMORY_ID_LENGTH = 64;

function isSafeMemoryId(memoryId) {
  if (typeof memoryId !== "string" || memoryId.length === 0 || memoryId.length > MAX_MEMORY_ID_LENGTH) {
    return false;
  }
  if (memoryId.includes("..") || memoryId.includes("/") || memoryId.includes("\\")) {
    return false;
  }
  return /^[A-Za-z0-9_-]+$/.test(memoryId);
}

/**
 * @param {string} commentBody
 * @param {(message: string) => void} [warn]
 * @returns {Array<{memoryId: string, content: string}>}
 */
function extractCommentMemoryEntries(commentBody, warn = () => {}) {
  if (!commentBody || typeof commentBody !== "string") {
    return [];
  }

  const entries = [];
  const closeTag = `</${COMMENT_MEMORY_TAG}>`;
  let cursor = 0;
  while (cursor < commentBody.length) {
    const openStart = commentBody.indexOf(`<${COMMENT_MEMORY_TAG} id="`, cursor);
    if (openStart < 0) {
      break;
    }

    const idStart = openStart + `<${COMMENT_MEMORY_TAG} id="`.length;
    const idEnd = commentBody.indexOf('">', idStart);
    if (idEnd < 0) {
      break;
    }

    const memoryId = commentBody.slice(idStart, idEnd);
    const contentStart = idEnd + 2;
    const closeStart = commentBody.indexOf(closeTag, contentStart);
    if (closeStart < 0) {
      break;
    }

    if (isSafeMemoryId(memoryId)) {
      entries.push({
        memoryId,
        content: (commentBody.slice(contentStart, closeStart) || "").trim(),
      });
    } else {
      warn(`skipping unsafe memory_id '${memoryId}'`);
    }

    cursor = closeStart + closeTag.length;
  }
  return entries;
}

function listCommentMemoryFiles(memoryDir = COMMENT_MEMORY_DIR) {
  if (!fs.existsSync(memoryDir)) {
    return [];
  }

  return fs
    .readdirSync(memoryDir)
    .filter(file => file.endsWith(COMMENT_MEMORY_EXTENSION))
    .sort()
    .map(file => ({
      memoryId: file.slice(0, -COMMENT_MEMORY_EXTENSION.length),
      filePath: path.join(memoryDir, file),
    }))
    .filter(entry => isSafeMemoryId(entry.memoryId));
}

module.exports = {
  COMMENT_MEMORY_TAG,
  COMMENT_MEMORY_DIR,
  COMMENT_MEMORY_EXTENSION,
  isSafeMemoryId,
  extractCommentMemoryEntries,
  listCommentMemoryFiles,
};
