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
 * 1. Parse through markdown line by line
 * 2. Track code block state (open/closed)
 * 3. When inside a code block, escape any fence-like patterns that could prematurely close the block
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

  // First pass: identify all fence lines
  const fences = [];
  for (let i = 0; i < lines.length; i++) {
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

  // Second pass: Detect and fix improperly nested fences
  // Strategy:
  // 1. For each fence, find ALL subsequent fences that could close it
  // 2. If there are 2+ such fences, it suggests nesting - escape all but the last
  
  const escapeLines = new Set();
  const unclosedFences = [];
  const processed = new Set();
  
  for (let i = 0; i < fences.length; i++) {
    if (processed.has(i)) continue;
    
    const openFence = fences[i];
    processed.add(i);
    
    // Find all potential closers (same char, >= length, NO language)
    // Continue searching until we find a fence that CANNOT match (different char or has language that starts a new block)
    const potentialClosers = [];
    for (let j = i + 1; j < fences.length; j++) {
      if (processed.has(j)) continue;
      
      const fence = fences[j];
      const canClose =
        fence.char === openFence.char &&
        fence.length >= openFence.length &&
        fence.language === "";
      
      if (canClose) {
        potentialClosers.push(j);
      }
      // Don't break - keep looking for all potential closers
    }
    
    if (potentialClosers.length === 0) {
      // No closer found
      unclosedFences.push(openFence);
    } else if (potentialClosers.length === 1) {
      // Exactly one closer - clean pair
      processed.add(potentialClosers[0]);
    } else {
      // Multiple potential closers - nested fences detected
      // Keep the last one as the closer, escape the rest
      const closerIndex = potentialClosers[potentialClosers.length - 1];
      processed.add(closerIndex);
      
      for (let j = 0; j < potentialClosers.length - 1; j++) {
        escapeLines.add(fences[potentialClosers[j]].lineIndex);
        processed.add(potentialClosers[j]);
      }
    }
  }

  // Third pass: build result
  for (let i = 0; i < lines.length; i++) {
    if (escapeLines.has(i)) {
      // Escape this line
      const fenceMatch = lines[i].match(/^(\s*)(`{3,}|~{3,})(.*)$/);
      if (fenceMatch) {
        const indent = fenceMatch[1];
        const fence = fenceMatch[2];
        const rest = fenceMatch[3];
        result.push(`${indent}\\${fence}${rest}`);
      } else {
        result.push(lines[i]);
      }
    } else {
      result.push(lines[i]);
    }
  }

  // Fourth pass: close any unclosed fences
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
        const canClose =
          fenceChar === openingFence.char && fenceLength >= openingFence.length;

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
        const canClose =
          fenceChar === openingFence.char && fenceLength >= openingFence.length;

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
