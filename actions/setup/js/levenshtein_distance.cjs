// @ts-check

/**
 * Compute Levenshtein edit distance between two strings.
 * Cost model: insertion=1, deletion=1, substitution=1.
 *
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */
function levenshteinDistance(a, b) {
  const source = String(a ?? "");
  const target = String(b ?? "");

  if (source === target) {
    return 0;
  }

  const sourceLength = source.length;
  const targetLength = target.length;

  if (sourceLength === 0) {
    return targetLength;
  }
  if (targetLength === 0) {
    return sourceLength;
  }

  let previous = Array.from({ length: targetLength + 1 }, (_, index) => index);
  let current = new Array(targetLength + 1);

  for (let sourceIndex = 1; sourceIndex <= sourceLength; sourceIndex++) {
    current[0] = sourceIndex;
    const sourceChar = source[sourceIndex - 1];

    for (let targetIndex = 1; targetIndex <= targetLength; targetIndex++) {
      const substitutionCost = sourceChar === target[targetIndex - 1] ? 0 : 1;
      const deletion = previous[targetIndex] + 1;
      const insertion = current[targetIndex - 1] + 1;
      const substitution = previous[targetIndex - 1] + substitutionCost;
      current[targetIndex] = Math.min(deletion, insertion, substitution);
    }

    [previous, current] = [current, previous];
  }

  return previous[targetLength];
}

module.exports = {
  levenshteinDistance,
};
