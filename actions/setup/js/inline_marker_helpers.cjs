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
  return content.slice(0, offset).split("\n").length;
}

function unknownInlineEndMarkerError(content, orphan, prefix, noun) {
  return new Error(`${prefix} end marker for unknown ${noun} "${orphan.name}" at line ${lineNumberAtOffset(content, orphan.start)} (no matching start marker with that name)`);
}

module.exports = { collectInlineEndMarkers, unknownInlineEndMarkerError };
