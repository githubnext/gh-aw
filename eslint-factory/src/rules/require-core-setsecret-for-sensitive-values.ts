import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier, isDestructuredCoreMethodIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const SENSITIVE_SEGMENTS = new Set(["secret", "password", "passwd", "credential", "credentials", "token", "apikey", "privatekey", "accesskey", "clientsecret"]);
const NON_SECRET_TOKEN_SUFFIXES = new Set(["budget", "count", "counts", "estimate", "limit", "metric", "rate", "threshold", "usage", "warning", "warnings"]);
const METADATA_SUFFIXES = new Set(["claims", "context", "detected", "events", "label", "list", "match", "message", "name", "names", "requirement", "result", "secrets", "summary", "values"]);

interface Candidate {
  name: string;
  node: TSESTree.Node;
}

function nameSegments(name: string): string[] {
  return name
    .replace(/([a-z0-9])([A-Z])/g, "$1_$2")
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(Boolean);
}

function isSensitiveName(name: string): boolean {
  const segments = nameSegments(name);
  const compact = segments.join("");
  if (["has", "is", "using"].includes(segments[0] ?? "")) return false;
  if (["docs", "names", "pattern", "regex", "template"].includes(segments.at(-1) ?? "")) return false;
  if (segments.includes("config") && segments.at(-1) === "key") return false;
  if (SENSITIVE_SEGMENTS.has(compact)) return true;

  const tokenIndex = segments.indexOf("token");
  if (tokenIndex >= 0 && NON_SECRET_TOKEN_SUFFIXES.has(segments[tokenIndex + 1] ?? "")) return false;
  if (segments.includes("tokens")) return false;

  return segments.some(segment => SENSITIVE_SEGMENTS.has(segment));
}

function isMetadataName(name: string): boolean {
  const segments = nameSegments(name);
  return ["has", "is", "using"].includes(segments[0] ?? "") || (segments.length > 1 && METADATA_SUFFIXES.has(segments.at(-1) ?? ""));
}

function staticPropertyName(node: TSESTree.MemberExpression): string | null {
  if (!node.computed && node.property.type === AST_NODE_TYPES.Identifier) return node.property.name;
  if (node.computed && node.property.type === AST_NODE_TYPES.Literal && typeof node.property.value === "string") return node.property.value;
  return null;
}

function containsSensitiveValue(node: TSESTree.Node): boolean {
  switch (node.type) {
    case AST_NODE_TYPES.Identifier:
      return isSensitiveName(node.name);
    case AST_NODE_TYPES.MemberExpression: {
      const propertyName = staticPropertyName(node);
      return (propertyName !== null && isSensitiveName(propertyName)) || containsSensitiveValue(node.object);
    }
    case AST_NODE_TYPES.ChainExpression:
      return containsSensitiveValue(node.expression);
    case AST_NODE_TYPES.AwaitExpression:
      return containsSensitiveValue(node.argument);
    case AST_NODE_TYPES.LogicalExpression:
      return containsSensitiveValue(node.left) || containsSensitiveValue(node.right);
    case AST_NODE_TYPES.ConditionalExpression:
      return containsSensitiveValue(node.consequent) || containsSensitiveValue(node.alternate);
    case AST_NODE_TYPES.CallExpression: {
      if (node.callee.type === AST_NODE_TYPES.MemberExpression && containsSensitiveValue(node.callee.object)) return true;
      const calleeText =
        node.callee.type === AST_NODE_TYPES.Identifier
          ? node.callee.name
          : node.callee.type === AST_NODE_TYPES.MemberExpression
            ? `${node.callee.object.type === AST_NODE_TYPES.Identifier ? node.callee.object.name : ""}.${staticPropertyName(node.callee) ?? ""}`
            : "";
      if (!/(?:^|\.)(?:from|parse|stringify|encode|decode|escape|atob|btoa)/i.test(calleeText)) return false;
      return node.arguments.some(argument => argument.type !== AST_NODE_TYPES.SpreadElement && containsSensitiveValue(argument));
    }
    case AST_NODE_TYPES.NewExpression:
      return false;
    case AST_NODE_TYPES.TemplateLiteral:
      return node.expressions.some(containsSensitiveValue);
    default:
      return false;
  }
}

function isBooleanProbe(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.UnaryExpression && node.operator === "!") return true;
  if (node.type === AST_NODE_TYPES.BinaryExpression) return true;
  return node.type === AST_NODE_TYPES.CallExpression && node.callee.type === AST_NODE_TYPES.Identifier && node.callee.name === "Boolean";
}

