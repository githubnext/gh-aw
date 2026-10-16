// @ts-check

/**
 * Helpers that make rendered agent logs aware of GitHub Actions `::add-mask::`
 * workflow commands.
 *
 * The GitHub Actions runner masks values registered with `::add-mask::` in the
 * live job log, but raw log files captured as artifacts (e.g. agent-stdio.log)
 * still contain both the `::add-mask::` command lines and the unmasked values.
 * Rendering those raw lines into issues or comments would leak the secrets, so
 * every log excerpt copied into generated content must be redacted first.
 */

/** Matches an `::add-mask::` workflow command and captures its payload. */
const ADD_MASK_COMMAND_RE = /::add-mask::(.*)$/;

/** Replacement used for masked values, matching the runner's own rendering. */
const MASK_REPLACEMENT = "***";

/**
 * Decode a GitHub Actions workflow command payload.
 *
 * @param {string} value
 * @returns {string}
 */
function unescapeWorkflowCommandValue(value) {
  return value.replace(/%0D/gi, "\r").replace(/%0A/gi, "\n").replace(/%25/g, "%");
}

/**
 * Check whether a log line is an `::add-mask::` workflow command.
 *
 * @param {string} line
 * @returns {boolean}
 */
function isAddMaskCommandLine(line) {
  return ADD_MASK_COMMAND_RE.test(line);
}

/**
 * Collect every value registered through `::add-mask::` in the given log content.
 *
 * Multi-line masked values are expanded into their individual lines as well, so
 * that partial occurrences are redacted too.
 *
 * @param {string} logContent
 * @returns {string[]} Unique masked values, longest first
 */
function collectAddMaskedValues(logContent) {
  /** @type {Set<string>} */
  const values = new Set();
  if (!logContent) return [];
  for (const line of logContent.split("\n")) {
    const match = line.match(ADD_MASK_COMMAND_RE);
    if (!match) continue;
    const decoded = unescapeWorkflowCommandValue(match[1]);
    for (const candidate of decoded.split(/\r\n|\r|\n/)) {
      if (!candidate.trim()) continue;
      // Register both the verbatim value and its trimmed form so that surrounding
      // whitespace in either the command payload or the log text never defeats redaction.
      values.add(candidate);
      values.add(candidate.trim());
    }
  }
  // Both verbatim and trimmed forms are included; trimming cannot increase length,
  // so descending-length order always prefers verbatim before trimmed variants.
  return Array.from(values).sort((a, b) => b.length - a.length);
}

/**
 * Yield non-overlapping matches, breaking equal-position ties by mask order.
 * @param {string} text
 * @param {string[]} maskedValues
 * @param {(match: {start: number, end: number, value: string}) => boolean} [allowed]
 * @param {boolean} [overlapping]
 * @yields {{start: number, end: number, value: string}}
 */
function* maskedValueMatches(text, maskedValues, allowed = () => true, overlapping = false) {
  const pending = [];
  for (let order = 0; order < maskedValues.length; order++) {
    const value = maskedValues[order];
    if (!value) continue;
    const index = text.indexOf(value);
    if (index !== -1) pending.push({ value, index, order });
  }
  const precedes = (a, b) => a.index < b.index || (a.index === b.index && a.order < b.order);
  const siftDown = start => {
    let parent = start;
    while (parent * 2 + 1 < pending.length) {
      let child = parent * 2 + 1;
      if (child + 1 < pending.length && precedes(pending[child + 1], pending[child])) child++;
      if (!precedes(pending[child], pending[parent])) break;
      [pending[parent], pending[child]] = [pending[child], pending[parent]];
      parent = child;
    }
  };
  for (let i = Math.floor(pending.length / 2) - 1; i >= 0; i--) siftDown(i);
  let cursor = 0;
  while (pending.length && cursor < text.length) {
    const next = pending[0];
    let searchStart = cursor;
    if (next.index >= cursor) {
      const match = { start: next.index, end: next.index + next.value.length, value: next.value };
      if (allowed(match)) {
        yield match;
        cursor = overlapping ? match.start + 1 : match.end;
        searchStart = cursor;
      } else {
        searchStart = next.index + 1;
      }
    }
    // Only refresh consumed or overlapping occurrences; later candidates stay cached.
    next.index = text.indexOf(next.value, searchStart);
    if (next.index === -1) {
      const last = pending.pop();
      if (pending.length) pending[0] = last;
    }
    siftDown(0);
  }
}

