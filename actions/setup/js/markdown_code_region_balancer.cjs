// @ts-check

/**
 * Balance markdown code regions by ensuring fences are properly matched.
 *
 * This module repairs markdown content where code regions (fenced code blocks)
 * may have improperly nested or unbalanced opening and closing fences.
 *
 * Problem:
 * AI models sometimes generate markdown with nested code regions using the same
 * indentation level, causing parsing issues. For example:
 *
 * ```javascript
 * function example() {
 *   ```
 *   nested content (this shouldn't be here)
 *   ```
 * }
 * ```
 *
 * Rules:
 * - Supports both backtick (`) and tilde (~) fences
 * - Minimum fence length is 3 characters
 * - A fence must be at least as long as the opening fence to close it
 * - Fences can have optional language specifiers
 * - Indentation is preserved but doesn't affect matching
 * - Content inside code blocks should never contain valid fences
 *
 * @module markdown_code_region_balancer
 */

/**
 * Balance markdown code regions by attempting to fix mismatched fences.
 *
 * The algorithm:
 * 1. Parse through markdown line by line, skipping XML comment regions
 * 2. Track code block state (open/closed)
 * 3. When nested fences are detected, increase outer fence length by 1
 * 4. Ensure all opened code blocks are properly closed
 *
 * @param {string} markdown - Markdown content to balance
 * @returns {string} Balanced markdown with properly matched code regions
 */
function balanceCodeRegions(markdown) {
  if (!markdown || typeof markdown !== "string") {
    return markdown || "";
  }

  // Normalize line endings to \n for consistent processing
  const normalizedMarkdown = markdown.replace(/\r\n/g, "\n");

  // Split into lines for processing
  const lines = normalizedMarkdown.split("\n");
  const result = [];

  // First pass: identify XML comment regions to skip
  const xmlCommentRegions = [];
  let inXmlComment = false;
  let xmlCommentStart = -1;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Check for XML comment start
    if (!inXmlComment && line.includes("<!--")) {
      inXmlComment = true;
      xmlCommentStart = i;
    }

    // Check for XML comment end
    if (inXmlComment && line.includes("-->")) {
      xmlCommentRegions.push({ start: xmlCommentStart, end: i });
      inXmlComment = false;
      xmlCommentStart = -1;
    }
  }

  // Helper function to check if a line is inside an XML comment
  const isInXmlComment = lineIndex => {
    for (const region of xmlCommentRegions) {
      if (lineIndex >= region.start && lineIndex <= region.end) {
        return true;
      }
    }
    return false;
  };

  // Second pass: identify all fence lines (excluding those in XML comments)
  const fences = [];
  for (let i = 0; i < lines.length; i++) {
    if (isInXmlComment(i)) continue;

    const fenceMatch = lines[i].match(/^(\s*)(`{3,}|~{3,})([^`~\s]*)?(.*)$/);
    if (fenceMatch) {
      fences.push({
        lineIndex: i,
        indent: fenceMatch[1],
        char: fenceMatch[2][0],
        length: fenceMatch[2].length,
        language: fenceMatch[3] || "",
        trailing: fenceMatch[4] || "",
      });
    }
  }

  // Third pass: Match fences, detecting and fixing nested patterns
  // Key insight: Find ALL valid closers for each opener. If there are multiple,
  // use the LAST one and increase fence length so middle ones become invalid.
  const fenceLengthAdjustments = new Map(); // lineIndex -> new length
  const processed = new Set();
  const unclosedFences = [];
  const pairedBlocks = []; // Track paired blocks

  let i = 0;
  while (i < fences.length) {
    if (processed.has(i)) {
      i++;
      continue;
    }

    const openFence = fences[i];
    processed.add(i);

    // Look for ALL valid closers
    const allMatchingClosers = []; // Track all potential closers

    for (let j = i + 1; j < fences.length; j++) {
      if (processed.has(j)) continue;

      const fence = fences[j];

      // If this fence has a language specifier and matches our char, it's a nested block
      if (fence.language !== "" && fence.char === openFence.char) {
        // Process this nested block with language
        processed.add(j);

        // Find its closer
        for (let k = j + 1; k < fences.length; k++) {
          if (processed.has(k)) continue;
          const nestedCloser = fences[k];
          if (nestedCloser.char === fence.char && nestedCloser.length >= fence.length && nestedCloser.language === "") {
            processed.add(k);
            break;
          }
        }
        continue;
      }

      // Check if this bare fence can close our opening fence
      const canClose = fence.char === openFence.char && fence.length >= openFence.length && fence.language === "";

      if (canClose) {
        allMatchingClosers.push({ index: j, length: fence.length });
      }
    }

    if (allMatchingClosers.length > 0) {
      // Use the LAST valid closer
      const closerIndex = allMatchingClosers[allMatchingClosers.length - 1].index;
      processed.add(closerIndex);

      pairedBlocks.push({
        start: fences[i].lineIndex,
        end: fences[closerIndex].lineIndex,
        openIndex: i,
        closeIndex: closerIndex,
      });

      // If there are multiple closers, we have nested fences
      if (allMatchingClosers.length > 1) {
        // Increase fence length so middle closers can no longer close
        const maxLength = Math.max(...allMatchingClosers.map(c => c.length), openFence.length);
        const newLength = maxLength + 1;
        fenceLengthAdjustments.set(fences[i].lineIndex, newLength);
        fenceLengthAdjustments.set(fences[closerIndex].lineIndex, newLength);

        // Mark middle closers as processed
        for (let k = 0; k < allMatchingClosers.length - 1; k++) {
          processed.add(allMatchingClosers[k].index);
        }
      }

      // Continue from after the closer
      i = closerIndex + 1;
    } else {
      // No closer found - check if this fence is inside a paired block
      const fenceLine = fences[i].lineIndex;
      let isInsideBlock = false;

      for (const block of pairedBlocks) {
        if (fenceLine > block.start && fenceLine < block.end) {
          isInsideBlock = true;
          break;
        }
      }

      if (!isInsideBlock) {
        unclosedFences.push(openFence);
      }

      i++;
    }
  }

  // Fifth pass: build result with adjusted fence lengths
  for (let i = 0; i < lines.length; i++) {
    if (fenceLengthAdjustments.has(i)) {
      const newLength = fenceLengthAdjustments.get(i);
      const fenceMatch = lines[i].match(/^(\s*)(`{3,}|~{3,})([^`~\s]*)?(.*)$/);
      if (fenceMatch) {
        const indent = fenceMatch[1];
        const char = fenceMatch[2][0];
        const language = fenceMatch[3] || "";
        const trailing = fenceMatch[4] || "";
        result.push(`${indent}${char.repeat(newLength)}${language}${trailing}`);
      } else {
        result.push(lines[i]);
      }
    } else {
      result.push(lines[i]);
    }
  }

  // Fifth pass: close any unclosed fences
  for (const openFence of unclosedFences) {
    const closingFence = `${openFence.indent}${openFence.char.repeat(openFence.length)}`;
    result.push(closingFence);
  }

  return result.join("\n");
}

