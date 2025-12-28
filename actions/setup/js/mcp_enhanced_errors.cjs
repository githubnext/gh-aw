// @ts-check

/**
 * MCP Enhanced Error Messages
 *
 * This module provides enhanced error messages for MCP tool validation errors
 * that include actionable guidance to help agents self-correct.
 */

/**
 * Generate an enhanced error message with actionable guidance for missing parameters
 * @param {string[]} missingFields - Array of missing field names
 * @param {string} toolName - Name of the tool that failed validation
 * @param {Object} inputSchema - The input schema for the tool
 * @returns {string} Enhanced error message with guidance and example
 */
function generateEnhancedErrorMessage(missingFields, toolName, inputSchema) {
  if (!missingFields || missingFields.length === 0) {
    return "Invalid arguments";
  }

  // Base error message
  const fieldsList = missingFields.map(m => `'${m}'`).join(", ");
  let message = `Invalid arguments: missing or empty ${fieldsList}\n`;

  // Add guidance for each missing field
  if (inputSchema && inputSchema.properties) {
    for (const field of missingFields) {
      const fieldSchema = inputSchema.properties[field];
      if (fieldSchema && fieldSchema.description) {
        message += `\nRequired parameter '${field}': ${fieldSchema.description}`;
      }
    }
  }

  // Add example usage based on tool schema
  message += "\n\nExample:";
  const example = generateExample(toolName, inputSchema);
  message += `\n${example}`;

  return message;
}

/**
 * Generate an example JSON object for a tool based on its schema
 * @param {string} toolName - Name of the tool
 * @param {Object} inputSchema - The input schema for the tool
 * @returns {string} JSON example string
 */
function generateExample(toolName, inputSchema) {
  if (!inputSchema || !inputSchema.properties) {
    return "{}";
  }

  const example = {};
  const requiredFields = inputSchema.required || [];

  // Add required fields to example
  for (const field of requiredFields) {
    const fieldSchema = inputSchema.properties[field];
    example[field] = getExampleValue(field, fieldSchema);
  }

  // Add one optional field if present (to show it's optional)
  const optionalFields = Object.keys(inputSchema.properties).filter(f => !requiredFields.includes(f));
  if (optionalFields.length > 0) {
    const firstOptional = optionalFields[0];
    const fieldSchema = inputSchema.properties[firstOptional];
    example[firstOptional] = getExampleValue(firstOptional, fieldSchema);
  }

  return JSON.stringify(example, null, 2);
}

/**
 * Generate an example value for a field based on its schema
 * @param {string} fieldName - Name of the field
 * @param {Object} fieldSchema - Schema for the field
 * @returns {*} Example value
 */
function getExampleValue(fieldName, fieldSchema) {
  if (!fieldSchema) {
    return "value";
  }

  // Handle array types
  if (fieldSchema.type === "array") {
    if (fieldName === "labels") {
      return ["bug", "enhancement"];
    }
    if (fieldName === "reviewers" || fieldName === "assignees") {
      return ["octocat"];
    }
    return ["example"];
  }

  // Handle number types
  if (fieldSchema.type === "number" || (Array.isArray(fieldSchema.type) && fieldSchema.type.includes("number"))) {
    if (fieldName.includes("number") || fieldName === "line") {
      return 123;
    }
    return 42;
  }

  // Handle enum types
  if (fieldSchema.enum && fieldSchema.enum.length > 0) {
    return fieldSchema.enum[0];
  }

  // Handle string types with specific field names
  if (fieldName === "title") {
    return "Issue title";
  }
  if (fieldName === "body") {
    return "Your comment or description text";
  }
  if (fieldName === "message") {
    return "Commit message or status message";
  }
  if (fieldName === "path") {
    return "src/file.js";
  }
  if (fieldName === "branch") {
    return "feature-branch";
  }
  if (fieldName === "tag") {
    return "v1.0.0";
  }

  // Default to string type
  return "example value";
}

module.exports = {
  generateEnhancedErrorMessage,
  generateExample,
  getExampleValue,
};
