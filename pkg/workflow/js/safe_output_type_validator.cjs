// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Type Validator
 *
 * A data-driven validation engine for safe output types.
 * Validation rules are defined in safe_outputs_tools.json and extended
 * with additional validation metadata.
 */

const { sanitizeContent } = require("./sanitize_content.cjs");
const { isTemporaryId } = require("./temporary_id.cjs");

/**
 * Default max body length for GitHub content
 */
const MAX_BODY_LENGTH = 65000;

/**
 * Maximum length for GitHub usernames
 * Reference: https://github.com/dead-claudia/github-limits
 */
const MAX_GITHUB_USERNAME_LENGTH = 39;

/**
 * Validation metadata for each safe output type.
 * This extends the tools.json schema with validation-specific configuration.
 *
 * @typedef {Object} FieldValidation
 * @property {boolean} [required] - Whether the field is required
 * @property {string} [type] - Expected type: 'string', 'number', 'boolean', 'array'
 * @property {boolean} [sanitize] - Whether to sanitize string content
 * @property {number} [maxLength] - Maximum length for strings
 * @property {boolean} [positiveInteger] - Must be a positive integer
 * @property {boolean} [optionalPositiveInteger] - Optional but if present must be positive integer
 * @property {boolean} [issueOrPRNumber] - Can be issue/PR number or undefined
 * @property {boolean} [issueNumberOrTemporaryId] - Can be issue number or temporary ID
 * @property {string[]} [enum] - Allowed values for the field
 * @property {string} [itemType] - For arrays, the type of items
 * @property {boolean} [itemSanitize] - For arrays, whether to sanitize items
 * @property {number} [itemMaxLength] - For arrays, max length per item
 * @property {string} [pattern] - Regex pattern the value must match
 * @property {string} [patternError] - Error message for pattern mismatch
 */

/**
 * @typedef {Object} TypeValidationConfig
 * @property {number} defaultMax - Default max count for this type
 * @property {Object.<string, FieldValidation>} fields - Field validation rules
 * @property {function(any, number): {isValid: boolean, error?: string}|null} [customValidation] - Custom validation function
 */

/**
 * Validation configuration for all safe output types.
 * Keys use underscores to match normalized type names.
 *
 * @type {Object.<string, TypeValidationConfig>}
 */
