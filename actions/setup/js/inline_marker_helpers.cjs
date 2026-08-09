// @ts-check

function collectInlineEndMarkers(content, endMarkerRe) {
  return [...content.matchAll(endMarkerRe)]
    .filter(m => m.index !== undefined)
    .map(m => {
      const markerStart = m.index;
      let lineEnd = markerStart + m[0].length;
      if (lineEnd < content.length && content[lineEnd] === "\n") lineEnd++;
      return { name: m[1], start: markerStart, end: lineEnd };
    });
}

function lineNumberAtOffset(content, offset) {
  return (content.slice(0, offset).match(/\n/g) || []).length + 1;
}

function unknownInlineEndMarkerError(content, orphan, noun) {
  return new Error(`end marker for unknown ${noun} "${orphan.name}" at line ${lineNumberAtOffset(content, orphan.start)} (no matching start marker with that name)`);
}

module.exports = { collectInlineEndMarkers, lineNumberAtOffset, unknownInlineEndMarkerError };
