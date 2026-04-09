// ================================================================
// Hover Tooltips for Frontmatter Keys
// ================================================================
//
// Shows documentation tooltips when hovering over YAML frontmatter
// keys in the editor. Tooltip content (description, type, enum
// values) comes from autocomplete-data.json, which is generated
// from the main workflow JSON Schema.
//
// This module is dependency-free — tooltips are positioned using
// monospace font math on a <textarea>.
// ================================================================

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
 * Returns { startLine, endLine } as 0-based line indices, or null
 * if no valid frontmatter is found.
 */
function findFrontmatterRegion(lines) {
  if (lines.length === 0) return null;
  if (!lines[0].startsWith('---')) return null;

  for (let i = 1; i < lines.length; i++) {
    if (/^---[ \t]*$/.test(lines[i])) {
      return { startLine: 0, endLine: i };
    }
  }

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
 * Given a 0-based line index, resolve the full key path by walking
 * upward through parent keys based on indentation.
 *
 * For example, if the cursor is on "toolsets" inside:
 *   tools:
 *     github:
 *       toolsets:
 *
 * This returns ["tools", "github", "toolsets"].
 */
function resolveKeyPath(lines, lineIndex, key, lineText) {
  const path = [key];
  const currentIndent = getIndent(lineText);

  if (currentIndent === 0) return path;

  let targetIndent = currentIndent;
  for (let i = lineIndex - 1; i >= 0; i--) {
    const prevLine = lines[i];
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
      return entry;
    }

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
  dom.className = 'tooltip-docs';

  // Header: key name + type badge
  const header = document.createElement('div');
  header.className = 'tooltip-docs-header';

  const nameEl = document.createElement('strong');
  nameEl.textContent = keyName;
  header.appendChild(nameEl);

  if (schemaEntry.type) {
    const typeEl = document.createElement('span');
    typeEl.className = 'tooltip-docs-type';
    typeEl.textContent = schemaEntry.type;
    header.appendChild(typeEl);
  }

  dom.appendChild(header);

  // Description
  if (schemaEntry.desc) {
    const descEl = document.createElement('div');
    descEl.className = 'tooltip-docs-desc';
    descEl.textContent = schemaEntry.desc;
    dom.appendChild(descEl);
  }

  // Enum values
  if (schemaEntry.enum && schemaEntry.enum.length > 0) {
    const enumEl = document.createElement('div');
    enumEl.className = 'tooltip-docs-enum';

    const label = document.createElement('span');
    label.className = 'tooltip-docs-enum-label';
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
      childEl.className = 'tooltip-docs-children';

      const label = document.createElement('span');
      label.className = 'tooltip-docs-enum-label';
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
// Character measurement
// ---------------------------------------------------------------

let _charWidth = null;

/**
 * Measure the width of a single monospace character by using a
 * hidden <span> styled to match the textarea font.
 */
function measureCharWidth(textarea) {
  if (_charWidth !== null) return _charWidth;

  const span = document.createElement('span');
  span.style.cssText = 'position:absolute;visibility:hidden;white-space:pre;';
  span.style.font = window.getComputedStyle(textarea).font;
  span.textContent = 'M';
  document.body.appendChild(span);
  _charWidth = span.getBoundingClientRect().width;
  document.body.removeChild(span);
  return _charWidth;
}

// ---------------------------------------------------------------
// Textarea hover-tooltip wiring
// ---------------------------------------------------------------

/**
 * Attach hover-tooltip behaviour to a <textarea>. Shows schema
 * documentation when the mouse hovers over a YAML frontmatter key.
 * The tooltip is appended to document.body to avoid fixed-positioning
 * issues with transformed parent elements.
 *
 * @param {HTMLTextAreaElement} textarea - The input textarea
 */
export function attachHoverTooltips(textarea) {
  const tooltip = document.createElement('div');
  tooltip.className = 'hover-tooltip';
  tooltip.style.display = 'none';
  document.body.appendChild(tooltip);

  // Cache line splits; invalidated on content change
  let cachedLines = null;
  textarea.addEventListener('input', () => { cachedLines = null; });

  function getLines() {
    if (!cachedLines) cachedLines = textarea.value.split('\n');
    return cachedLines;
  }

  // Cache computed style values; invalidated on resize
  let cachedStyle = null;
  function getStyleMetrics() {
    if (!cachedStyle) {
      const cs = window.getComputedStyle(textarea);
      cachedStyle = {
        lineHeight: parseFloat(cs.lineHeight) || parseFloat(cs.fontSize) * 1.6,
        paddingTop: parseFloat(cs.paddingTop) || 0,
        paddingLeft: parseFloat(cs.paddingLeft) || 0,
      };
    }
    return cachedStyle;
  }
  window.addEventListener('resize', () => { cachedStyle = null; _charWidth = null; });

  let hideTimer = null;

  function showTooltip(dom, x, y) {
    if (hideTimer) { clearTimeout(hideTimer); hideTimer = null; }
    tooltip.replaceChildren(dom);
    tooltip.style.display = 'block';

    // Position near cursor, clamped to viewport
    const pad = 8;
    let left = x + pad;
    let top = y - tooltip.offsetHeight - pad;
    if (top < 0) top = y + pad;
    const minLeft = pad;
    const maxLeft = window.innerWidth - tooltip.offsetWidth - pad;
    left = Math.max(minLeft, Math.min(left, maxLeft));
    tooltip.style.left = left + 'px';
    tooltip.style.top = top + 'px';
  }

  function hideTooltip() {
    if (hideTimer) clearTimeout(hideTimer);
    hideTimer = setTimeout(() => {
      tooltip.style.display = 'none';
      tooltip.replaceChildren();
      hideTimer = null;
    }, 120);
  }

  let rafPending = false;

  textarea.addEventListener('mousemove', (e) => {
    if (rafPending) return;
    rafPending = true;
    requestAnimationFrame(() => {
      rafPending = false;
      handleMouseMove(e);
    });
  });

  function handleMouseMove(e) {
    if (!schemaData) { hideTooltip(); return; }

    const lines = getLines();
    const region = findFrontmatterRegion(lines);
    if (!region) { hideTooltip(); return; }

    const { lineHeight, paddingTop, paddingLeft } = getStyleMetrics();
    const charWidth = measureCharWidth(textarea);

    const rect = textarea.getBoundingClientRect();
    const offsetY = e.clientY - rect.top - paddingTop + textarea.scrollTop;
    const offsetX = e.clientX - rect.left - paddingLeft + textarea.scrollLeft;

    const lineIndex = Math.floor(offsetY / lineHeight);
    const charIndex = Math.floor(offsetX / charWidth);

    if (lineIndex < 0 || lineIndex >= lines.length) { hideTooltip(); return; }

    // Only show inside frontmatter
    if (lineIndex <= region.startLine || lineIndex >= region.endLine) { hideTooltip(); return; }

    const lineText = lines[lineIndex];
    if (lineText.trim() === '---') { hideTooltip(); return; }

    const keyInfo = extractKeyFromLine(lineText, charIndex);
    if (!keyInfo) { hideTooltip(); return; }

    const keyPath = resolveKeyPath(lines, lineIndex, keyInfo.key, lineText);
    const schemaEntry = lookupSchema(keyPath);
    if (!schemaEntry) { hideTooltip(); return; }

    showTooltip(buildTooltipDOM(keyInfo.key, schemaEntry), e.clientX, e.clientY);
  }

  textarea.addEventListener('mouseleave', hideTooltip);
}