const VALIDATION_CONFIG = {
  create_issue: {
    defaultMax: 1,
    fields: {
      title: { required: true, type: "string", sanitize: true, maxLength: 128 },
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      labels: { type: "array", itemType: "string", itemSanitize: true, itemMaxLength: 128 },
      parent: { issueOrPRNumber: true },
      temporary_id: { type: "string" },
    },
  },
  create_agent_task: {
    defaultMax: 1,
    fields: {
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
    },
  },
  add_comment: {
    defaultMax: 1,
    fields: {
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      item_number: { issueOrPRNumber: true },
    },
  },
  create_pull_request: {
    defaultMax: 1,
    fields: {
      title: { required: true, type: "string", sanitize: true, maxLength: 128 },
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      branch: { required: true, type: "string", sanitize: true, maxLength: 256 },
      labels: { type: "array", itemType: "string", itemSanitize: true, itemMaxLength: 128 },
    },
  },
  add_labels: {
    defaultMax: 5,
    fields: {
      labels: { required: true, type: "array", itemType: "string", itemSanitize: true, itemMaxLength: 128 },
      item_number: { issueOrPRNumber: true },
    },
  },
  add_reviewer: {
    defaultMax: 3,
    fields: {
      reviewers: { required: true, type: "array", itemType: "string", itemSanitize: true, itemMaxLength: MAX_GITHUB_USERNAME_LENGTH },
      pull_request_number: { issueOrPRNumber: true },
    },
  },
  assign_milestone: {
    defaultMax: 1,
    fields: {
      issue_number: { issueOrPRNumber: true },
      milestone_number: { required: true, positiveInteger: true },
    },
  },
  assign_to_agent: {
    defaultMax: 1,
    fields: {
      issue_number: { required: true, positiveInteger: true },
      agent: { type: "string", sanitize: true, maxLength: 128 },
    },
  },
  update_issue: {
    defaultMax: 1,
    fields: {
      status: { type: "string", enum: ["open", "closed"] },
      title: { type: "string", sanitize: true, maxLength: 128 },
      body: { type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      issue_number: { issueOrPRNumber: true },
    },
    customValidation: (item, lineNum) => {
      const hasValidField = item.status !== undefined || item.title !== undefined || item.body !== undefined;
      if (!hasValidField) {
        return {
          isValid: false,
          error: `Line ${lineNum}: update_issue requires at least one of: 'status', 'title', or 'body' fields`,
        };
      }
      return null;
    },
  },
  update_pull_request: {
    defaultMax: 1,
    fields: {
      title: { type: "string", sanitize: true, maxLength: 256 },
      body: { type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      operation: { type: "string", enum: ["replace", "append", "prepend"] },
      pull_request_number: { issueOrPRNumber: true },
    },
    customValidation: (item, lineNum) => {
      const hasValidField = item.title !== undefined || item.body !== undefined;
      if (!hasValidField) {
        return {
          isValid: false,
          error: `Line ${lineNum}: update_pull_request requires at least one of: 'title' or 'body' fields`,
        };
      }
      return null;
    },
  },
  push_to_pull_request_branch: {
    defaultMax: 1,
    fields: {
      branch: { required: true, type: "string", sanitize: true, maxLength: 256 },
      message: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      pull_request_number: { issueOrPRNumber: true },
    },
  },
  create_pull_request_review_comment: {
    defaultMax: 1,
    fields: {
      path: { required: true, type: "string" },
      line: { required: true, positiveInteger: true },
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      start_line: { optionalPositiveInteger: true },
      side: { type: "string", enum: ["LEFT", "RIGHT"] },
    },
    customValidation: (item, lineNum) => {
      if (item.start_line !== undefined && item.line !== undefined) {
        const startLine = typeof item.start_line === "string" ? parseInt(item.start_line, 10) : item.start_line;
        const endLine = typeof item.line === "string" ? parseInt(item.line, 10) : item.line;
        if (startLine > endLine) {
          return {
            isValid: false,
            error: `Line ${lineNum}: create_pull_request_review_comment 'start_line' must be less than or equal to 'line'`,
          };
        }
      }
      return null;
    },
  },
  create_discussion: {
    defaultMax: 1,
    fields: {
      title: { required: true, type: "string", sanitize: true, maxLength: 128 },
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      category: { type: "string", sanitize: true, maxLength: 128 },
    },
  },
  close_discussion: {
    defaultMax: 1,
    fields: {
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      reason: { type: "string", enum: ["RESOLVED", "DUPLICATE", "OUTDATED", "ANSWERED"] },
      discussion_number: { optionalPositiveInteger: true },
    },
  },
  close_issue: {
    defaultMax: 1,
    fields: {
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      issue_number: { optionalPositiveInteger: true },
    },
  },
  close_pull_request: {
    defaultMax: 1,
    fields: {
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
      pull_request_number: { optionalPositiveInteger: true },
    },
  },
  missing_tool: {
    defaultMax: 20,
    fields: {
      tool: { required: true, type: "string", sanitize: true, maxLength: 128 },
      reason: { required: true, type: "string", sanitize: true, maxLength: 256 },
      alternatives: { type: "string", sanitize: true, maxLength: 512 },
    },
  },
  update_release: {
    defaultMax: 1,
    fields: {
      tag: { type: "string", sanitize: true, maxLength: 256 },
      operation: { required: true, type: "string", enum: ["replace", "append", "prepend"] },
      body: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
    },
  },
  upload_asset: {
    defaultMax: 10,
    fields: {
      path: { required: true, type: "string" },
    },
  },
  noop: {
    defaultMax: 1,
    fields: {
      message: { required: true, type: "string", sanitize: true, maxLength: MAX_BODY_LENGTH },
    },
  },
  create_code_scanning_alert: {
    defaultMax: 40,
    fields: {
      file: { required: true, type: "string", sanitize: true, maxLength: 512 },
      line: { required: true, positiveInteger: true },
      severity: { required: true, type: "string", enum: ["error", "warning", "info", "note"] },
      message: { required: true, type: "string", sanitize: true, maxLength: 2048 },
      column: { optionalPositiveInteger: true },
      ruleIdSuffix: { type: "string", pattern: "^[a-zA-Z0-9_-]+$", patternError: "must contain only alphanumeric characters, hyphens, and underscores", sanitize: true, maxLength: 128 },
    },
  },
  link_sub_issue: {
    defaultMax: 5,
    fields: {
      parent_issue_number: { required: true, issueNumberOrTemporaryId: true },
      sub_issue_number: { required: true, issueNumberOrTemporaryId: true },
    },
    customValidation: (item, lineNum) => {
      // Normalize values for comparison
      const normalizeValue = v => (typeof v === "string" ? v.toLowerCase() : v);
      if (normalizeValue(item.parent_issue_number) === normalizeValue(item.sub_issue_number)) {
        return {
          isValid: false,
          error: `Line ${lineNum}: link_sub_issue 'parent_issue_number' and 'sub_issue_number' must be different`,
        };
      }
      return null;
    },
  },
};

/**
 * Get the default max count for a type
 * @param {string} itemType - The safe output type
 * @param {Object} [config] - Configuration override from safe-outputs config
 * @returns {number} The max allowed count
 */
function getMaxAllowedForType(itemType, config) {
  const itemConfig = config?.[itemType];
  if (itemConfig && typeof itemConfig === "object" && "max" in itemConfig && itemConfig.max) {
    return itemConfig.max;
  }
  const typeConfig = VALIDATION_CONFIG[itemType];
  return typeConfig?.defaultMax ?? 1;
}

/**
 * Get the minimum required count for a type
 * @param {string} itemType - The safe output type
 * @param {Object} [config] - Configuration from safe-outputs config
 * @returns {number} The minimum required count
 */
function getMinRequiredForType(itemType, config) {
  const itemConfig = config?.[itemType];
  if (itemConfig && typeof itemConfig === "object" && "min" in itemConfig && itemConfig.min) {
    return itemConfig.min;
  }
  return 0;
}

/**
 * Validate a positive integer field
 * @param {any} value - Value to validate
 * @param {string} fieldName - Field name for error messages
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, normalizedValue?: number, error?: string}}
 */
function validatePositiveInteger(value, fieldName, lineNum) {
  if (value === undefined || value === null) {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} is required`,
    };
  }
  if (typeof value !== "number" && typeof value !== "string") {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a number or string`,
    };
  }
  const parsed = typeof value === "string" ? parseInt(value, 10) : value;
  if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a valid positive integer (got: ${value})`,
    };
  }
  return { isValid: true, normalizedValue: parsed };
}

/**
 * Validate an optional positive integer field
 * @param {any} value - Value to validate
 * @param {string} fieldName - Field name for error messages
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, normalizedValue?: number, error?: string}}
 */
function validateOptionalPositiveInteger(value, fieldName, lineNum) {
  if (value === undefined) {
    return { isValid: true };
  }
  if (typeof value !== "number" && typeof value !== "string") {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a number or string`,
    };
  }
  const parsed = typeof value === "string" ? parseInt(value, 10) : value;
  if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a valid positive integer (got: ${value})`,
    };
  }
  return { isValid: true, normalizedValue: parsed };
}

/**
 * Validate an issue/PR number field (optional, accepts number or string)
 * @param {any} value - Value to validate
 * @param {string} fieldName - Field name for error messages
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, error?: string}}
 */
function validateIssueOrPRNumber(value, fieldName, lineNum) {
  if (value === undefined) {
    return { isValid: true };
  }
  if (typeof value !== "number" && typeof value !== "string") {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a number or string`,
    };
  }
  return { isValid: true };
}

