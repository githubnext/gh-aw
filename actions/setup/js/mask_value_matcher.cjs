// @ts-check

/** Replacement used for masked values, matching the runner's own rendering. */
const MASK_REPLACEMENT = "***";

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
 * @param {Iterable<{start: number, end: number, value: string, ambiguous?: boolean, replacement?: string}>} matches
 */
function* overlappingMatchGroups(matches) {
  let group;
  for (const match of matches) {
    if (group && match.start < group.end) {
      group.end = Math.max(group.end, match.end);
      if (match.value.length > group.value.length) {
        group.value = match.value;
        group.replacement = match.replacement;
      } else if (match.replacement !== undefined && match.value.length === group.value.length) {
        group.replacement = match.replacement;
      }
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

module.exports = {
  MASK_REPLACEMENT,
  maskedValueMatches,
  overlappingMatchGroups,
  redactMaskedValues,
};
