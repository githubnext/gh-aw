// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  const fs = require("fs");
  const { sanitizeContent } = require("./sanitize_content.cjs");
  const {
    validateItem,
    getMaxAllowedForType,
    getMinRequiredForType,
    hasValidationConfig,
    MAX_BODY_LENGTH: maxBodyLength,
  } = require("./safe_output_type_validator.cjs");

  function repairJson(jsonStr) {
    let repaired = jsonStr.trim();
    const _ctrl = { 8: "\\b", 9: "\\t", 10: "\\n", 12: "\\f", 13: "\\r" };
    repaired = repaired.replace(/[\u0000-\u001F]/g, ch => {
      const c = ch.charCodeAt(0);
      return _ctrl[c] || "\\u" + c.toString(16).padStart(4, "0");
    });
    repaired = repaired.replace(/'/g, '"');
    repaired = repaired.replace(/([{,]\s*)([a-zA-Z_$][a-zA-Z0-9_$]*)\s*:/g, '$1"$2":');
    repaired = repaired.replace(/"([^"\\]*)"/g, (match, content) => {
      if (content.includes("\n") || content.includes("\r") || content.includes("\t")) {
        const escaped = content.replace(/\\/g, "\\\\").replace(/\n/g, "\\n").replace(/\r/g, "\\r").replace(/\t/g, "\\t");
        return `"${escaped}"`;
      }
      return match;
    });
    repaired = repaired.replace(/"([^"]*)"([^":,}\]]*)"([^"]*)"(\s*[,:}\]])/g, (match, p1, p2, p3, p4) => `"${p1}\\"${p2}\\"${p3}"${p4}`);
    repaired = repaired.replace(/(\[\s*(?:"[^"]*"(?:\s*,\s*"[^"]*")*\s*),?)\s*}/g, "$1]");
    const openBraces = (repaired.match(/\{/g) || []).length;
    const closeBraces = (repaired.match(/\}/g) || []).length;
    if (openBraces > closeBraces) {
      repaired += "}".repeat(openBraces - closeBraces);
    } else if (closeBraces > openBraces) {
      repaired = "{".repeat(closeBraces - openBraces) + repaired;
    }
    const openBrackets = (repaired.match(/\[/g) || []).length;
    const closeBrackets = (repaired.match(/\]/g) || []).length;
    if (openBrackets > closeBrackets) {
      repaired += "]".repeat(openBrackets - closeBrackets);
    } else if (closeBrackets > openBrackets) {
      repaired = "[".repeat(closeBrackets - openBrackets) + repaired;
    }
    repaired = repaired.replace(/,(\s*[}\]])/g, "$1");
    return repaired;
  }

  function validateFieldWithInputSchema(value, fieldName, inputSchema, lineNum) {
    if (inputSchema.required && (value === undefined || value === null)) {
      return {
        isValid: false,
        error: `Line ${lineNum}: ${fieldName} is required`,
      };
    }
    if (value === undefined || value === null) {
      return {
        isValid: true,
        normalizedValue: inputSchema.default || undefined,
      };
    }
    const inputType = inputSchema.type || "string";
    let normalizedValue = value;
    switch (inputType) {
      case "string":
        if (typeof value !== "string") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a string`,
          };
        }
        normalizedValue = sanitizeContent(value);
        break;
      case "boolean":
        if (typeof value !== "boolean") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a boolean`,
          };
        }
        break;
      case "number":
        if (typeof value !== "number") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a number`,
          };
        }
        break;
      case "choice":
        if (typeof value !== "string") {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be a string for choice type`,
          };
        }
        if (inputSchema.options && !inputSchema.options.includes(value)) {
          return {
            isValid: false,
            error: `Line ${lineNum}: ${fieldName} must be one of: ${inputSchema.options.join(", ")}`,
          };
        }
        normalizedValue = sanitizeContent(value);
        break;
      default:
        if (typeof value === "string") {
          normalizedValue = sanitizeContent(value);
        }
        break;
    }
    return {
      isValid: true,
      normalizedValue,
    };
  }
  function validateItemWithSafeJobConfig(item, jobConfig, lineNum) {
    const errors = [];
    const normalizedItem = { ...item };
    if (!jobConfig.inputs) {
      return {
        isValid: true,
        errors: [],
        normalizedItem: item,
      };
    }
    for (const [fieldName, inputSchema] of Object.entries(jobConfig.inputs)) {
      const fieldValue = item[fieldName];
      const validation = validateFieldWithInputSchema(fieldValue, fieldName, inputSchema, lineNum);
      if (!validation.isValid && validation.error) {
        errors.push(validation.error);
      } else if (validation.normalizedValue !== undefined) {
        normalizedItem[fieldName] = validation.normalizedValue;
      }
    }
    return {
      isValid: errors.length === 0,
      errors,
      normalizedItem,
    };
  }
  function parseJsonWithRepair(jsonStr) {
    try {
      return JSON.parse(jsonStr);
    } catch (originalError) {
      try {
        const repairedJson = repairJson(jsonStr);
        return JSON.parse(repairedJson);
      } catch (repairError) {
        core.info(`invalid input json: ${jsonStr}`);
        const originalMsg = originalError instanceof Error ? originalError.message : String(originalError);
        const repairMsg = repairError instanceof Error ? repairError.message : String(repairError);
        throw new Error(`JSON parsing failed. Original: ${originalMsg}. After attempted repair: ${repairMsg}`);
      }
    }
  }
  const outputFile = process.env.GH_AW_SAFE_OUTPUTS;
  // Read config from file instead of environment variable
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || "/tmp/gh-aw/safeoutputs/config.json";
  let safeOutputsConfig;
  try {
    if (fs.existsSync(configPath)) {
      const configFileContent = fs.readFileSync(configPath, "utf8");
      safeOutputsConfig = JSON.parse(configFileContent);
    }
  } catch (error) {
    core.warning(`Failed to read config file from ${configPath}: ${error instanceof Error ? error.message : String(error)}`);
  }

  if (!outputFile) {
    core.info("GH_AW_SAFE_OUTPUTS not set, no output to collect");
    core.setOutput("output", "");
    return;
  }
  if (!fs.existsSync(outputFile)) {
    core.info(`Output file does not exist: ${outputFile}`);
    core.setOutput("output", "");
    return;
  }
  const outputContent = fs.readFileSync(outputFile, "utf8");
  if (outputContent.trim() === "") {
    core.info("Output file is empty");
  }
  core.info(`Raw output content length: ${outputContent.length}`);
  let expectedOutputTypes = {};
  if (safeOutputsConfig) {
    try {
      // safeOutputsConfig is already a parsed object from the file
      // Normalize all config keys to use underscores instead of dashes
      expectedOutputTypes = Object.fromEntries(Object.entries(safeOutputsConfig).map(([key, value]) => [key.replace(/-/g, "_"), value]));
      core.info(`Expected output types: ${JSON.stringify(Object.keys(expectedOutputTypes))}`);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      core.info(`Warning: Could not parse safe-outputs config: ${errorMsg}`);
    }
  }
  const lines = outputContent.trim().split("\n");
  const parsedItems = [];
  const errors = [];
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line === "") continue;
    try {
      const item = parseJsonWithRepair(line);
      if (item === undefined) {
        errors.push(`Line ${i + 1}: Invalid JSON - JSON parsing failed`);
        continue;
      }
      if (!item.type) {
        errors.push(`Line ${i + 1}: Missing required 'type' field`);
        continue;
      }
      // Normalize type to use underscores (convert any dashes to underscores for resilience)
      const itemType = item.type.replace(/-/g, "_");
      // Update item.type to normalized value
      item.type = itemType;
      if (!expectedOutputTypes[itemType]) {
        errors.push(`Line ${i + 1}: Unexpected output type '${itemType}'. Expected one of: ${Object.keys(expectedOutputTypes).join(", ")}`);
        continue;
      }
      const typeCount = parsedItems.filter(existing => existing.type === itemType).length;
      const maxAllowed = getMaxAllowedForType(itemType, expectedOutputTypes);
      if (typeCount >= maxAllowed) {
        errors.push(`Line ${i + 1}: Too many items of type '${itemType}'. Maximum allowed: ${maxAllowed}.`);
        continue;
      }
      core.info(`Line ${i + 1}: type '${itemType}'`);

      // Use the validation engine to validate the item
      if (hasValidationConfig(itemType)) {
        const validationResult = validateItem(item, itemType, i + 1);
        if (!validationResult.isValid) {
          if (validationResult.error) {
            errors.push(validationResult.error);
          }
          continue;
        }
        // Update item with normalized values
        Object.assign(item, validationResult.normalizedItem);
      } else {
        // Fall back to validateItemWithSafeJobConfig for unknown types
        const jobOutputType = expectedOutputTypes[itemType];
        if (!jobOutputType) {
          errors.push(`Line ${i + 1}: Unknown output type '${itemType}'`);
          continue;
        }
        const safeJobConfig = jobOutputType;
        if (safeJobConfig && safeJobConfig.inputs) {
          const validation = validateItemWithSafeJobConfig(item, safeJobConfig, i + 1);
          if (!validation.isValid) {
            errors.push(...validation.errors);
            continue;
          }
          Object.assign(item, validation.normalizedItem);
        }
      }

      core.info(`Line ${i + 1}: Valid ${itemType} item`);
      parsedItems.push(item);
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      errors.push(`Line ${i + 1}: Invalid JSON - ${errorMsg}`);
    }
  }
  if (errors.length > 0) {
    core.warning("Validation errors found:");
    errors.forEach(error => core.warning(`  - ${error}`));
    if (parsedItems.length === 0) {
      core.setFailed(errors.map(e => `  - ${e}`).join("\n"));
      return;
    }
  }
  for (const itemType of Object.keys(expectedOutputTypes)) {
    const minRequired = getMinRequiredForType(itemType, expectedOutputTypes);
    if (minRequired > 0) {
      const actualCount = parsedItems.filter(item => item.type === itemType).length;
      if (actualCount < minRequired) {
        errors.push(`Too few items of type '${itemType}'. Minimum required: ${minRequired}, found: ${actualCount}.`);
      }
    }
  }
  core.info(`Successfully parsed ${parsedItems.length} valid output items`);
  const validatedOutput = {
    items: parsedItems,
    errors: errors,
  };
  const agentOutputFile = "/tmp/gh-aw/agent_output.json";
  const validatedOutputJson = JSON.stringify(validatedOutput);
  try {
    fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    fs.writeFileSync(agentOutputFile, validatedOutputJson, "utf8");
    core.info(`Stored validated output to: ${agentOutputFile}`);
    core.exportVariable("GH_AW_AGENT_OUTPUT", agentOutputFile);
  } catch (error) {
    const errorMsg = error instanceof Error ? error.message : String(error);
    core.error(`Failed to write agent output file: ${errorMsg}`);
  }
  core.setOutput("output", JSON.stringify(validatedOutput));
  core.setOutput("raw_output", outputContent);
  const outputTypes = Array.from(new Set(parsedItems.map(item => item.type)));
  core.info(`output_types: ${outputTypes.join(", ")}`);
  core.setOutput("output_types", outputTypes.join(","));

  // Check if patch file exists for detection job conditional
  const patchPath = "/tmp/gh-aw/aw.patch";
  const hasPatch = fs.existsSync(patchPath);
  core.info(`Patch file ${hasPatch ? "exists" : "does not exist"} at: ${patchPath}`);
  core.setOutput("has_patch", hasPatch ? "true" : "false");
}
await main();