function findVariable(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): TSESLint.Scope.Variable | null {
  let scope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (scope !== null) {
    const variable = scope.set.get(identifier.name);
    if (variable !== undefined) return variable;
    scope = scope.upper;
  }
  return null;
}

function isCoreSetSecretCall(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): boolean {
  const { callee } = node;
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return isDestructuredCoreMethodIdentifier(callee, "setSecret", sourceCode);
  }
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.object.type !== AST_NODE_TYPES.Identifier) return false;
  if (!CORE_ALIASES.has(callee.object.name) && !isCoreAliasIdentifier(callee.object, sourceCode)) return false;
  return staticPropertyName(callee) === "setSecret";
}

function variableIsMasked(variable: TSESLint.Scope.Variable, sourceCode: TSESLint.SourceCode): boolean {
  return variable.references.some(reference => {
    let current: TSESTree.Node | undefined = reference.identifier;
    while (current?.parent) {
      const parent: TSESTree.Node = current.parent;
      if (parent.type === AST_NODE_TYPES.CallExpression && isCoreSetSecretCall(parent, sourceCode)) {
        return parent.arguments[0] === current;
      }
      current = parent;
      if (current.type === AST_NODE_TYPES.ExpressionStatement || current.type === AST_NODE_TYPES.VariableDeclarator || current.type === AST_NODE_TYPES.AssignmentExpression || current.type === AST_NODE_TYPES.ReturnStatement) {
        return false;
      }
    }
    return false;
  });
}

export const requireCoreSetSecretForSensitiveValuesRule = createRule({
  name: "require-core-setsecret-for-sensitive-values",
  meta: {
    type: "problem",
    docs: {
      description: "Require values that heuristically look like parsed credentials or secrets to be registered with the core setSecret API for GitHub Actions log masking.",
    },
    schema: [],
    messages: {
      requireSetSecret: "Sensitive value '{{name}}' is not passed to the core setSecret API. Register it for GitHub Actions log masking before it can reach logs.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const candidates = new Map<TSESLint.Scope.Variable, Candidate>();

    function track(identifier: TSESTree.Identifier, value: TSESTree.Node, reportNode: TSESTree.Node, force = false): void {
      if (isBooleanProbe(value) || isMetadataName(identifier.name)) return;
      if (
        value.type === AST_NODE_TYPES.Literal ||
        value.type === AST_NODE_TYPES.ArrayExpression ||
        value.type === AST_NODE_TYPES.ObjectExpression ||
        value.type === AST_NODE_TYPES.FunctionExpression ||
        value.type === AST_NODE_TYPES.ArrowFunctionExpression ||
        value.type === AST_NODE_TYPES.ClassExpression ||
        (value.type === AST_NODE_TYPES.TemplateLiteral && value.expressions.length === 0)
      )
        return;
      if (!force && !isSensitiveName(identifier.name) && !containsSensitiveValue(value)) return;
      const variable = findVariable(identifier, sourceCode);
      if (variable !== null && !candidates.has(variable)) {
        candidates.set(variable, { name: identifier.name, node: reportNode });
      }
    }

    return {
      VariableDeclarator(node) {
        if (!node.init) return;
        if (node.id.type === AST_NODE_TYPES.Identifier) {
          track(node.id, node.init, node);
          return;
        }
        if (node.id.type !== AST_NODE_TYPES.ObjectPattern) return;
        if (node.init.type === AST_NODE_TYPES.CallExpression && node.init.callee.type === AST_NODE_TYPES.Identifier && node.init.callee.name === "require") return;

        for (const property of node.id.properties) {
          if (property.type !== AST_NODE_TYPES.Property || property.value.type !== AST_NODE_TYPES.Identifier) continue;
          const key = property.key.type === AST_NODE_TYPES.Identifier ? property.key.name : property.key.type === AST_NODE_TYPES.Literal ? String(property.key.value) : "";
          if (key === "setSecret") continue;
          if (isSensitiveName(key) || isSensitiveName(property.value.name)) {
            track(property.value, property.key, property, true);
          }
        }
      },
      AssignmentExpression(node) {
        if (node.left.type === AST_NODE_TYPES.Identifier) {
          track(node.left, node.right, node);
        }
      },
      "Program:exit"() {
        for (const [variable, candidate] of candidates) {
          if (!variableIsMasked(variable, sourceCode)) {
            context.report({
              node: candidate.node,
              messageId: "requireSetSecret",
              data: { name: candidate.name },
            });
          }
        }
      },
    };
  },
});
