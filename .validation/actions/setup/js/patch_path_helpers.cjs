// @ts-check
const A_PREFIX_LENGTH = 2;
const B_PREFIX_LENGTH = 2;
const QUOTED_PREFIX_LENGTH = 3;

/**
 * Parses a single `diff --git` header line and extracts both old/new paths.
 * Handles unquoted and C-style quoted pathspecs.
 *
 * @param {string} headerLine
 * @returns {{ oldPath: string|null, newPath: string|null, parseable: boolean }}
 */
function parseDiffGitHeader(headerLine) {
  const sanitizedHeaderLine = headerLine.endsWith("\r") ? headerLine.slice(0, -1) : headerLine;
  const rest = sanitizedHeaderLine.replace(/^diff --git /, "");
  if (rest === sanitizedHeaderLine) {
    return { oldPath: null, newPath: null, parseable: false };
  }

  // Git may emit unquoted paths that still contain spaces in `diff --git`
  // headers. In that case, split using the required ` b/` token boundary
  // instead of generic whitespace tokenization.
  if (rest.startsWith("a/")) {
    const quotedSep = rest.indexOf(' "b/');
    const unquotedSep = rest.indexOf(" b/");
    const foundSeparatorIndices = [quotedSep, unquotedSep].filter(idx => idx >= 0);
    if (foundSeparatorIndices.length > 0) {
      const sep = Math.min(...foundSeparatorIndices);
      const oldPath = rest.slice(A_PREFIX_LENGTH, sep) || null;
      const newToken = rest.slice(sep + 1).trimEnd();
      let newPath = null;
      if (newToken.startsWith('"b/')) {
        if (newToken.endsWith('"')) {
          newPath = newToken.slice(QUOTED_PREFIX_LENGTH, -1) || null;
        } else {
          newPath = newToken.slice(QUOTED_PREFIX_LENGTH) || null;
        }
      } else if (newToken.startsWith("b/")) {
        newPath = newToken.slice(B_PREFIX_LENGTH) || null;
      }
      if (oldPath || newPath) {
        return { oldPath, newPath, parseable: true };
      }
    }
  }

  /** @type {string[]} */
  const tokens = [];
  const isWhitespace = ch => ch === " " || ch === "\t" || ch === "\r" || ch === "\n";
  let i = 0;
  while (i < rest.length && tokens.length < 2) {
    while (i < rest.length && isWhitespace(rest[i])) {
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
      while (i < rest.length && !isWhitespace(rest[i])) {
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
      return tok.slice(QUOTED_PREFIX_LENGTH, tok.endsWith('"') ? -1 : undefined);
    }
    if (tok.startsWith("a/") || tok.startsWith("b/")) {
      return tok.slice(B_PREFIX_LENGTH);
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
