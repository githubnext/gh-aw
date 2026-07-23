import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;
type ChildProcessMethod = "exec" | "execSync" | "spawn" | "spawnSync" | "execFile" | "execFileSync";
const SHELL_CONDITIONAL_METHODS = new Set<ChildProcessMethod>(["spawn", "spawnSync", "execFile", "execFileSync"]);

function isStaticExpression(node: TSESTree.Expression): boolean {
  if (node.type === AST_NODE_TYPES.Literal) return true;
  if (node.type === AST_NODE_TYPES.TemplateLiteral) return node.expressions.length === 0;
  if (node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+") {
    return isStaticExpression(node.left) && isStaticExpression(node.right);
  }
  return false;
}

function isDynamicStringConcatenation(node: TSESTree.Expression): boolean {
  return node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+" && !isStaticExpression(node);
}

function getDynamicCommandKind(node: TSESTree.Expression): string | null {
  if (node.type === AST_NODE_TYPES.TemplateLiteral && node.expressions.length > 0) return "interpolated template literal";
  if (isDynamicStringConcatenation(node)) return "dynamic string concatenation";
  return null;
}

function getImportSpecifierName(node: TSESTree.ImportSpecifier): string | null {
  if (node.imported.type === AST_NODE_TYPES.Identifier) return node.imported.name;
  if (node.imported.type === AST_NODE_TYPES.Literal && typeof node.imported.value === "string") return node.imported.value;
  return null;
}

function getShellPropertyValue(optionsArg: TSESTree.ObjectExpression): boolean {
  for (const prop of optionsArg.properties) {
    if (prop.type !== AST_NODE_TYPES.Property || prop.computed) continue;

    const keyName = prop.key.type === AST_NODE_TYPES.Identifier ? prop.key.name : prop.key.type === AST_NODE_TYPES.Literal ? prop.key.value : null;
    if (keyName !== "shell") continue;

    return prop.value.type === AST_NODE_TYPES.Literal && (prop.value.value === true || typeof prop.value.value === "string");
  }

  return false;
}

function resolveObjectExpression(arg: TSESTree.CallExpressionArgument, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): TSESTree.ObjectExpression | null {
  if (arg.type === AST_NODE_TYPES.ObjectExpression) return arg;
  if (arg.type !== AST_NODE_TYPES.Identifier) return null;

  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(arg.name);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        if (def.type !== "Variable") continue;
        const declarator = def.node as TSESTree.VariableDeclarator;
        if (declarator.id.type !== AST_NODE_TYPES.Identifier || declarator.id.name !== arg.name) continue;
        if (declarator.init?.type === AST_NODE_TYPES.ObjectExpression) return declarator.init;
      }
      return null;
    }
    scope = scope.upper;
  }

  return null;
}

function isShellTrueOption(optionsArg: TSESTree.CallExpressionArgument | undefined, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  if (!optionsArg) return false;
  if (optionsArg.type === AST_NODE_TYPES.SpreadElement) return true; // Conservative: spread arguments are treated as possibly shell-enabled.

  const resolvedObject = resolveObjectExpression(optionsArg, scopeNode, sourceCode);
  if (!resolvedObject) return false;
  return getShellPropertyValue(resolvedObject);
}

function requiresShellTrue(method: ChildProcessMethod): boolean {
  return SHELL_CONDITIONAL_METHODS.has(method);
}

function hasShellTrueOptions(node: TSESTree.CallExpression, method: ChildProcessMethod, sourceCode: TSESLint.SourceCode): boolean {
  if (!requiresShellTrue(method)) return true;
  return isShellTrueOption(node.arguments[1], node, sourceCode) || isShellTrueOption(node.arguments[2], node, sourceCode);
}

function isChildProcessMethodBinding(method: ChildProcessMethod, identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const importedName = getImportSpecifierName(def.node);
          if (importedName === method) return true;
        }

        if (def.type !== "Variable") continue;

        const declarator = def.node as TSESTree.VariableDeclarator;
        if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
          for (const prop of declarator.id.properties) {
            if (prop.type !== AST_NODE_TYPES.Property || prop.key.type !== AST_NODE_TYPES.Identifier || prop.key.name !== method) continue;
            const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
            if (boundName === identifierName) return true;
          }
        }

        if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
          const init = declarator.init;
          if (!init.computed && init.object.type === AST_NODE_TYPES.Identifier && isChildProcessObjectBinding(init.object.name, init.object, sourceCode) && init.property.type === AST_NODE_TYPES.Identifier && init.property.name === method) {
            return true;
          }
        }
      }
      return false;
    }
    scope = scope.upper;
  }
  return false;
}

function resolveChildProcessMethod(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): ChildProcessMethod | null {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) {
    const methods: ChildProcessMethod[] = ["exec", "execSync", "spawn", "spawnSync", "execFile", "execFileSync"];
    for (const method of methods) {
      if (isChildProcessMethodBinding(method, callee.name, callee, sourceCode)) return method;
    }
    return null;
  }

  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return null;
  if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.property.type !== AST_NODE_TYPES.Identifier) return null;
  if (!isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode)) return null;

  const method = callee.property.name;
  return method === "exec" || method === "execSync" || method === "spawn" || method === "spawnSync" || method === "execFile" || method === "execFileSync" ? method : null;
}

export const noChildProcessInterpolatedCommandRule = createRule({
  name: "no-child-process-interpolated-command",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow interpolated template literals or dynamic string concatenation as child_process command arguments for shell-evaluated execution paths. " +
        "Dynamic command strings can become shell-injection vectors. Prefer static command names with argument arrays and avoid shell: true.",
    },
    schema: [],
    messages: {
      interpolatedCommand:
        "Avoid passing a {{kind}} as the command to child_process.{{method}} — shell-evaluated command strings can enable shell injection. " +
        "Prefer a static executable plus an argument array (for example, execFileSync(cmd, [arg1, arg2])) and avoid shell: true when possible.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const method = resolveChildProcessMethod(node, sourceCode);
        if (!method) return;
        if (!hasShellTrueOptions(node, method, sourceCode)) return;

        const firstArg = node.arguments[0];
        if (!firstArg || firstArg.type === AST_NODE_TYPES.SpreadElement) return;

        const kind = getDynamicCommandKind(firstArg);
        if (!kind) return;

        context.report({
          node: firstArg,
          messageId: "interpolatedCommand",
          data: { kind, method },
        });
      },
    };
  },
});