/**
 * @param {Iterable<{start: number, end: number, value: string, ambiguous?: boolean}>} matches
 */
function* overlappingMatchGroups(matches) {
  let group;
  for (const match of matches) {
    if (group && match.start < group.end) {
      group.end = Math.max(group.end, match.end);
      if (match.value.length > group.value.length) group.value = match.value;
      group.ambiguous ||= match.ambiguous;
    } else {
      if (group) yield group;
      group = { ...match };
    }
  }
  if (group) yield group;
}

/**
 * Coalesce overlapping credentials before choosing their replacement marker.
 * @param {string} text
 * @param {string[]} maskedValues - Longest credentials first
 * @param {(value: string) => string} replacementForValue
 * @returns {string}
 */
function redactOverlappingMaskedValues(text, maskedValues, replacementForValue) {
  const parts = [];
  let cursor = 0;
  for (const match of overlappingMatchGroups(maskedValueMatches(text, maskedValues, undefined, true))) {
    parts.push(text.slice(cursor, match.start), replacementForValue(match.value));
    cursor = match.end;
  }
  parts.push(text.slice(cursor));
  return parts.join("");
}

/**
 * Replace all occurrences of the masked values, defaulting to `***`.
 *
 * @param {string} text
 * @param {string[]} maskedValues
 * @param {(value: string) => string} [replacementForValue] - Explicit markers coalesce overlapping credentials
 * @returns {string}
 */
function redactMaskedValues(text, maskedValues, replacementForValue) {
  if (!text || !maskedValues || maskedValues.length === 0) return text;
  if (replacementForValue)
    return redactOverlappingMaskedValues(
      text,
      [...maskedValues].sort((a, b) => b.length - a.length),
      replacementForValue
    );
  const parts = [];
  let cursor = 0;
  let replacements = 0;
  for (const match of maskedValueMatches(text, maskedValues)) {
    if (match.start > cursor) {
      if (replacements) parts.push(MASK_REPLACEMENT.repeat(replacements));
      replacements = 0;
      parts.push(text.slice(cursor, match.start));
    }
    replacements++;
    cursor = match.end;
  }
  if (replacements) parts.push(MASK_REPLACEMENT.repeat(replacements));
  parts.push(text.slice(cursor));
  return parts.join("");
}

/**
 * Remove `::add-mask::` command lines and redact every masked value from the text.
 *
 * @param {string} text - Text about to be rendered into generated content
 * @param {string[]} maskedValues - Values collected via {@link collectAddMaskedValues}
 * @param {(value: string) => string} [replacementForValue]
 * @returns {string}
 */
function applyAddMaskRedaction(text, maskedValues, replacementForValue) {
  if (!text) return text;
  const withoutCommands = text
    .split("\n")
    .filter(line => !isAddMaskCommandLine(line))
    .join("\n");
  return redactMaskedValues(withoutCommands, maskedValues, replacementForValue);
}

/**
 * Visit quoted JSON tokens without a repeated regex alternation, which can
 * exhaust the RegExp stack on multi-megabyte log records.
 * Malformed prefixes are preserved as text; their quote pairing is ambiguous.
 * @param {string} text
 * @param {(token: string) => string} transformString
 * @param {(text: string) => string} [transformText]
 * @returns {string}
 */
