// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeTitle } = require("./sanitize_title.cjs");
const { parseBoolTemplatable } = require("./templatable.cjs");

/**
 * Build shared update payload fields for issue/PR update handlers.
 *
 * @param {Object} item
 * @param {Object} config
 * @param {Object} options
 * @param {boolean} [options.allowTitle=true]
 * @param {string} [options.defaultOperation] - Required when item.body may be present; used as fallback operation if item.operation and configDefaultOperation are both absent.
 * @param {string | undefined} [options.configDefaultOperation]
 * @param {boolean} [options.includeBodyInApiData=false]
 * @param {(() => void) | undefined} [options.onBodyDisallowed]
 * @returns {{updateData: Object, hasCommonUpdates: boolean}}
 */
function buildCommonEntityUpdateData(item, config, options = {}) {
  const { allowTitle = true, defaultOperation, configDefaultOperation, includeBodyInApiData = false, onBodyDisallowed } = options;

  const updateData = {};
  let hasCommonUpdates = false;

  if (allowTitle && item.title !== undefined) {
    updateData.title = sanitizeTitle(item.title);
    hasCommonUpdates = true;
  }

  const canUpdateBody = config.allow_body !== false;
  if (item.body !== undefined && canUpdateBody) {
    const resolvedOperation = item.operation || configDefaultOperation || defaultOperation;
    if (!resolvedOperation) {
      throw new Error("buildCommonEntityUpdateData: defaultOperation is required when body may be present");
    }
    updateData._operation = resolvedOperation;
    updateData._rawBody = item.body;
    if (includeBodyInApiData) {
      updateData.body = item.body;
    }
    hasCommonUpdates = true;
  } else if (item.body !== undefined && !canUpdateBody && typeof onBodyDisallowed === "function") {
    onBodyDisallowed();
  }

  // Always populate _includeFooter: downstream executeUpdate reads it regardless of
  // whether title/body changed, matching pre-refactor behavior in both callers.
  updateData._includeFooter = parseBoolTemplatable(config.footer, true);

  return { updateData, hasCommonUpdates };
}

module.exports = { buildCommonEntityUpdateData };
