import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { resolveWriteOnceInitializerChain } from "./command-initializer-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const ERROR_CODE_PATTERN = /ERR_[A-Z_]+|E[0-9]{3}(?!\d)/;

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
      callExpressionNeedsReview:
        "This file imports error_codes.cjs but this thrown Error message comes from a helper call that cannot be statically verified to reference an error code. Add a visible ERR_* prefix at the throw site or review the helper and suppress this warning intentionally.",
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

    type Scope = ReturnType<typeof sourceCode.getScope>;

    function findVariableInScopeChain(scope: Scope, name: string) {
      let current: Scope | null = scope;
      while (current) {
        const variable = current.variables.find(v => v.name === name);
        if (variable) return variable;
        current = current.upper;
      }
      return null;
    }

    type MessageAuditResult = "hasCode" | "missingCode" | "needsReview";

    function auditMessageExpression(node: TSESTree.Node, unresolvedIdentifierResult: MessageAuditResult = "hasCode"): MessageAuditResult {
      if (messageReferencesErrorCode(node)) return "hasCode";
      if (node.type === AST_NODE_TYPES.CallExpression) {
        const returns = resolveSimpleLocalCallReturns(node);
        if (!returns) return "needsReview";
        for (const returnExpr of returns) {
          const returnResult = auditMessageExpression(returnExpr, "needsReview");
          if (returnResult !== "hasCode") return returnResult;
        }
        return "hasCode";
      }
      if (node.type === AST_NODE_TYPES.Identifier) {
        const resolved = resolveWriteOnceInitializerChain(node, sourceCode);
        if (resolved === node) return unresolvedIdentifierResult;
        if (!isAuditableMessageExpression(resolved)) return resolved.type === AST_NODE_TYPES.CallExpression ? "needsReview" : "hasCode";
        return messageReferencesErrorCode(resolved) ? "hasCode" : "missingCode";
      }
      return "missingCode";
    }

    function isAuditableMessageExpression(node: TSESTree.Node): boolean {
      return node.type === AST_NODE_TYPES.TemplateLiteral || node.type === AST_NODE_TYPES.Literal || node.type === AST_NODE_TYPES.Identifier || (node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+");
    }

    function resolveSimpleLocalCallReturns(node: TSESTree.CallExpression): TSESTree.Expression[] | null {
      if (node.callee.type !== AST_NODE_TYPES.Identifier) return null;
      const variable = findVariableInScopeChain(sourceCode.getScope(node.callee), node.callee.name);
      if (!variable || variable.defs.length !== 1) return null;
      const def = variable.defs[0];

      let body: TSESTree.BlockStatement | TSESTree.Expression | null = null;
      if (def.type === "FunctionName") {
        body = (def.node as TSESTree.FunctionDeclaration).body;
      } else if (def.type === "Variable" && def.node.init && (def.node.init.type === AST_NODE_TYPES.FunctionExpression || def.node.init.type === AST_NODE_TYPES.ArrowFunctionExpression)) {
        body = def.node.init.body;
      } else {
        return null;
      }

      if (!body) return null;
      if (body.type !== AST_NODE_TYPES.BlockStatement) return [body];
      let returnIndex = -1;
      for (let index = 0; index < body.body.length; index++) {
        if (body.body[index].type !== AST_NODE_TYPES.ReturnStatement) continue;
        if (returnIndex !== -1) return null;
        returnIndex = index;
      }
      if (returnIndex === -1) return null;
      if (body.body.slice(0, returnIndex).some(statement => statement.type !== AST_NODE_TYPES.VariableDeclaration)) return null;
      if (returnIndex + 1 < body.body.length) return null;
      const statement = body.body[returnIndex];
      return statement.type === AST_NODE_TYPES.ReturnStatement && statement.argument ? [statement.argument] : null;
    }

    function isErrorConstructorViaScope(callee: TSESTree.Identifier): boolean {
      const visited = new Set<object>();

      function check(name: string, scope: Scope): boolean {
        if (name === "Error") return true;

        const variable = findVariableInScopeChain(scope, name);
        if (!variable) return false;

        for (const def of variable.defs) {
          if (def.type !== "ClassName") continue;
          const classNode = def.node as TSESTree.ClassDeclaration;
          if (visited.has(classNode)) continue;
          visited.add(classNode);

          if (!classNode.superClass || classNode.superClass.type !== AST_NODE_TYPES.Identifier) continue;

          const superScope = sourceCode.getScope(classNode.superClass);
          if (check(classNode.superClass.name, superScope)) return true;
        }

        return false;
      }

      return check(callee.name, sourceCode.getScope(callee));
    }

    return {
      ThrowStatement(node: TSESTree.ThrowStatement) {
        const arg = node.argument;
        if (!arg || arg.type !== AST_NODE_TYPES.NewExpression) return;
        const callee = arg.callee;
        if (callee.type !== AST_NODE_TYPES.Identifier || !isErrorConstructorViaScope(callee)) return;
        const messageArg = arg.arguments[0];
        if (!messageArg) return;
        if (!isAuditableMessageExpression(messageArg) && messageArg.type !== AST_NODE_TYPES.CallExpression) {
          return;
        }

        const auditResult = auditMessageExpression(messageArg);
        if (auditResult === "hasCode") return;

        context.report({
          node: arg,
          messageId: auditResult === "needsReview" ? "callExpressionNeedsReview" : "missingErrorCode",
        });
      },
    };
  },
});
