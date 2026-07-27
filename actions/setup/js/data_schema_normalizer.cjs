// @ts-check

const SUPPORTED_TYPES = new Set(["object", "array", "string", "number", "integer", "boolean"]);
const ALLOWED_KEYS = new Set(["type", "description", "properties", "required", "items", "enum", "additionalProperties", "minLength", "maxLength", "minimum", "maximum", "pattern"]);

function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function hasSchemaKeyword(value) {
  return Object.keys(value).some(key => ALLOWED_KEYS.has(key));
}

/**
 * @param {any} raw
 * @param {string} path
 * @param {boolean} allowShorthand
 * @returns {Record<string, any>}
 */
function simplifySchemaNode(raw, path, allowShorthand) {
  if (typeof raw === "string") {
    if (!allowShorthand) {
      throw new Error(`${path}: string shorthand is not allowed here`);
    }
    if (!SUPPORTED_TYPES.has(raw)) {
      throw new Error(`${path}: unsupported type "${raw}"`);
    }
    return { type: raw };
  }

  if (!isPlainObject(raw)) {
    throw new Error(`${path}: expected an object schema`);
  }

  let node = raw;
  let explicit = hasSchemaKeyword(node);
  if (!explicit && allowShorthand) {
    explicit = true;
    node = { type: "object", properties: node };
  }
  if (!explicit) {
    throw new Error(`${path}: expected JSON schema keywords or shorthand properties`);
  }

  for (const key of Object.keys(node)) {
    if (!ALLOWED_KEYS.has(key)) {
      throw new Error(`${path}: unsupported keyword "${key}"`);
    }
  }

  const result = {};
  let typeName = typeof node.type === "string" ? node.type : "";
  if (!typeName) {
    if (node.properties !== undefined || node.required !== undefined || node.additionalProperties !== undefined) {
      typeName = "object";
    } else if (node.items !== undefined) {
      typeName = "array";
    }
  }
  if (typeName) {
    if (!SUPPORTED_TYPES.has(typeName)) {
      throw new Error(`${path}.type: unsupported type "${typeName}"`);
    }
    result.type = typeName;
  }

  if (node.description !== undefined) {
    if (typeof node.description !== "string") {
      throw new Error(`${path}.description: must be a string`);
    }
    result.description = node.description;
  }

  if (node.enum !== undefined) {
    if (!Array.isArray(node.enum) || node.enum.length === 0) {
      throw new Error(`${path}.enum: must be a non-empty array`);
    }
    for (let i = 0; i < node.enum.length; i++) {
      const enumItem = node.enum[i];
      if (typeof enumItem !== "string" && typeof enumItem !== "number" && typeof enumItem !== "boolean") {
        throw new Error(`${path}.enum[${i}]: must be a scalar value`);
      }
    }
    result.enum = node.enum;
  }

  if (typeName === "object") {
    if (!isPlainObject(node.properties)) {
      throw new Error(`${path}.properties: is required for object schemas`);
    }
    const normalizedProperties = {};
    for (const [key, value] of Object.entries(node.properties)) {
      normalizedProperties[key] = simplifySchemaNode(value, `${path}.properties.${key}`, true);
    }
    result.properties = normalizedProperties;

    const requiredSet = new Set(Object.keys(normalizedProperties));
    if (node.required !== undefined) {
      if (!Array.isArray(node.required)) {
        throw new Error(`${path}.required: must be an array of strings`);
      }
      for (let i = 0; i < node.required.length; i++) {
        const requiredName = node.required[i];
        if (typeof requiredName !== "string" || requiredName.trim().length === 0) {
          throw new Error(`${path}.required[${i}]: must be a non-empty string`);
        }
        if (!Object.prototype.hasOwnProperty.call(normalizedProperties, requiredName)) {
          throw new Error(`${path}.required[${i}]: unknown property "${requiredName}"`);
        }
        requiredSet.add(requiredName);
      }
    }
    result.required = [...requiredSet].sort();

    if (node.additionalProperties !== undefined) {
      if (typeof node.additionalProperties !== "boolean") {
        throw new Error(`${path}.additionalProperties: must be boolean`);
      }
      if (node.additionalProperties) {
        throw new Error(`${path}.additionalProperties: must be false for OpenAI Codex structured outputs compatibility`);
      }
    }
    result.additionalProperties = false;
  } else if (typeName === "array") {
    if (node.items === undefined) {
      throw new Error(`${path}.items: is required for array schemas`);
    }
    result.items = simplifySchemaNode(node.items, `${path}.items`, true);
  } else if (typeName === "string") {
    if (node.minLength !== undefined) result.minLength = node.minLength;
    if (node.maxLength !== undefined) result.maxLength = node.maxLength;
    if (node.pattern !== undefined) result.pattern = node.pattern;
  } else if (typeName === "number" || typeName === "integer") {
    if (node.minimum !== undefined) result.minimum = node.minimum;
    if (node.maximum !== undefined) result.maximum = node.maximum;
  }

  return result;
}

/**
 * @param {any} rawSchema
 * @param {string} path
 * @returns {Record<string, any>}
 */
function resolveDataSchema(rawSchema, path) {
  if (isPlainObject(rawSchema)) {
    const normalized = simplifySchemaNode(rawSchema, path, true);
    if (normalized.type !== "object") {
      throw new Error(`${path}: must resolve to an object schema`);
    }
    return normalized;
  }
  if (typeof rawSchema === "string") {
    const parsed = JSON.parse(rawSchema);
    if (!isPlainObject(parsed)) {
      throw new Error(`${path}: string JSON must decode to an object schema`);
    }
    const normalized = simplifySchemaNode(parsed, path, true);
    if (normalized.type !== "object") {
      throw new Error(`${path}: must resolve to an object schema`);
    }
    return normalized;
  }
  throw new Error(`${path}: must be an object schema or JSON string`);
}

module.exports = {
  resolveDataSchema,
};
