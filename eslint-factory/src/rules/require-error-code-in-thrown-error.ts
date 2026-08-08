import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

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

function getLocalSuperclassName(node: TSESTree.ClassDeclaration): string | null {
  if (!node.id || !node.superClass || node.superClass.type !== AST_NODE_TYPES.Identifier) {
    return null;
  }
  return node.superClass.name;
}

function isAstNode(value: unknown): value is TSESTree.Node {
  return typeof value === "object" && value !== null && typeof (value as { type?: unknown }).type === "string";
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
    const localClassExtends = new Map<string, string>();

    if (!importsErrorCodes) {
      return {};
    }

    function isErrorConstructor(name: string): boolean {
      const seen = new Set<string>();
      let current: string | undefined = name;
      while (current) {
        if (current === "Error") return true;
        if (seen.has(current)) return false;
        seen.add(current);
        current = localClassExtends.get(current);
      }
      return false;
    }

    function collectLocalClassExtends(node: TSESTree.Node, seen = new WeakSet<object>()) {
      if (seen.has(node)) return;
      seen.add(node);
      if (node.type === AST_NODE_TYPES.ClassDeclaration) {
        const superclassName = getLocalSuperclassName(node);
        if (superclassName && node.id) {
          localClassExtends.set(node.id.name, superclassName);
        }
      }

      for (const [key, value] of Object.entries(node as unknown as Record<string, unknown>)) {
        if (key === "parent") continue;
        if (Array.isArray(value)) {
          for (const child of value) {
            if (isAstNode(child)) collectLocalClassExtends(child, seen);
          }
        } else if (isAstNode(value)) {
          collectLocalClassExtends(value, seen);
        }
      }
    }

    return {
      Program(node: TSESTree.Program) {
        collectLocalClassExtends(node);
      },
      ThrowStatement(node: TSESTree.ThrowStatement) {
        const arg = node.argument;
        if (!arg || arg.type !== AST_NODE_TYPES.NewExpression) return;
        const callee = arg.callee;
        if (callee.type !== AST_NODE_TYPES.Identifier || !isErrorConstructor(callee.name)) return;
        const messageArg = arg.arguments[0];
        if (!messageArg) return;
        if (messageArg.type !== AST_NODE_TYPES.TemplateLiteral && messageArg.type !== AST_NODE_TYPES.Literal && messageArg.type !== AST_NODE_TYPES.Identifier && messageArg.type !== AST_NODE_TYPES.BinaryExpression) {
          return;
        }

        if (messageReferencesErrorCode(messageArg)) return;

        context.report({
          node: arg,
          messageId: "missingErrorCode",
        });
      },
    };
  },
});