/**
 * Validate a value that can be either a positive integer (issue number) or a temporary ID.
 * @param {any} value - The value to validate
 * @param {string} fieldName - Name of the field for error messages
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, normalizedValue?: number|string, isTemporary?: boolean, error?: string}}
 */
function validateIssueNumberOrTemporaryId(value, fieldName, lineNum) {
  if (value === undefined || value === null) {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} is required`,
    };
  }
  if (typeof value !== "number" && typeof value !== "string") {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a number or string`,
    };
  }
  // Check if it's a temporary ID
  if (isTemporaryId(value)) {
    return { isValid: true, normalizedValue: String(value).toLowerCase(), isTemporary: true };
  }
  // Try to parse as positive integer
  const parsed = typeof value === "string" ? parseInt(value, 10) : value;
  if (isNaN(parsed) || parsed <= 0 || !Number.isInteger(parsed)) {
    return {
      isValid: false,
      error: `Line ${lineNum}: ${fieldName} must be a positive integer or temporary ID (got: ${value})`,
    };
  }
  return { isValid: true, normalizedValue: parsed, isTemporary: false };
}

/**
 * Validate a single field based on its validation configuration
 * @param {any} value - The field value
 * @param {string} fieldName - The field name
 * @param {FieldValidation} validation - The validation configuration
 * @param {string} itemType - The item type for error messages
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, normalizedValue?: any, error?: string}}
 */