function transformJSONStrings(text, transformString, transformText = value => value) {
  const parts = [];
  let cursor = 0;
  while (cursor < text.length) {
    const start = text.indexOf('"', cursor);
    if (start === -1) {
      parts.push(transformText(text.slice(cursor)));
      break;
    }
    let end = start + 1;
    let invalidString = false;
    while (end < text.length && text[end] !== '"') {
      if (text.charCodeAt(end) < 0x20) {
        invalidString = true;
        break;
      }
      if (text[end] === "\\") {
        end++;
        const escaped = text[end];
        if (escaped === "u") {
          if (!/^[0-9a-fA-F]{4}$/.test(text.slice(end + 1, end + 5))) {
            invalidString = true;
            break;
          }
          end += 4;
        } else if (!escaped || !'"\\/bfnrt'.includes(escaped)) {
          invalidString = true;
          break;
        }
      }
      end++;
    }
    if (invalidString) {
      // Do not rescan escaped quotes in the rejected prefix.
      parts.push(transformText(text.slice(cursor, end)));
      cursor = end;
      continue;
    }
    if (end >= text.length) {
      parts.push(transformText(text.slice(cursor)));
      break;
    }
    parts.push(transformText(text.slice(cursor, start)));
    parts.push(transformString(text.slice(start, end + 1)));
    cursor = end + 1;
  }
  return parts.join("");
}

/**
 * Decode ambiguous escapes for matching only, mapping matches back to source
 * spans so unrelated plain-text paths and literal escapes keep their bytes.
 * @param {string} text
 * @param {string[]} maskedValues
 * @param {(value: string) => string} replacementForValue
 * @returns {string}
 */
