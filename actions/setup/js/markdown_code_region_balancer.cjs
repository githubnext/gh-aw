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
 * Common AI-Generated Error Patterns (in order of frequency):
 * 1. Unclosed code blocks at end of content (FIXED: adds closing fence)
 * 2. Nested fences at same indentation level (FIXED: escapes by increasing fence length)
 * 3. Mixed fence types causing confusion (HANDLED: treats ` and ~ separately)
 * 4. Indented bare fences in markdown examples (HANDLED: preserves as content)
 *
 * Rules:
 * - Supports both backtick (`) and tilde (~) fences
 * - Minimum fence length is 3 characters
 * - A fence must be at least as long as the opening fence to close it
 * - Fences can have optional language specifiers
 * - Indentation is preserved but doesn't affect matching
 * - Content inside code blocks should never contain valid fences
 * - Indented fences (different indentation than opener) are treated as content
 *
 * @module markdown_code_region_balancer
 */

/**
 * Balance markdown code regions by attempting to fix mismatched fences.
 *
 * The algorithm:
 * 1. Normalize line endings to ensure consistent processing
 * 2. Parse through markdown line by line, skipping XML comment regions
 * 3. Track code block state (open/closed)
 * 4. When nested fences are detected, increase outer fence length by 1
 * 5. Ensure all opened code blocks are properly closed
 * 6. Quality check: Verify the result doesn't create more unbalanced regions
 *    than the original input - if it does, return the original (normalized)
 *
 * Quality guarantees:
 * - Never creates MORE unbalanced code regions than the input
 * - Always normalizes line endings (\r\n -> \n)
 * - If the algorithm would degrade quality, returns original content
 * - Preserves indentation and fence character types
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
  // Strategy:
  // 1. Process fences in order
  // 2. For each opener, find potential closers
  // 3. If first closer has intermediate opener, defer this opener
  // 4. Otherwise, pair with first direct closer (greedy matching)
  // 5. Make a second pass for deferred openers
  const fenceLengthAdjustments = new Map(); // lineIndex -> new length
  const processed = new Set();
  const deferred = new Set(); // Fences to process in second pass
  const unclosedFences = [];
  const pairedBlocks = []; // Track paired blocks with their line ranges

  // Helper function to check if a line is inside any paired or unclosed block
  const isInsideBlock = lineIndex => {
    // Check if inside a successfully paired block
    for (const block of pairedBlocks) {
      if (lineIndex > block.start && lineIndex < block.end) {
        return true;
      }
    }

    // Check if inside an unclosed block
    // An unclosed block starts at an opening fence and extends to the end of the document
    // if no closer was found
    for (const fence of unclosedFences) {
      if (lineIndex > fence.lineIndex) {
        return true;
      }
    }

    return false;
  };

  let i = 0;
  while (i < fences.length) {
    if (processed.has(i)) {
      i++;
      continue;
    }

    const openFence = fences[i];
    processed.add(i);

    // Find potential closers: bare fences at same indentation that can close this opener
    // For each closer, track if there's an opener between our opener and that closer
    const potentialClosers = [];
    const openIndentLength = openFence.indent.length;

    for (let j = i + 1; j < fences.length; j++) {
      if (processed.has(j)) continue;

      const fence = fences[j];

      // Skip if this fence is inside a paired block
      if (isInsideBlock(fence.lineIndex)) {
        continue;
      }

      // Check if this bare fence can close our opening fence
      const canClose = fence.char === openFence.char && fence.length >= openFence.length && fence.language === "";

      if (canClose) {
        const fenceIndentLength = fence.indent.length;

        // Only consider fences at the SAME indentation as potential closers
        if (fenceIndentLength === openIndentLength) {
          // Check if there's an opener between our opener (i) and this closer (j)
          let hasOpenerBetween = false;
          for (let k = i + 1; k < j; k++) {
            if (processed.has(k)) continue;
            const intermediateFence = fences[k];
            if (intermediateFence.language !== "" && intermediateFence.indent.length === openIndentLength) {
              hasOpenerBetween = true;
              break;
            }
          }

          potentialClosers.push({
            index: j,
            length: fence.length,
            hasOpenerBetween,
          });
        }
      }
    }

    if (potentialClosers.length > 0) {
      // Check the first potential closer
      const firstCloser = potentialClosers[0];

      if (firstCloser.hasOpenerBetween) {
        // There's an opener between our opener and the first closer
        // Defer this opener - we'll process it after intermediate openers are paired
        deferred.add(i);
        processed.delete(i); // Unmark so it can be processed in second pass
        i++;
      } else {
        // No opener before the first closer, so it's a direct match
        // Check if there are MORE closers without intermediate openers
        const directClosers = potentialClosers.filter(c => !c.hasOpenerBetween);

        if (directClosers.length > 1) {
          // Multiple bare closers without intermediate openers
          // Count openers between our opener and the last direct closer to determine if this is true nesting
          const lastDirectCloser = directClosers[directClosers.length - 1];
          let openerCount = 0;
          for (let k = i + 1; k < lastDirectCloser.index; k++) {
            if (processed.has(k)) continue;
            const intermediateFence = fences[k];
            if (intermediateFence.language !== "" && intermediateFence.indent.length === openIndentLength) {
              openerCount++;
            }
          }

          // True nesting: more closers than openers (e.g., 1 opener, 3 closers)
          // Nested blocks: closers = openers + 1 (e.g., 2 openers [including us], 2 closers)
          const closerCount = directClosers.length;
          const isTrueNesting = closerCount > openerCount + 1;

          if (isTrueNesting) {
            // TRUE nesting - use the LAST closer and escape middle ones
            const closerIndex = lastDirectCloser.index;
            processed.add(closerIndex);

            pairedBlocks.push({
              start: fences[i].lineIndex,
              end: fences[closerIndex].lineIndex,
              openIndex: i,
              closeIndex: closerIndex,
            });

            // Increase fence length so middle closers can no longer close
            const maxLength = Math.max(...directClosers.map(c => c.length), openFence.length);
            const newLength = maxLength + 1;
            fenceLengthAdjustments.set(fences[i].lineIndex, newLength);
            fenceLengthAdjustments.set(fences[closerIndex].lineIndex, newLength);

            // Mark middle closers as processed (they're now treated as content)
            for (let k = 0; k < directClosers.length - 1; k++) {
              processed.add(directClosers[k].index);
            }

            i = closerIndex + 1;
          } else {
            // Nested blocks - use the FIRST direct closer (greedy matching)
            const closerIndex = directClosers[0].index;
            processed.add(closerIndex);

            pairedBlocks.push({
              start: fences[i].lineIndex,
              end: fences[closerIndex].lineIndex,
              openIndex: i,
              closeIndex: closerIndex,
            });

            i = closerIndex + 1;
          }
        } else {
          // Only one direct closer, use it (normal case)
          const closerIndex = firstCloser.index;
          processed.add(closerIndex);

          pairedBlocks.push({
            start: fences[i].lineIndex,
            end: fences[closerIndex].lineIndex,
            openIndex: i,
            closeIndex: closerIndex,
          });

          i = closerIndex + 1;
        }
      }
    } else {
      // No closer found - check if this fence is inside a paired block
      const fenceLine = fences[i].lineIndex;

      if (!isInsideBlock(fenceLine)) {
        unclosedFences.push(openFence);
      }

      i++;
    }
  }

  // Fourth pass: Process deferred fences (those that had intermediate openers)
  for (const deferredIndex of deferred) {
    if (processed.has(deferredIndex)) continue; // Already processed in first pass somehow

    const openFence = fences[deferredIndex];
    processed.add(deferredIndex);

    // Find potential closers (same logic as before)
    const potentialClosers = [];
    const openIndentLength = openFence.indent.length;

    for (let j = deferredIndex + 1; j < fences.length; j++) {
      if (processed.has(j)) continue;

      const fence = fences[j];
      if (isInsideBlock(fence.lineIndex)) continue;

      const canClose = fence.char === openFence.char && fence.length >= openFence.length && fence.language === "";

      if (canClose && fence.indent.length === openIndentLength) {
        potentialClosers.push({ index: j, length: fence.length });
      }
    }

    if (potentialClosers.length > 0) {
      // Use the FIRST available closer
      const closerIndex = potentialClosers[0].index;
      processed.add(closerIndex);

      pairedBlocks.push({
        start: fences[deferredIndex].lineIndex,
        end: fences[closerIndex].lineIndex,
        openIndex: deferredIndex,
        closeIndex: closerIndex,
      });
    } else {
      // No closer found
      if (!isInsideBlock(fences[deferredIndex].lineIndex)) {
        unclosedFences.push(openFence);
      }
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

  const resultMarkdown = result.join("\n");

  // Quality check: Verify we didn't make things worse
  // Compare the unbalanced counts before and after
  const originalCounts = countCodeRegions(normalizedMarkdown);
  const resultCounts = countCodeRegions(resultMarkdown);

  // If we created MORE unbalanced regions, give up and return original (normalized)
  if (resultCounts.unbalanced > originalCounts.unbalanced) {
    return normalizedMarkdown;
  }

  // If we didn't improve the balance at all (same unbalanced count),
  // and we modified the markdown significantly, check if we should give up
  if (resultCounts.unbalanced === originalCounts.unbalanced && resultMarkdown !== normalizedMarkdown) {
    // If the total count increased (we added more fences somehow), give up
    if (resultCounts.total > originalCounts.total) {
      return normalizedMarkdown;
    }
  }

  return resultMarkdown;
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
