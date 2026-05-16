// @ts-check

/**
 * Parses a single `diff --git` header line and extracts both old/new paths.
 * Handles unquoted and C-style quoted pathspecs.
 *
 * @param {string} headerLine
 * @returns {{ oldPath: string|null, newPath: string|null, parseable: boolean }}
 */
function parseDiffGitHeader(headerLine) {
  const rest = headerLine.replace(/^diff --git /, "");
  if (rest === headerLine) {
    return { oldPath: null, newPath: null, parseable: false };
  }

  /** @type {string[]} */
  const tokens = [];
  let i = 0;
  while (i < rest.length && tokens.length < 2) {
    while (i < rest.length && rest[i] === " ") {
      i++;
    }
    if (i >= rest.length) {
      break;
    }

    let token = "";
    if (rest[i] === '"') {
      token += rest[i++];
      let closedQuote = false;
      while (i < rest.length) {
        const ch = rest[i++];
        token += ch;
        if (ch === "\\" && i < rest.length) {
          token += rest[i++];
        } else if (ch === '"') {
          closedQuote = true;
          break;
        }
      }
      if (!closedQuote) {
        return { oldPath: null, newPath: null, parseable: false };
      }
    } else {
      while (i < rest.length && rest[i] !== " ") {
        token += rest[i++];
      }
    }
    tokens.push(token);
  }

  if (tokens.length < 2) {
    return { oldPath: null, newPath: null, parseable: false };
  }

  const stripPrefix = tok => {
    if (tok.startsWith('"a/') || tok.startsWith('"b/')) {
      return tok.slice(3, tok.endsWith('"') ? -1 : undefined);
    }
    if (tok.startsWith("a/") || tok.startsWith("b/")) {
      return tok.slice(2);
    }
    return tok;
  };

  const oldPath = stripPrefix(tokens[0]) || null;
  const newPath = stripPrefix(tokens[1]) || null;
  if (!oldPath && !newPath) {
    return { oldPath: null, newPath: null, parseable: false };
  }

  return { oldPath, newPath, parseable: true };
}

/**
 * Extracts parsed entries for all `diff --git` headers in a patch.
 *
 * @param {string} patchContent
 * @returns {{ oldPath: string|null, newPath: string|null, parseable: boolean, headerIndex: number, headerLine: string }[]}
 */
function extractDiffGitHeaderEntries(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return [];
  }

  /** @type {{ oldPath: string|null, newPath: string|null, parseable: boolean, headerIndex: number, headerLine: string }[]} */
  const entries = [];
  const headerRe = /^diff --git .*$/gm;
  let match;
  while ((match = headerRe.exec(patchContent)) !== null) {
    entries.push({
      ...parseDiffGitHeader(match[0]),
      headerIndex: match.index,
      headerLine: match[0],
    });
  }
  return entries;
}

module.exports = { parseDiffGitHeader, extractDiffGitHeaderEntries };
