// @ts-check

const ALLOWED_KEYWORDS = new Set(["type", "enum", "required", "properties", "additionalProperties", "items", "oneOf", "anyOf"]);
const ALLOWED_TYPES = new Set(["object", "array", "string", "number", "integer", "boolean", "null"]);
const MAX_SCHEMA_DEPTH = 32;

/**
 * @param {unknown} schema
 * @param {string} [label]
 */
function validateSchemaContract(schema, label = "Ledger") {
  const isLedger = label === "Ledger";
  const kind = label.toLowerCase();
  const invalid = message => {
    throw new TypeError(`${label} ${message}`);
  };
  function check(node, depth = 0) {
    if (!node || typeof node !== "object" || Array.isArray(node) || depth > MAX_SCHEMA_DEPTH) invalid("schema must contain bounded objects");
    const entries = Object.entries(/** @type {Record<string, unknown>} */ node);
    if (("oneOf" in node || "anyOf" in node) && entries.length !== 1) {
      throw new TypeError(`${label} schema alternatives cannot combine with ignored constraints`);
    }
    for (const [key, value] of entries) {
      if (!ALLOWED_KEYWORDS.has(key)) throw new TypeError(`Unsupported ${kind} schema keyword: ${key}`);
      if (key === "type") {
        const types = Array.isArray(value) ? value : [value];
        if (!types.length || !types.every(type => typeof type === "string" && ALLOWED_TYPES.has(type))) {
          throw new TypeError(`Invalid ${kind} schema type`);
        }
      } else if (key === "enum") {
        if (!Array.isArray(value) || !value.length || !value.every(item => item === null || (["string", "number", "boolean"].includes(typeof item) && (typeof item !== "number" || Number.isFinite(item))))) {
          throw new TypeError(`${label} schema enum supports only primitive JSON values`);
        }
        if (!isLedger && value.some(item => typeof item === "number" && Number.isInteger(item) && !Number.isSafeInteger(item))) {
          throw new TypeError(`${label} schema enum integers must be exactly representable`);
        }
      } else if (key === "required") {
        if (!Array.isArray(value) || !value.every(item => typeof item === "string")) {
          throw new TypeError(`Invalid ${kind} schema required fields`);
        }
      } else if (key === "additionalProperties") {
        if (value !== false) throw new TypeError(isLedger ? "Only additionalProperties: false is supported" : `${label} schema supports only additionalProperties: false`);
      } else if (key === "properties") {
        if (!value || typeof value !== "object" || Array.isArray(value)) {
          throw new TypeError(`Invalid ${kind} schema properties`);
        }
        Object.values(value).forEach(child => check(child, depth + 1));
      } else if (key === "items") {
        check(value, depth + 1);
      } else if (key === "oneOf" || key === "anyOf") {
        if (!Array.isArray(value) || value.length === 0) {
          throw new TypeError(`Invalid ${kind} schema alternatives`);
        }
        value.forEach(child => check(child, depth + 1));
      }
    }
  }
  check(schema);
}

module.exports = { validateSchemaContract, MAX_SCHEMA_DEPTH };
