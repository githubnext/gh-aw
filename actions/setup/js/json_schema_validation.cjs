// @ts-check

const { validateValueAgainstSchema } = require("./schema-validation.cjs");
const { validateSchemaContract, MAX_SCHEMA_DEPTH } = require("./memory_schema_contract.cjs");

/**
 * @param {string} text Already-decoded valid JSON.
 * @returns {Map<string, string>}
 */
function collectRawJSONNumbers(text) {
  const tokens = new Map();
  const numberPattern = /-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/y;
  let index = 0;
  const skipWhitespace = () => {
    while (/\s/.test(text[index] || "")) index++;
  };
  const parseString = () => {
    const start = index++;
    let escaped = false;
    while (index < text.length) {
      const character = text[index++];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') return JSON.parse(text.slice(start, index));
    }
    return "";
  };
  /** @param {(string | number)[]} path */
  const parseValue = path => {
    skipWhitespace();
    const character = text[index];
    if ((character === "{" || character === "[") && path.length >= MAX_SCHEMA_DEPTH) {
      // No bounded schema can reach deeper values; skip without recursive calls.
      let depth = 0;
      let quoted = false;
      do {
        const token = text[index++];
        if (quoted) {
          if (token === "\\") index++;
          else if (token === '"') quoted = false;
        } else if (token === '"') quoted = true;
        else if (token === "{" || token === "[") depth++;
        else if (token === "}" || token === "]") depth--;
      } while (depth > 0);
      return;
    }
    if (character === "{") {
      index++;
      skipWhitespace();
      while (text[index] !== "}") {
        skipWhitespace();
        const key = parseString();
        skipWhitespace();
        index++;
        parseValue([...path, key]);
        skipWhitespace();
        if (text[index] !== ",") break;
        index++;
      }
      index++;
    } else if (character === "[") {
      index++;
      skipWhitespace();
      let itemIndex = 0;
      while (text[index] !== "]") {
        parseValue([...path, itemIndex++]);
        skipWhitespace();
        if (text[index] !== ",") break;
        index++;
      }
      index++;
    } else if (character === '"') {
      parseString();
    } else if (character === "t" || character === "f" || character === "n") {
      index += character === "t" ? 4 : character === "f" ? 5 : 4;
    } else {
      numberPattern.lastIndex = index;
      const match = numberPattern.exec(text);
      if (!match) return;
      tokens.set(JSON.stringify(path), match[0]);
      index = numberPattern.lastIndex;
    }
  };
  parseValue([]);
  return tokens;
}

/** @param {string} token */
function canonicalDecimal(token) {
  const match = /^(-?)(\d+)(?:\.(\d+))?(?:[eE]([+-]?\d+))?$/.exec(token);
  if (!match) return token;
  const sign = match[1];
  let digits = `${match[2]}${match[3] || ""}`.replace(/^0+/, "");
  if (!digits) return "0";
  const trailingZeros = digits.length - digits.replace(/0+$/, "").length;
  digits = digits.slice(0, digits.length - trailingZeros);
  const exponent = BigInt(match[4] || "0") - BigInt((match[3] || "").length) + BigInt(trailingZeros);
  return `${sign}${digits}e${exponent}`;
}

/** @param {Record<string, any>} schema */
function hasNumericEnum(schema) {
  return (
    (Array.isArray(schema.enum) && schema.enum.some(item => typeof item === "number")) ||
    Object.values(schema.properties || {}).some(hasNumericEnum) ||
    (schema.items !== undefined && hasNumericEnum(schema.items)) ||
    (schema.oneOf || schema.anyOf || []).some(hasNumericEnum)
  );
}

/**
 * Prepare the bounded schema once per input. Token exactness is needed only for
 * numeric enums; ordinary JSON numbers retain normal JSON.parse semantics.
 * @param {Record<string, any>} schema
 */
function prepareJsonSchema(schema) {
  validateSchemaContract(schema, "JSON");
  const checkNumbers = hasNumericEnum(schema);
  /** @param {any} value @param {string} text */
  return (value, text) => {
    if (checkNumbers) {
      const tokens = collectRawJSONNumbers(text);
      /**
       * @param {any} currentValue
       * @param {Record<string, any>} currentSchema
       * @param {(string | number)[]} currentPath
       * @returns {{path: string, message: string}|undefined}
       */
      const visit = (currentValue, currentSchema, currentPath) => {
        if (Array.isArray(currentSchema.enum) && currentSchema.enum.some(item => typeof item === "number") && typeof currentValue === "number") {
          const token = tokens.get(JSON.stringify(currentPath));
          if (token !== undefined && (token.length > 256 || !Number.isFinite(Number(token)) || canonicalDecimal(token) !== canonicalDecimal(String(Number(token))))) {
            return { path: "", message: "contains a JSON number that cannot be represented exactly for numeric enum validation" };
          }
        }
        if (currentValue && typeof currentValue === "object" && !Array.isArray(currentValue) && currentSchema.properties) {
          for (const [key, childSchema] of Object.entries(currentSchema.properties)) {
            if (Object.hasOwn(currentValue, key)) {
              const error = visit(currentValue[key], childSchema, [...currentPath, key]);
              if (error) return error;
            }
          }
        }
        if (Array.isArray(currentValue) && currentSchema.items) {
          for (let index = 0; index < currentValue.length; index++) {
            const error = visit(currentValue[index], currentSchema.items, [...currentPath, index]);
            if (error) return error;
          }
        }
        for (const alternative of currentSchema.oneOf || currentSchema.anyOf || []) {
          const error = visit(currentValue, alternative, currentPath);
          if (error) return error;
        }
        return undefined;
      };
      const error = visit(value, schema, []);
      if (error) return error;
    }
    const error = validateValueAgainstSchema(value, schema, { includeReceived: false });
    return error ? { path: error.path, message: error.message } : undefined;
  };
}

module.exports = { prepareJsonSchema };
