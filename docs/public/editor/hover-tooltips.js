// ================================================================
// Hover Tooltips for Frontmatter Keys
// ================================================================
//
// Shows documentation tooltips when hovering over YAML frontmatter
// keys in the editor. Tooltip content (description, type, enum
// values) comes from autocomplete-data.json, which is generated
// from the main workflow JSON Schema.
// ================================================================

import { hoverTooltip } from 'https://esm.sh/@codemirror/view@6.39.14';

// ---------------------------------------------------------------
// Schema data loader
// ---------------------------------------------------------------
let schemaData = null;

fetch('./autocomplete-data.json')
  .then(r => r.json())
  .then(data => { schemaData = data; })
  .catch(() => { /* silently degrade — no tooltips if schema fails to load */ });

// ---------------------------------------------------------------
// Frontmatter boundary detection
// ---------------------------------------------------------------

/**
 * Find the frontmatter region (between opening and closing ---).
 * Returns { start, end } as character offsets, or null if no
 * valid frontmatter is found.
 */
function findFrontmatterRegion(doc) {
  // Frontmatter must start at the very beginning of the document.
  // We check the first line rather than converting the entire
  // document to a string for performance reasons.
  if (doc.lines === 0) return null;

  const firstLine = doc.line(1);
  if (!firstLine.text.startsWith('---')) return null;

  // Scan forward line-by-line to find the closing --- on its own line.
  // A closing line looks like: "---" followed by optional spaces/tabs.
  for (let lineNumber = 2; lineNumber <= doc.lines; lineNumber++) {
    const line = doc.line(lineNumber);
    if (/^---[ \t]*$/.test(line.text)) {
      // The frontmatter region spans from character 0 to the end of
      // the closing --- line.
      return {
        start: 0,
        end: line.to,
      };
    }
  }

  // No valid closing delimiter found.
  return null;
}

// ---------------------------------------------------------------
// Key extraction from YAML lines
// ---------------------------------------------------------------

/**
 * Extract the YAML key name from a line, if the position falls
 * within the key portion (before the colon).
 *
 * Returns { key, keyStart, keyEnd } relative to the line, or null.
 */
function extractKeyFromLine(lineText, posInLine) {
  // A YAML key line looks like: "  some-key: value" or "  some-key:"
  const match = lineText.match(/^(\s*)([\w][\w.-]*)(\s*:)/);
  if (!match) return null;

  const indent = match[1].length;
  const key = match[2];
  const keyStart = indent;
  const keyEnd = indent + key.length;

  // Only trigger if the hover position is within the key text
  if (posInLine < keyStart || posInLine >= keyEnd) return null;

  return { key, keyStart, keyEnd };
}

/**
 * Determine the indentation level (number of spaces) of a line.
 */
function getIndent(lineText) {
  const match = lineText.match(/^(\s*)/);
  return match ? match[1].length : 0;
}

/**
 * Given a line number in the document, resolve the full key path
 * by walking upward through parent keys based on indentation.
 *
 * For example, if the cursor is on "toolsets" inside:
 *   tools:
 *     github:
 *       toolsets:
 *
 * This returns ["tools", "github", "toolsets"].
 */
function resolveKeyPath(doc, lineNumber, key, lineText) {
  const path = [key];
  const currentIndent = getIndent(lineText);

  if (currentIndent === 0) return path;

  // Walk upward to find parent keys
  let targetIndent = currentIndent;
  for (let i = lineNumber - 1; i >= 0; i--) {
    const prevLine = doc.line(i + 1).text; // doc.line is 1-based
    // Skip blank lines and comments
    if (prevLine.trim() === '' || prevLine.trim().startsWith('#')) continue;

    const prevIndent = getIndent(prevLine);
    if (prevIndent < targetIndent) {
      const parentMatch = prevLine.match(/^(\s*)([\w][\w.-]*)(\s*:)/);
      if (parentMatch) {
        path.unshift(parentMatch[2]);
        targetIndent = prevIndent;
        if (prevIndent === 0) break;
      }
    }
  }

  return path;
}

// ---------------------------------------------------------------
// Schema lookup
// ---------------------------------------------------------------

/**
 * Look up a key path in the schema data, returning the schema
 * entry for that path, or null if not found.
 */
