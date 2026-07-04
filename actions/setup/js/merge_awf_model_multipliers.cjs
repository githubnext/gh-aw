// @ts-check

// Backward-compatibility shim for older compiled workflows that still require
// merge_awf_model_multipliers.cjs from ${RUNNER_TEMP}/gh-aw/actions.
//
// The canonical implementation now lives in merge_frontmatter_models.cjs.
const frontmatterModels = require("./merge_frontmatter_models.cjs");

module.exports = {
  ...frontmatterModels,
  // Legacy export name preserved for older callers.
  writeMergedModelMultipliersJSON: frontmatterModels.writeMergedModelsJSON,
};
