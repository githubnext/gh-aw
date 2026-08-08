import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { resolveWriteOnceInitializerChain } from "./command-initializer-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const ERROR_CODE_PATTERN = /\bERR_[A-Z_]+\b|\bE[0-9]{3}\b/;

/**
 * Returns true when the given template literal or string literal argument
 * textually references a standardized error code (e.g. ERR_API, ERR_NOT_FOUND,
 * or a SAFE_OUTPUT_E001-style numeric code).
 */
function messageReferencesErrorCode(node: TSESTree.Node): boolean {
  const text = (node as unknown as { raw?: string }).raw ?? "";
  if (ERROR_CODE_PATTERN.test(text)) return true;
  if (node.type === AST_NODE_TYPES.TemplateLiteral) {
    for (const quasi of node.quasis) {
      if (ERROR_CODE_PATTERN.test(quasi.value.raw)) return true;
    }
    for (const expr of node.expressions) {
      if (expr.type === AST_NODE_TYPES.Identifier && ERROR_CODE_PATTERN.test(expr.name)) return true;
      if (expr.type === AST_NODE_TYPES.MemberExpression && expr.property.type === AST_NODE_TYPES.Identifier && ERROR_CODE_PATTERN.test(expr.property.name)) {
        return true;
      }
    }
  }
  if (node.type === AST_NODE_TYPES.Identifier && ERROR_CODE_PATTERN.test(node.name)) return true;
  if (node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+") {
    return messageReferencesErrorCode(node.left) || messageReferencesErrorCode(node.right);
  }
  return false;
}

export const requireErrorCodeInThrownErrorRule = createRule({
  name: "require-error-code-in-thrown-error",
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Require thrown Error messages to reference a standardized error code (ERR_* from error_codes.cjs) in files that already import error_codes.cjs — keeps error-code coverage consistent so logs/dashboards can filter reliably.",
    },
    schema: [],
    messages: {
      missingErrorCode:
        "This file imports error_codes.cjs but this thrown Error message does not reference a standardized error code (e.g. ERR_API, ERR_NOT_FOUND). Prefix the message with an imported ERR_* constant for consistency with other errors in this file.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const fullText = sourceCode.getText();
    const importsErrorCodes = /require\(\s*["']\.\/error_codes\.cjs["']\s*\)/.test(fullText);

    if (!importsErrorCodes) {
      return {};
    }

    return {
      ThrowStatement(node: TSESTree.ThrowStatement) {
        const arg = node.argument;
        if (!arg || arg.type !== AST_NODE_TYPES.NewExpression) return;
        const callee = arg.callee;
        if (callee.type !== AST_NODE_TYPES.Identifier || callee.name !== "Error") return;
        const messageArg = arg.arguments[0];
        if (!messageArg) return;
        if (messageArg.type !== AST_NODE_TYPES.TemplateLiteral && messageArg.type !== AST_NODE_TYPES.Literal && messageArg.type !== AST_NODE_TYPES.Identifier && messageArg.type !== AST_NODE_TYPES.BinaryExpression) {
          return;
        }

        if (messageReferencesErrorCode(messageArg)) return;

        if (messageArg.type === AST_NODE_TYPES.Identifier) {
          // Resolve write-once local initializers so a message built from an
          // ERR_* constant is recognized even when it is held in a plain-named
          // variable (e.g. `const errorMsg = `${ERR_SYSTEM}: ...`;`).
          const resolved = resolveWriteOnceInitializerChain(messageArg, sourceCode);
          // Unresolvable values (parameters, reassigned bindings, call results,
          // values from a guarded branch) stay silent: false positives are worse
          // than silence for this consistency rule.
          if (resolved.type !== AST_NODE_TYPES.TemplateLiteral && resolved.type !== AST_NODE_TYPES.Literal && resolved.type !== AST_NODE_TYPES.BinaryExpression) return;
          if (messageReferencesErrorCode(resolved)) return;
        }

        context.report({
          node: arg,
          messageId: "missingErrorCode",
        });
      },
    };
  },
});
