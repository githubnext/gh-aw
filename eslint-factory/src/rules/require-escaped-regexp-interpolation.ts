import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Matches function/method names that are specifically regex-escaping helpers,
// e.g. escapeRegExp, escapeRegex, regExpEscape. Requires both "escape" and "reg"
// to be present, preventing false negatives from escapeHtml, unescape, etc.
const ESCAPE_CALL_NAME_PATTERN = /escape.*reg|reg.*escape/i;
const REGEXP_META_CHARS = new Set(["\\", "^", "$", ".", "*", "+", "?", "(", ")", "[", "]", "{", "}", "|"]);

// Matches identifier/property names that signal a value has already been
// regex-escaped, e.g. escapedValue, ESCAPED_NAME. Requires the name to START
// with "escaped", so unescapedValue and escapeHelper are never whitelisted.
const ESCAPED_IDENT_PATTERN = /^escaped/i;

/**
 * Returns true when `node` is a call expression whose callee name looks like
 * a regex-escaping helper (e.g. `escapeRegExp(value)`, `utils.escapeRegex(value)`).
 * Both "escape" and "reg" must appear in the name so unrelated helpers such as
 * `escapeHtml` or `unescape` are not treated as regex-safe.
 */
function isEscapeHelperCall(node: TSESTree.Node): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) return ESCAPE_CALL_NAME_PATTERN.test(callee.name);
  if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier) {
    return ESCAPE_CALL_NAME_PATTERN.test(callee.property.name);
  }
  return false;
}

/**
 * Returns true when `node` is a call of the form
 * `value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")` — the standard inline
 * pattern for escaping all regex metacharacters before interpolation.
 */
function isRegexEscapeReplaceCall(node: TSESTree.Node): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const { callee, arguments: args } = node;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || callee.property.name !== "replace") return false;
  if (args.length < 2) return false;
  const search = getFixedLiteralSearchText(args[0]);
  const replacement = getStringLiteralValue(args[1]);
  if (replacement === "\\$&") return true;
  return search !== null && replacement !== null && isLiteralRegexEscapeReplacement(search, replacement);
}

function getStringLiteralValue(node: TSESTree.Node): string | null {
  return node.type === AST_NODE_TYPES.Literal && typeof node.value === "string" ? node.value : null;
}

function getFixedLiteralSearchText(node: TSESTree.Node): string | null {
  const stringValue = getStringLiteralValue(node);
  if (stringValue !== null) return stringValue;
  if (node.type !== AST_NODE_TYPES.Literal || !("regex" in node) || !node.regex) return null;
  return decodeFixedLiteralRegexPattern(node.regex.pattern);
}

function decodeFixedLiteralRegexPattern(pattern: string): string | null {
  let decoded = "";

  for (let index = 0; index < pattern.length; index++) {
    const char = pattern[index];
    if (char === "\\") {
      index++;
      const escapedChar = pattern[index];
      if (escapedChar === undefined || !REGEXP_META_CHARS.has(escapedChar)) return null;
      decoded += escapedChar;
      continue;
    }

    if (REGEXP_META_CHARS.has(char)) return null;
    decoded += char;
  }

  return decoded;
}

function isLiteralRegexEscapeReplacement(search: string, replacement: string): boolean {
  if (search.length === 0 || replacement !== `\\${search}`) return false;
  if (!REGEXP_META_CHARS.has(search[0])) return false;

  for (const char of search.slice(1)) {
    if (REGEXP_META_CHARS.has(char)) return false;
  }

  return true;
}

/**
 * Returns true when `node` is an identifier or member expression whose name
 * indicates the value has already been escaped, e.g. `ESCAPED_FOO` or
 * `escapedValue`. The name must start with "escaped" so that `unescapedValue`
 * is never treated as safe.
 */
function isEscapedNameReference(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.Identifier) return ESCAPED_IDENT_PATTERN.test(node.name);
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.property.type === AST_NODE_TYPES.Identifier) {
    return ESCAPED_IDENT_PATTERN.test(node.property.name);
  }
  return false;
}

/**
 * Returns true when the interpolated expression is recognized as already
 * escaped — via a named regex-escape helper call, the standard inline
 * `.replace()` form, or a variable name that starts with "escaped".
 */
function isRecognizedAsEscaped(node: TSESTree.Node): boolean {
  return isEscapeHelperCall(node) || isRegexEscapeReplaceCall(node) || isEscapedNameReference(node);
}

export const requireEscapedRegexpInterpolationRule = createRule({
  name: "require-escaped-regexp-interpolation",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require values interpolated into a `new RegExp()` template-literal pattern to be passed through a regex-escaping helper first. " +
        "Unescaped interpolation of a value containing regex metacharacters (e.g. `.`, `*`, `+`, `(`, `)`) can produce unintended matches " +
        "or, with attacker-controlled input, a ReDoS-prone pattern.",
    },
    schema: [],
    messages: {
      unescapedInterpolation:
        "Interpolated value `{{expr}}` in `new RegExp()` template literal is not passed through a regex-escaping helper. " + 'Escape regex metacharacters before interpolating, e.g. `{{expr}}.replace(/[.*+?^${}()|[\\]\\\\]/g, "\\\\$&")`.',
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      NewExpression(node) {
        if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "RegExp") return;

        const patternArg = node.arguments[0];
        if (!patternArg || patternArg.type !== AST_NODE_TYPES.TemplateLiteral) return;
        if (patternArg.expressions.length === 0) return;

        for (const expr of patternArg.expressions) {
          if (isRecognizedAsEscaped(expr)) continue;

          context.report({
            node: expr,
            messageId: "unescapedInterpolation",
            data: { expr: sourceCode.getText(expr) },
          });
        }
      },
    };
  },
});
