// @ts-check
/// <reference types="@actions/github-script" />
// @safe-outputs-exempt SEC-005: this file only normalizes issue-intent labels/metadata; "target repository" appears only in a label-not-found error message, with no target-repo parameter or cross-repo write path.

const { sanitizeContent } = require("./sanitize_content.cjs");
const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { hasRuntimeFeature, parseRuntimeFeatures } = require("./runtime_features.cjs");

const ISSUE_INTENTS_FEATURE = "issue_intents";
const ISSUE_INTENT_RATIONALE_MAX_LENGTH = 1024;

function hasIssueIntentsRuntimeFeature() {
  if (typeof global.hasRuntimeFeature === "function") {
    return global.hasRuntimeFeature(ISSUE_INTENTS_FEATURE);
  }
  return hasRuntimeFeature(parseRuntimeFeatures(process.env.GH_AW_RUNTIME_FEATURES), ISSUE_INTENTS_FEATURE);
}

function normalizeIssueIntentMetadata(source) {
  if (!source || typeof source !== "object") {
    return {};
  }

  /** @type {{ rationale?: string, confidence?: "LOW"|"MEDIUM"|"HIGH", suggest?: boolean }} */
  const metadata = {};

  if (typeof source.rationale === "string") {
    const rationale = sanitizeContent(source.rationale, { maxLength: ISSUE_INTENT_RATIONALE_MAX_LENGTH }).trim();
    if (rationale) {
      metadata.rationale = rationale;
    }
  }

  if (source.confidence !== undefined && source.confidence !== null && source.confidence !== "") {
    const confidenceRaw = String(source.confidence).trim().toUpperCase();
    /** @type {"LOW"|"MEDIUM"|"HIGH"} */
    let confidence;
    switch (confidenceRaw) {
      case "LOW":
        confidence = "LOW";
        break;
      case "MEDIUM":
        confidence = "MEDIUM";
        break;
      case "HIGH":
        confidence = "HIGH";
        break;
      default:
        throw new Error(`Invalid confidence ${JSON.stringify(source.confidence)}. Expected one of: LOW, MEDIUM, HIGH.`);
    }
    metadata.confidence = confidence;
  }

  if (source.suggest !== undefined) {
    if (typeof source.suggest !== "boolean") {
      throw new Error(`Invalid suggest ${JSON.stringify(source.suggest)}. Expected a boolean value.`);
    }
    if (source.suggest) {
      metadata.suggest = true;
    }
  }

  return metadata;
}

function normalizeIssueIntentLabelSpecs(labels) {
  if (labels === undefined) {
    return [];
  }
  if (!Array.isArray(labels)) {
    const receivedType = labels === null ? "null" : typeof labels;
    throw new Error(`Invalid labels. Expected an array of label names or label spec objects; received ${receivedType}.`);
  }

  return labels.map((label, index) => {
    if (typeof label === "string") {
      const name = sanitizeLabelContent(label);
      if (!name) {
        throw new Error(`Invalid labels[${index}] entry. Label names must be non-empty strings.`);
      }
      if (name.startsWith("-")) {
        throw new Error(`Label removal is not permitted. Found line starting with '-': ${name}`);
      }
      return { name };
    }

    if (!label || typeof label !== "object" || typeof label.name !== "string") {
      throw new Error(`Invalid labels[${index}] entry. Expected a string label name or an object with a string "name" field.`);
    }

    const name = sanitizeLabelContent(label.name);
    if (!name) {
      throw new Error(`Invalid labels[${index}] entry. Label names must be non-empty strings.`);
    }
    if (name.startsWith("-")) {
      throw new Error(`Label removal is not permitted. Found line starting with '-': ${name}`);
    }

    return {
      name,
      ...normalizeIssueIntentMetadata(label),
    };
  });
}

function normalizeIssueIntentLabelNames(labels) {
  if (labels === undefined) {
    return [];
  }
  if (!Array.isArray(labels)) {
    const receivedType = labels === null ? "null" : typeof labels;
    throw new Error(`Invalid labels. Expected an array of label names or label spec objects; received ${receivedType}.`);
  }

  return labels.map((label, index) => {
    if (typeof label === "string") {
      return label;
    }
    const normalized = normalizeIssueIntentLabelSpecs([label]);
    return normalized[0].name;
  });
}

function getIssueIntentLabelNames(labelSpecs) {
  return labelSpecs.map(label => label.name);
}

function buildIssueIntentLabelUpdates(labelSpecs, labelIdByName) {
  return labelSpecs.map(spec => {
    const labelId = labelIdByName.get(spec.name.toLowerCase());
    if (!labelId) {
      throw new Error(`Label ${JSON.stringify(spec.name)} not found. Ensure the label exists in the target repository.`);
    }

    return {
      labelId,
      ...normalizeIssueIntentMetadata(spec),
    };
  });
}

module.exports = {
  buildIssueIntentLabelUpdates,
  getIssueIntentLabelNames,
  hasIssueIntentsRuntimeFeature,
  normalizeIssueIntentLabelNames,
  normalizeIssueIntentLabelSpecs,
  normalizeIssueIntentMetadata,
};