function validateField(value, fieldName, validation, itemType, lineNum) {
  // For positiveInteger fields, delegate required check to validatePositiveInteger
  if (validation.positiveInteger) {
    return validatePositiveInteger(value, `${itemType} '${fieldName}'`, lineNum);
  }

  // For issueNumberOrTemporaryId fields, delegate required check to validateIssueNumberOrTemporaryId
  if (validation.issueNumberOrTemporaryId) {
    return validateIssueNumberOrTemporaryId(value, `${itemType} '${fieldName}'`, lineNum);
  }

  // Handle required check for other fields
  if (validation.required && (value === undefined || value === null)) {
    const fieldType = validation.type || "string";
    return {
      isValid: false,
      error: `Line ${lineNum}: ${itemType} requires a '${fieldName}' field (${fieldType})`,
    };
  }

  // If not required and not present, skip other validations
  if (value === undefined || value === null) {
    return { isValid: true };
  }

  // Handle optionalPositiveInteger validation
  if (validation.optionalPositiveInteger) {
    return validateOptionalPositiveInteger(value, `${itemType} '${fieldName}'`, lineNum);
  }

  // Handle issueOrPRNumber validation
  if (validation.issueOrPRNumber) {
    return validateIssueOrPRNumber(value, `${itemType} '${fieldName}'`, lineNum);
  }

  // Handle type validation
  if (validation.type === "string") {
    if (typeof value !== "string") {
      // For required fields, use "requires a" format for both missing and wrong type
      if (validation.required) {
        return {
          isValid: false,
          error: `Line ${lineNum}: ${itemType} requires a '${fieldName}' field (string)`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${itemType} '${fieldName}' must be a string`,
      };
    }

    // Handle pattern validation
    if (validation.pattern) {
      const regex = new RegExp(validation.pattern);
      if (!regex.test(value.trim())) {
        const errorMsg = validation.patternError || `must match pattern ${validation.pattern}`;
        return {
          isValid: false,
          error: `Line ${lineNum}: ${itemType} '${fieldName}' ${errorMsg}`,
        };
      }
    }

    // Handle enum validation
    if (validation.enum) {
      const normalizedValue = value.toLowerCase ? value.toLowerCase() : value;
      const normalizedEnum = validation.enum.map(e => (e.toLowerCase ? e.toLowerCase() : e));
      if (!normalizedEnum.includes(normalizedValue)) {
        // Use special format for 2-option enums: "'field' must be 'A' or 'B'"
        // Use standard format for more options: "'field' must be one of: A, B, C"
        let errorMsg;
        if (validation.enum.length === 2) {
          errorMsg = `Line ${lineNum}: ${itemType} '${fieldName}' must be '${validation.enum[0]}' or '${validation.enum[1]}'`;
        } else {
          errorMsg = `Line ${lineNum}: ${itemType} '${fieldName}' must be one of: ${validation.enum.join(", ")}`;
        }
        return {
          isValid: false,
          error: errorMsg,
        };
      }
      // Return the properly cased enum value if there's a case difference
      const matchIndex = normalizedEnum.indexOf(normalizedValue);
      let normalizedResult = validation.enum[matchIndex];
      // Apply sanitization if configured
      if (validation.sanitize && validation.maxLength) {
        normalizedResult = sanitizeContent(normalizedResult, validation.maxLength);
      }
      return { isValid: true, normalizedValue: normalizedResult };
    }

    // Handle sanitization
    if (validation.sanitize) {
      const sanitized = sanitizeContent(value, validation.maxLength || MAX_BODY_LENGTH);
      return { isValid: true, normalizedValue: sanitized };
    }

    return { isValid: true, normalizedValue: value };
  }

  if (validation.type === "array") {
    if (!Array.isArray(value)) {
      // For required fields, use "requires a" format for both missing and wrong type
      if (validation.required) {
        return {
          isValid: false,
          error: `Line ${lineNum}: ${itemType} requires a '${fieldName}' field (array)`,
        };
      }
      return {
        isValid: false,
        error: `Line ${lineNum}: ${itemType} '${fieldName}' must be an array`,
      };
    }

    // Validate array items
    if (validation.itemType === "string") {
      const hasInvalidItem = value.some(item => typeof item !== "string");
      if (hasInvalidItem) {
        return {
          isValid: false,
          error: `Line ${lineNum}: ${itemType} ${fieldName} array must contain only strings`,
        };
      }

      // Sanitize items if configured
      if (validation.itemSanitize) {
        const sanitizedItems = value.map(item =>
          typeof item === "string" ? sanitizeContent(item, validation.itemMaxLength || 128) : item
        );
        return { isValid: true, normalizedValue: sanitizedItems };
      }
    }

    return { isValid: true, normalizedValue: value };
  }

  if (validation.type === "boolean") {
    if (typeof value !== "boolean") {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${itemType} '${fieldName}' must be a boolean`,
      };
    }
    return { isValid: true, normalizedValue: value };
  }

  if (validation.type === "number") {
    if (typeof value !== "number") {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${itemType} '${fieldName}' must be a number`,
      };
    }
    return { isValid: true, normalizedValue: value };
  }

  // No specific type validation, return as-is
  return { isValid: true, normalizedValue: value };
}

/**
 * Validate a safe output item against its type configuration
 * @param {Object} item - The item to validate
 * @param {string} itemType - The item type (e.g., "create_issue")
 * @param {number} lineNum - Line number for error messages
 * @returns {{isValid: boolean, normalizedItem?: Object, error?: string}}
 */
function validateItem(item, itemType, lineNum) {
  const typeConfig = VALIDATION_CONFIG[itemType];

  if (!typeConfig) {
    // Unknown type - let the caller handle this
    return { isValid: true, normalizedItem: item };
  }

  const normalizedItem = { ...item };
  const errors = [];

  // Run custom validation first if defined
  if (typeConfig.customValidation) {
    const customResult = typeConfig.customValidation(item, lineNum);
    if (customResult && !customResult.isValid) {
      return customResult;
    }
  }

  // Validate each configured field
  for (const [fieldName, validation] of Object.entries(typeConfig.fields)) {
    const fieldValue = item[fieldName];
    const result = validateField(fieldValue, fieldName, validation, itemType, lineNum);

    if (!result.isValid) {
      errors.push(result.error);
    } else if (result.normalizedValue !== undefined) {
      normalizedItem[fieldName] = result.normalizedValue;
    }
  }

  if (errors.length > 0) {
    return { isValid: false, error: errors[0] }; // Return first error
  }

  return { isValid: true, normalizedItem };
}

/**
 * Check if a type has validation configuration
 * @param {string} itemType - The item type
 * @returns {boolean}
 */
function hasValidationConfig(itemType) {
  return itemType in VALIDATION_CONFIG;
}

/**
 * Get the validation configuration for a type
 * @param {string} itemType - The item type
 * @returns {TypeValidationConfig|undefined}
 */
function getValidationConfig(itemType) {
  return VALIDATION_CONFIG[itemType];
}

/**
 * Get all known safe output types
 * @returns {string[]}
 */
function getKnownTypes() {
  return Object.keys(VALIDATION_CONFIG);
}

module.exports = {
  // Main validation functions
  validateItem,
  validateField,
  validatePositiveInteger,
  validateOptionalPositiveInteger,
  validateIssueOrPRNumber,
  validateIssueNumberOrTemporaryId,

  // Configuration accessors
  getMaxAllowedForType,
  getMinRequiredForType,
  hasValidationConfig,
  getValidationConfig,
  getKnownTypes,

  // Constants
  MAX_BODY_LENGTH,
  MAX_GITHUB_USERNAME_LENGTH,
  VALIDATION_CONFIG,
};