function redactAmbiguousMaskedValues(text, maskedValues, replacementForValue) {
  const escapes = { '"': '"', "\\": "\\", "/": "/", b: "\b", f: "\f", n: "\n", r: "\r", t: "\t" };
  const pattern = /\\(["\\/bfnrt]|u[0-9a-fA-F]{4})/g;
  const unescaped = text.replace(pattern, (_, escape) => {
    if (escape[0] === "u") return String.fromCharCode(parseInt(escape.slice(1), 16));
    return escapes[escape];
  });
  let nextEscape = pattern.exec(text);
  let offset = 0;
  const sourceOffset = position => {
    while (nextEscape && nextEscape.index - offset < position) {
      offset += nextEscape[0].length - 1;
      nextEscape = pattern.exec(text);
    }
    return position + offset;
  };
  const parts = [];
  let cursor = 0;
  for (const match of maskedValueMatches(unescaped, maskedValues)) {
    const start = sourceOffset(match.start);
    const end = sourceOffset(match.end);
    parts.push(text.slice(cursor, start), replacementForValue(match.value));
    cursor = end;
  }
  // Ambiguous records retain their JSON-string wrapper, but its decoded value
  // now changes only at the source spans that actually matched a mask.
  return cursor ? JSON.stringify(parts.join("") + text.slice(cursor)) : text;
}

/**
 * @param {string} encoded
 * @returns {string}
 */
function decodeJSONString(encoded) {
  try {
    return JSON.parse(encoded);
  } catch (error) {
    throw new Error("Unable to decode JSON log string during secret redaction", { cause: error });
  }
}

/**
 * Resolve raw and encoded matches together before a malformed record can be
 * rewritten. Quoted-token interiors use decoded semantics; raw matches spanning
 * token boundaries remain eligible so a shorter mask cannot fragment them.
 * @param {string} text
 * @param {string[]} maskedValues
 * @param {(value: string) => string} replacementForValue
 * @returns {string}
 */
function redactMixedMaskedValues(text, maskedValues, replacementForValue) {
  maskedValues = [...maskedValues].sort((a, b) => b.length - a.length);
  const tokens = [];
  let scanned = 0;
  transformJSONStrings(
    text,
    token => {
      decodeJSONString(token);
      tokens.push(scanned, scanned + token.length);
      scanned += token.length;
      return token;
    },
    plain => {
      scanned += plain.length;
      return plain;
    }
  );
  const tokenAt = position => {
    let low = 0;
    let high = tokens.length / 2;
    while (low < high) {
      const mid = (low + high) >>> 1;
      if (tokens[mid * 2] <= position) low = mid + 1;
      else high = mid;
    }
    return (low - 1) * 2;
  };
  const escapeStarts = [];
  const escapeOffsets = [];
  const escapes = { '"': '"', "\\": "\\", "/": "/", b: "\b", f: "\f", n: "\n", r: "\r", t: "\t" };
  let offset = 0;
  const unescaped = text.replace(/\\(["\\/bfnrt]|u[0-9a-fA-F]{4})/g, (match, escape, index) => {
    escapeStarts.push(index - offset);
    offset += match.length - 1;
    escapeOffsets.push(offset);
    return escape[0] === "u" ? String.fromCharCode(parseInt(escape.slice(1), 16)) : escapes[escape];
  });
  const sourceOffset = position => {
    let low = 0;
    let high = escapeStarts.length;
    while (low < high) {
      const mid = (low + high) >>> 1;
      if (escapeStarts[mid] < position) low = mid + 1;
      else high = mid;
    }
    return position + (low ? escapeOffsets[low - 1] : 0);
  };
  const toSource = match => ({ start: sourceOffset(match.start), end: sourceOffset(match.end), value: match.value });
  const rawMatches = maskedValueMatches(
    text,
    maskedValues,
    match => {
      const token = tokenAt(match.start);
      return token < 0 || match.end > tokens[token + 1];
    },
    true
  );
  const decodedMatches = maskedValueMatches(
    unescaped,
    maskedValues,
    match => {
      const source = toSource(match);
      const token = tokenAt(source.start);
      return token < 0 || source.end > tokens[token + 1] || (source.start > tokens[token] && source.end < tokens[token + 1]);
    },
    true
  );
  function* sourceMatches() {
    let raw = rawMatches.next();
    let decoded = decodedMatches.next();
    let decodedSource = decoded.done ? undefined : toSource(decoded.value);
    while (!raw.done || !decoded.done) {
      const useRaw = !decodedSource || (!raw.done && (raw.value.start < decodedSource.start || (raw.value.start === decodedSource.start && raw.value.end >= decodedSource.end)));
      const match = useRaw && !raw.done ? raw.value : decodedSource;
      if (!match) throw new Error("Unable to select log spans during secret redaction");
      const token = tokenAt(match.start);
      const insideToken = token >= 0 && match.start > tokens[token] && match.end < tokens[token + 1];
      const overlapsToken = (token >= 0 && match.start < tokens[token + 1]) || (token + 2 < tokens.length && tokens[token + 2] < match.end);
      yield { ...match, ambiguous: !insideToken && (overlapsToken || text.slice(match.start, match.end) !== match.value) };
      if (useRaw) raw = rawMatches.next();
      else {
        decoded = decodedMatches.next();
        decodedSource = decoded.done ? undefined : toSource(decoded.value);
      }
    }
  }
  const parts = [];
  let cursor = 0;
  let ambiguous = false;
  let repeated = "";
  let repetitions = 0;
  for (const match of overlappingMatchGroups(sourceMatches())) {
    const token = tokenAt(match.start);
    const insideToken = token >= 0 && match.start > tokens[token] && match.end < tokens[token + 1];
    ambiguous ||= match.ambiguous;
    const marker = replacementForValue(match.value);
    const replacement = insideToken ? JSON.stringify(marker).slice(1, -1) : marker;
    if (match.start > cursor || replacement !== repeated) {
      if (repetitions) parts.push(repeated.repeat(repetitions));
      repetitions = 0;
      if (match.start > cursor) parts.push(text.slice(cursor, match.start));
    }
    repeated = replacement;
    repetitions++;
    cursor = match.end;
  }
  if (repetitions) parts.push(repeated.repeat(repetitions));
  parts.push(text.slice(cursor));
  const redacted = transformJSONStrings(parts.join(""), encoded => {
    const value = decodeJSONString(encoded);
    const withoutCommands = applyAddMaskRedaction(value, []);
    return withoutCommands === value ? encoded : JSON.stringify(withoutCommands);
  });
  // A registered raw whole-line value removes all of the original syntax.
  return ambiguous && !maskedValues.includes(text) ? JSON.stringify(redacted) : redacted;
}

/**
 * Sanitize artifact sources while runtime masks are still available in memory.
 * Decode JSON strings first so escaping cannot hide a registered value, redact
 * matching numeric tokens as JSON strings, and preserve untouched records
 * byte-for-byte.
 * @param {string} content
 * @param {string[]} maskedValues
 * @param {(value: any) => any} [redactJSONValue] - Structured protocol redactor for valid JSON records
 * @param {(value: string) => string} [replacementForValue] - Marker for each matched registered value
 * @returns {string}
 */
function redactArtifactMaskedValues(content, maskedValues, redactJSONValue, replacementForValue) {
  if (!content || (!maskedValues.length && !redactJSONValue)) return content;
  const preserveLexemes = !!replacementForValue && !redactJSONValue;
  const replacement = replacementForValue || (() => MASK_REPLACEMENT);
  const redactValue = value => {
    if (typeof value === "string") return applyAddMaskRedaction(value, maskedValues, replacementForValue);
    if (Array.isArray(value)) return value.map(redactValue);
    if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([key, nested]) => [redactMaskedValues(key, maskedValues, replacementForValue), redactValue(nested)]));
    return value;
  };
  const redactString = (encoded, isKey = false) => {
    const value = decodeJSONString(encoded);
    const redacted = isKey ? redactMaskedValues(value, maskedValues, replacementForValue) : redactValue(value);
    return redacted === value ? encoded : JSON.stringify(redacted);
  };
  const redactJson = text => {
    let value;
    try {
      value = JSON.parse(text);
    } catch {
      return undefined;
    }
    // Skip quoted strings and match original numeric tokens at every depth so
    // rounding or exponent notation cannot hide a mask.
    const numericRedacted = transformJSONStrings(
      text,
      token => token,
      plain =>
        plain.replace(/-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?/g, token => {
          const redacted = redactMaskedValues(token, maskedValues, replacementForValue);
          return redacted === token ? token : JSON.stringify(redacted);
        })
    );
    if (numericRedacted !== text) {
      try {
        value = JSON.parse(numericRedacted);
      } catch (error) {
        throw new Error("Unable to decode redacted JSON log record during secret redaction", { cause: error });
      }
    }
    if (preserveLexemes) {
      let cursor = 0;
      return transformJSONStrings(
        numericRedacted,
        token => {
          cursor += token.length;
          let next = cursor;
          while (next < numericRedacted.length && " \t\r\n".includes(numericRedacted[next])) next++;
          return redactString(token, numericRedacted[next] === ":");
        },
        plain => {
          cursor += plain.length;
          return plain;
        }
      );
    }
    const redacted = JSON.stringify((redactJSONValue || redactValue)(value));
    return redacted === JSON.stringify(value) ? numericRedacted : text.match(/^\s*/)[0] + redacted + text.match(/\s*$/)[0];
  };
  const json = redactJson(content);
  if (json !== undefined) return json;
  return content
    .split("\n")
    .filter(line => !isAddMaskCommandLine(line))
    .map(line => {
      const jsonLine = redactJson(line);
      if (jsonLine !== undefined) return jsonLine;
      if (preserveLexemes) return redactMixedMaskedValues(line, maskedValues, replacement);
      // Mixed or malformed logs can still contain recoverable JSON strings.
      const decoded = transformJSONStrings(line, redactString);
      const redacted = redactMaskedValues(decoded, maskedValues, replacementForValue);
      return redactAmbiguousMaskedValues(redacted, maskedValues, replacement);
    })
    .join("\n");
}

module.exports = {
  ADD_MASK_COMMAND_RE,
  MASK_REPLACEMENT,
  applyAddMaskRedaction,
  collectAddMaskedValues,
  isAddMaskCommandLine,
  redactArtifactMaskedValues,
  redactMaskedValues,
  transformJSONStrings,
  unescapeWorkflowCommandValue,
};