/**
 * Check if markdown has balanced code regions.
 *
 * @param {string} markdown - Markdown content to check
 * @returns {boolean} True if all code regions are balanced, false otherwise
 */
function isBalanced(markdown) {
  if (!markdown || typeof markdown !== "string") {
    return true;
  }

  const normalizedMarkdown = markdown.replace(/\r\n/g, "\n");
  const lines = normalizedMarkdown.split("\n");

  let inCodeBlock = false;
  let openingFence = null;

  for (const line of lines) {
    const fenceMatch = line.match(/^(\s*)(`{3,}|~{3,})([^`~\s]*)?(.*)$/);

    if (fenceMatch) {
      const fence = fenceMatch[2];
      const fenceChar = fence[0];
      const fenceLength = fence.length;

      if (!inCodeBlock) {
        inCodeBlock = true;
        openingFence = {
          char: fenceChar,
          length: fenceLength,
        };
      } else {
        const canClose = openingFence !== null && fenceChar === openingFence.char && fenceLength >= openingFence.length;

        if (canClose) {
          inCodeBlock = false;
          openingFence = null;
        }
        // If can't close, this is an unbalanced fence (nested)
      }
    }
  }

  // Balanced if no unclosed code blocks
  return !inCodeBlock;
}

/**
 * Count code regions in markdown.
 *
 * @param {string} markdown - Markdown content to analyze
 * @returns {{ total: number, balanced: number, unbalanced: number }} Count statistics
 */
function countCodeRegions(markdown) {
  if (!markdown || typeof markdown !== "string") {
    return { total: 0, balanced: 0, unbalanced: 0 };
  }

  const normalizedMarkdown = markdown.replace(/\r\n/g, "\n");
  const lines = normalizedMarkdown.split("\n");

  let total = 0;
  let balanced = 0;
  let inCodeBlock = false;
  let openingFence = null;

  for (const line of lines) {
    const fenceMatch = line.match(/^(\s*)(`{3,}|~{3,})([^`~\s]*)?(.*)$/);

    if (fenceMatch) {
      const fence = fenceMatch[2];
      const fenceChar = fence[0];
      const fenceLength = fence.length;

      if (!inCodeBlock) {
        inCodeBlock = true;
        total++;
        openingFence = {
          char: fenceChar,
          length: fenceLength,
        };
      } else {
        const canClose = openingFence !== null && fenceChar === openingFence.char && fenceLength >= openingFence.length;

        if (canClose) {
          inCodeBlock = false;
          balanced++;
          openingFence = null;
        }
      }
    }
  }

  const unbalanced = total - balanced;
  return { total, balanced, unbalanced };
}

module.exports = {
  balanceCodeRegions,
  isBalanced,
  countCodeRegions,
};
