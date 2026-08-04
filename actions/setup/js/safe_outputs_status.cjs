// @ts-check

/**
 * Determine whether a processing result is a non-skipped, non-deferred, non-cancelled failure.
 *
 * @param {{success?: boolean, deferred?: boolean, skipped?: boolean, cancelled?: boolean}|null|undefined} result
 * @returns {boolean}
 */
function isFailedProcessingResult(result) {
  return Boolean(result?.success === false && !result?.deferred && !result?.skipped && !result?.cancelled);
}

/**
 * Compute item-level safe-output status for logs, step summary, and GitHub Actions outputs.
 *
 * @param {Array<{success?: boolean, deferred?: boolean, skipped?: boolean, cancelled?: boolean}>|null|undefined} results
 * @returns {{itemsSucceeded: number, itemsFailed: number, status: "success" | "partial_success" | "failure"}}
 */
function computeSafeOutputsStatus(results) {
  const safeResults = Array.isArray(results) ? results : [];
  const itemsSucceeded = safeResults.filter(r => r?.success).length;
  const itemsFailed = safeResults.filter(isFailedProcessingResult).length;
  const status = itemsFailed === 0 ? "success" : itemsSucceeded > 0 ? "partial_success" : "failure";

  return { itemsSucceeded, itemsFailed, status };
}

module.exports = {
  computeSafeOutputsStatus,
  isFailedProcessingResult,
};