function lookupSchema(keyPath) {
  if (!schemaData || !schemaData.root) return null;

  let current = schemaData.root;

  for (let i = 0; i < keyPath.length; i++) {
    const segment = keyPath[i];
    const entry = current[segment];
    if (!entry) return null;

    if (i === keyPath.length - 1) {
      // This is the target key
      return entry;
    }

    // Navigate into children for the next segment
    if (entry.children) {
      current = entry.children;
    } else {
      return null;
    }
  }

  return null;
}

// ---------------------------------------------------------------
// Tooltip DOM construction
// ---------------------------------------------------------------

/**
 * Build the tooltip DOM element for a schema entry.
 */
function buildTooltipDOM(keyName, schemaEntry) {
  const dom = document.createElement('div');
  dom.className = 'cm-tooltip-docs';

  // Header: key name + type badge
  const header = document.createElement('div');
  header.className = 'cm-tooltip-docs-header';

  const nameEl = document.createElement('strong');
  nameEl.textContent = keyName;
  header.appendChild(nameEl);

  if (schemaEntry.type) {
    const typeEl = document.createElement('span');
    typeEl.className = 'cm-tooltip-docs-type';
    typeEl.textContent = schemaEntry.type;
    header.appendChild(typeEl);
  }

  dom.appendChild(header);

  // Description
  if (schemaEntry.desc) {
    const descEl = document.createElement('div');
    descEl.className = 'cm-tooltip-docs-desc';
    descEl.textContent = schemaEntry.desc;
    dom.appendChild(descEl);
  }

  // Enum values
  if (schemaEntry.enum && schemaEntry.enum.length > 0) {
    const enumEl = document.createElement('div');
    enumEl.className = 'cm-tooltip-docs-enum';

    const label = document.createElement('span');
    label.className = 'cm-tooltip-docs-enum-label';
    label.textContent = 'Values: ';
    enumEl.appendChild(label);

    const code = document.createElement('code');
    code.textContent = schemaEntry.enum.map(v => String(v)).join(' | ');
    enumEl.appendChild(code);

    dom.appendChild(enumEl);
  }

  // Children hint (if the key has sub-keys)
  if (schemaEntry.children) {
    const childKeys = Object.keys(schemaEntry.children);
    if (childKeys.length > 0) {
      const childEl = document.createElement('div');
      childEl.className = 'cm-tooltip-docs-children';

      const label = document.createElement('span');
      label.className = 'cm-tooltip-docs-enum-label';
      label.textContent = 'Keys: ';
      childEl.appendChild(label);

      const code = document.createElement('code');
      const displayKeys = childKeys.slice(0, 8);
      code.textContent = displayKeys.join(', ') + (childKeys.length > 8 ? ', ...' : '');
      childEl.appendChild(code);

      dom.appendChild(childEl);
    }
  }

  return dom;
}

// ---------------------------------------------------------------
// CodeMirror hoverTooltip extension
// ---------------------------------------------------------------

export const frontmatterHoverTooltip = hoverTooltip((view, pos, side) => {
  if (!schemaData) return null;

  const doc = view.state.doc;
  const region = findFrontmatterRegion(doc);
  if (!region) return null;

  // Only show tooltips inside the frontmatter region
  if (pos < region.start || pos >= region.end) return null;

  // Get the line at the hover position
  const line = doc.lineAt(pos);
  const lineText = line.text;
  const posInLine = pos - line.from;

  // Skip the opening/closing --- delimiters
  if (lineText.trim() === '---') return null;

  // Extract the key at the hover position
  const keyInfo = extractKeyFromLine(lineText, posInLine);
  if (!keyInfo) return null;

  // Resolve the full key path (handles nested keys)
  const lineNumber = line.number - 1; // 0-based for our helper
  const keyPath = resolveKeyPath(doc, lineNumber, keyInfo.key, lineText);

  // Look up the schema entry
  const schemaEntry = lookupSchema(keyPath);
  if (!schemaEntry) return null;

  // Calculate absolute positions for the key span
  const wordStart = line.from + keyInfo.keyStart;
  const wordEnd = line.from + keyInfo.keyEnd;

  return {
    pos: wordStart,
    end: wordEnd,
    above: true,
    create() {
      const dom = buildTooltipDOM(keyInfo.key, schemaEntry);
      return { dom };
    }
  };
});
