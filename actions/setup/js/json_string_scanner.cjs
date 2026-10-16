// @ts-check

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

module.exports = { transformJSONStrings };
