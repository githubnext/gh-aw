// @ts-check
/**
 * Remove duplicate title from description
 * @module remove_duplicate_title
 */

/**
 * Removes duplicate title from the beginning of description content.
 * If the description starts with a header (# or ## or ### etc.) that matches
 * the title, it will be removed along with any trailing newlines.
 *
 * @param {string} title - The title text to match and remove
 * @param {string} description - The description content that may contain duplicate title
 * @returns {string} The description with duplicate title removed
 */
function removeDuplicateTitleFromDescription(title, description) {
  // Handle null/undefined/empty inputs
  if (!title || typeof title !== "string") {
    return description || "";
  }
  if (!description || typeof description !== "string") {
    return "";
  }

  const trimmedTitle = title.trim();
  const trimmedDescription = description.trim();

  if (!trimmedTitle || !trimmedDescription) {
    return trimmedDescription;
  }

  // Match any header level (# to ######) followed by the title at the start
  // This regex matches:
  // - Start of string
  // - One or more # characters
  // - One or more spaces
  // - The exact title (escaped for regex special chars)
  // - Optional trailing spaces
  // - Optional newlines after the header
  const escapedTitle = trimmedTitle.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const headerRegex = new RegExp(`^#{1,6}\\s+${escapedTitle}\\s*(?:\\r?\\n)*`, "i");

  if (headerRegex.test(trimmedDescription)) {
    return trimmedDescription.replace(headerRegex, "").trim();
  }

  return trimmedDescription;
}

module.exports = { removeDuplicateTitleFromDescription };
