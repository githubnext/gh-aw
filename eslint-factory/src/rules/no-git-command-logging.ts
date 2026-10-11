import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
const childMethods = new Set(["exec", "execSync", "execFile", "execFileSync", "spawn", "spawnSync"]);

function variableFor(identifier: TSESTree.Identifier, source: TSESLint.SourceCode) {
  let scope: TSESLint.Scope.Scope | null = source.getScope(identifier);
  while (scope) {
    const variable = scope.set.get(identifier.name);
    if (variable) return variable;
    scope = scope.upper;
  }
  return undefined;
}

function hasWrites(variable: TSESLint.Scope.Variable): boolean {
  return variable.references.some(ref => {
    if (ref.isWrite() && !ref.init) return true;
    const member = ref.identifier.parent;
    if (member?.type !== AST_NODE_TYPES.MemberExpression || member.object !== ref.identifier) return false;
    const parent = member.parent;
    return (parent?.type === AST_NODE_TYPES.AssignmentExpression && parent.left === member) || parent?.type === AST_NODE_TYPES.UpdateExpression;
  });
}

function initializer(identifier: TSESTree.Identifier, source: TSESLint.SourceCode): TSESTree.Expression | undefined {
  const variable = variableFor(identifier, source);
  if (!variable || variable.defs.length !== 1 || hasWrites(variable)) return undefined;
  const def = variable.defs[0];
  return def.type === "Variable" && def.node.type === AST_NODE_TYPES.VariableDeclarator && def.node.id.type === AST_NODE_TYPES.Identifier ? (def.node.init ?? undefined) : undefined;
}

function resolve(expression: TSESTree.Node | undefined, source: TSESLint.SourceCode, seen = new Set<TSESTree.Node>()): TSESTree.Node | undefined {
  if (!expression || seen.has(expression)) return undefined;
  seen.add(expression);
  if (expression.type === AST_NODE_TYPES.Identifier) {
    const init = initializer(expression, source);
    if (init) return resolve(init, source, seen);
  }
  return expression;
}

function staticString(expression: TSESTree.Node | undefined, source: TSESLint.SourceCode): string | undefined {
  const value = resolve(expression, source);
  if (value?.type === AST_NODE_TYPES.Literal && typeof value.value === "string") return value.value;
  if (value?.type === AST_NODE_TYPES.TemplateLiteral && value.expressions.length === 0) return value.quasis[0].value.cooked ?? undefined;
  return undefined;
}

function propertyValue(expression: TSESTree.Node | undefined, name: string, source: TSESLint.SourceCode, seen = new Set<TSESTree.Node>()): TSESTree.Node | undefined {
  const object = resolve(expression, source);
  if (!object || seen.has(object)) return undefined;
  seen.add(object);
  if (object.type !== AST_NODE_TYPES.ObjectExpression) return undefined;
  let result: TSESTree.Node | undefined;
  for (const prop of object.properties) {
    if (prop.type === AST_NODE_TYPES.SpreadElement) {
      // An unknown spread can overwrite a previously safe option.
      const spread = resolve(prop.argument, source);
      if (spread?.type !== AST_NODE_TYPES.ObjectExpression) result = undefined;
      else {
        const value = propertyValue(spread, name, source, new Set(seen));
        const declaresProperty = spread.properties.some(p => p.type === AST_NODE_TYPES.SpreadElement || (p.type === AST_NODE_TYPES.Property && staticKey(p) === name));
        if (declaresProperty) result = value;
      }
    } else if (prop.type === AST_NODE_TYPES.Property) {
      if (prop.computed && staticKey(prop) === undefined) result = undefined;
      else if (staticKey(prop) === name) result = resolve(prop.value, source);
    }
  }
  return result;
}

function staticKey(prop: TSESTree.Property): string | undefined {
  if (!prop.computed && prop.key.type === AST_NODE_TYPES.Identifier) return prop.key.name;
  return prop.key.type === AST_NODE_TYPES.Literal && typeof prop.key.value === "string" ? prop.key.value : undefined;
}

function isGitLoggingOptions(expression: TSESTree.Node | undefined, source: TSESLint.SourceCode): boolean {
  if (expression?.type !== AST_NODE_TYPES.CallExpression || expression.callee.type !== AST_NODE_TYPES.Identifier) return false;
  const name = expression.callee.name;
  const variable = variableFor(expression.callee, source);
  if (!variable || hasWrites(variable)) return false;
  return (
    variable.defs.some(def => {
      if (def.type !== "Variable" || def.node.type !== AST_NODE_TYPES.VariableDeclarator || def.node.id.type !== AST_NODE_TYPES.ObjectPattern) return false;
      const init = def.node.init;
      return (
        init?.type === AST_NODE_TYPES.CallExpression &&
        init.callee.type === AST_NODE_TYPES.Identifier &&
        init.callee.name === "require" &&
        staticString(init.arguments[0], source) === "./git_logging.cjs" &&
        def.node.id.properties.some(prop => prop.type === AST_NODE_TYPES.Property && staticKey(prop) === "gitLoggingOptions" && prop.value.type === AST_NODE_TYPES.Identifier && prop.value.name === name)
      );
    }) ?? false
  );
}

function methodFor(node: TSESTree.CallExpression, source: TSESLint.SourceCode): { name: string; child: boolean } | undefined {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.MemberExpression) {
    const name = !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier ? callee.property.name : staticString(callee.property, source);
    if (!name) return undefined;
    if (callee.object.type === AST_NODE_TYPES.Identifier && isChildProcessObjectBinding(callee.object.name, callee.object, source) && childMethods.has(name)) return { name, child: true };
    if (name === "getExecOutput" || (name === "exec" && node.arguments.length >= 2)) return { name, child: false };
  }
  if (callee.type === AST_NODE_TYPES.Identifier) {
    const variable = variableFor(callee, source);
    for (const def of variable?.defs ?? []) {
      if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
        const imported = def.node.imported;
        const name = imported.type === AST_NODE_TYPES.Identifier ? imported.name : imported.value;
        if (childMethods.has(name)) return { name, child: true };
      }
      if (def.type === "Variable" && def.node.type === AST_NODE_TYPES.VariableDeclarator && def.node.id.type === AST_NODE_TYPES.ObjectPattern) {
        const child = isRequireChildProcess(def.node.init);
        for (const prop of def.node.id.properties) {
          if (prop.type !== AST_NODE_TYPES.Property || prop.value.type !== AST_NODE_TYPES.Identifier || prop.value.name !== callee.name) continue;
          const name = staticKey(prop);
          if (name && ((child && childMethods.has(name)) || (!child && (name === "exec" || name === "getExecOutput")))) return { name, child };
        }
      }
    }
    const init = initializer(callee, source);
    if (init?.type === AST_NODE_TYPES.MemberExpression) {
      const name = !init.computed && init.property.type === AST_NODE_TYPES.Identifier ? init.property.name : staticString(init.property, source);
      if (name && childMethods.has(name) && init.object.type === AST_NODE_TYPES.Identifier && isChildProcessObjectBinding(init.object.name, init.object, source)) return { name, child: true };
      if (name === "exec" || name === "getExecOutput") return { name, child: false };
    }
    if (callee.name === "execGetExecOutput") return { name: "getExecOutput", child: false };
  }
  return undefined;
}

function isGitOutput(expression: TSESTree.Node | undefined, source: TSESLint.SourceCode, seen = new Set<TSESTree.Node>()): boolean {
  if (!expression || seen.has(expression)) return false;
  seen.add(expression);
  if (expression.type === AST_NODE_TYPES.AwaitExpression) return isGitOutput(expression.argument, source, seen);
  if (expression.type === AST_NODE_TYPES.NewExpression && expression.callee.type === AST_NODE_TYPES.Identifier && expression.callee.name === "Error") return isGitOutput(expression.arguments[0], source, seen);
  if (expression.type === AST_NODE_TYPES.Identifier) {
    const variable = variableFor(expression, source);
    if (!variable || variable.defs.length !== 1 || hasWrites(variable)) return false;
    const def = variable.defs[0];
    if (def.type !== "Variable" || def.node.type !== AST_NODE_TYPES.VariableDeclarator) return false;
    if (def.node.id.type === AST_NODE_TYPES.ObjectPattern) {
      const prop = def.node.id.properties.find(p => p.type === AST_NODE_TYPES.Property && p.value.type === AST_NODE_TYPES.Identifier && p.value.name === expression.name);
      if (!prop || prop.type !== AST_NODE_TYPES.Property || !["stdout", "stderr"].includes(staticKey(prop) ?? "")) return false;
    }
    return isGitOutput(def.node.init ?? undefined, source, seen);
  }
  if (expression.type === AST_NODE_TYPES.MemberExpression) {
    const name = !expression.computed && expression.property.type === AST_NODE_TYPES.Identifier ? expression.property.name : staticString(expression.property, source);
    if (["length", "exitCode", "status", "signal", "code"].includes(name ?? "")) return false;
    return isGitOutput(expression.object, source, seen);
  }
  if (expression.type === AST_NODE_TYPES.CallExpression) {
    if (expression.callee.type === AST_NODE_TYPES.Identifier && expression.callee.name === "execGitSync") return true;
    const method = methodFor(expression, source);
    if (method && method.name !== "exec" && /^(?:.*[/\\])?git(?:\.exe)?(?:\s|$)/.test(staticString(expression.arguments[0], source) ?? "")) return true;
    if (expression.callee.type === AST_NODE_TYPES.MemberExpression) return isGitOutput(expression.callee.object, source, seen);
    return false;
  }
  if (expression.type === AST_NODE_TYPES.TemplateLiteral) return expression.expressions.some(part => isGitOutput(part, source, new Set(seen)));
  if (expression.type === AST_NODE_TYPES.BinaryExpression && expression.operator === "+") return isGitOutput(expression.left, source, new Set(seen)) || isGitOutput(expression.right, source, new Set(seen));
  if (expression.type === AST_NODE_TYPES.LogicalExpression) return isGitOutput(expression.left, source, new Set(seen)) || isGitOutput(expression.right, source, new Set(seen));
  if (expression.type === AST_NODE_TYPES.ConditionalExpression) return isGitOutput(expression.consequent, source, new Set(seen)) || isGitOutput(expression.alternate, source, new Set(seen));
  return false;
}

function isLogCall(node: TSESTree.CallExpression, source: TSESLint.SourceCode): boolean {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) return callee.name === "debugLog";
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed || callee.property.type !== AST_NODE_TYPES.Identifier) return false;
  const receiver = resolve(callee.object, source);
  if (receiver?.type === AST_NODE_TYPES.Identifier && receiver.name === "core") return false;
  if (receiver?.type === AST_NODE_TYPES.CallExpression && receiver.callee.type === AST_NODE_TYPES.Identifier && receiver.callee.name === "require" && staticString(receiver.arguments[0], source) === "@actions/core") return false;
  if (receiver?.type === AST_NODE_TYPES.MemberExpression && !receiver.computed && receiver.property.type === AST_NODE_TYPES.Identifier && receiver.property.name === "core") return false;
  if (["writeFileSync", "writeFile", "appendFileSync", "appendFile"].includes(callee.property.name)) {
    const path = node.arguments[0];
    return /\.log(?:$|\.)/.test(staticString(path, source) ?? "") || (path?.type === AST_NODE_TYPES.Identifier && /log(File|Path)/i.test(path.name));
  }
  return ["info", "debug", "debugError", "warning", "warn", "error", "log", "setFailed"].includes(callee.property.name);
}

export const noGitCommandLoggingRule = createRule({
  name: "no-git-command-logging",
  meta: {
    type: "problem",
    docs: { description: "Require silent @actions/exec git calls and piped child_process git output to keep credentials and repository contents out of logs." },
    schema: [],
    fixable: "code",
    messages: {
      silent: "Git commands may expose credentials or file contents. Set silent: true after all option spreads; consume output without logging it.",
      stdio: "Do not inherit or redirect git stdout/stderr to files. Use piped output and consume it without logging it.",
      output: "Do not send captured Git output to raw/file-backed loggers. Use runner-redacted core.info/core.debug or log fixed diagnostics, exit codes, or byte counts.",
    },
  },
  defaultOptions: [],
  create(context) {
    const source = context.sourceCode;
    return {
      CallExpression(node) {
        if (isLogCall(node, source) && node.arguments.some(arg => isGitOutput(arg, source))) context.report({ node, messageId: "output" });
        const method = methodFor(node, source);
        const command = staticString(node.arguments[0], source);
        if (!method || !command || !/^(?:.*[/\\])?git(?:\.exe)?(?:\s|$)/.test(command)) return;
        const secondIsOptions = method.child && resolve(node.arguments[1], source)?.type === AST_NODE_TYPES.ObjectExpression;
        const index = method.child && (method.name === "exec" || method.name === "execSync" || secondIsOptions) ? 1 : 2;
        let options: TSESTree.CallExpressionArgument | undefined = node.arguments[index];
        if (method.child && (method.name === "exec" || method.name === "execFile")) {
          const candidate = resolve(options, source);
          const callbackVariable = candidate?.type === AST_NODE_TYPES.Identifier ? variableFor(candidate, source) : undefined;
          if (
            candidate?.type === AST_NODE_TYPES.ArrowFunctionExpression ||
            candidate?.type === AST_NODE_TYPES.FunctionExpression ||
            (callbackVariable && !hasWrites(callbackVariable) && callbackVariable.defs.some(def => def.type === "FunctionName"))
          )
            options = undefined;
        }
        if (method.child) {
          const stdio = propertyValue(options, "stdio", source);
          // child_process defaults to pipes. Explicit descriptors must also keep
          // stdout/stderr off inherited streams and arbitrary file descriptors.
          const object = resolve(options, source);
          if (!stdio && (!options || (object?.type === AST_NODE_TYPES.ObjectExpression && object.properties.every(prop => prop.type === AST_NODE_TYPES.Property && !prop.computed)))) return;
          if (["pipe", "ignore"].includes(staticString(stdio, source) ?? "")) return;
          if (
            stdio?.type === AST_NODE_TYPES.ArrayExpression &&
            !stdio.elements.slice(0, 3).some(element => element?.type === AST_NODE_TYPES.SpreadElement) &&
            [1, 2].every(i => {
              const value = stdio.elements[i];
              return !value || (value.type === AST_NODE_TYPES.Literal && value.value == null) || ["pipe", "ignore"].includes(staticString(value, source) ?? "");
            })
          )
            return;
          context.report({ node, messageId: "stdio" });
          return;
        }
        if (isGitLoggingOptions(options, source)) return;
        const silent = propertyValue(options, "silent", source);
        if (silent?.type === AST_NODE_TYPES.Literal && silent.value === true) return;
        context.report({
          node,
          messageId: "silent",
          fix(fixer) {
            if (options?.type === AST_NODE_TYPES.SpreadElement || node.arguments.length > 3) return null;
            if (!options) {
              const last = node.arguments[node.arguments.length - 1];
              if (!last || node.arguments.length < 2) return null;
              return fixer.insertTextAfter(last, ", { silent: true }");
            }
            if (options.type === AST_NODE_TYPES.ObjectExpression) {
              const silentProperties = options.properties.filter((prop): prop is TSESTree.Property => prop.type === AST_NODE_TYPES.Property && staticKey(prop) === "silent");
              if (silentProperties.length === 1 && options.properties[options.properties.length - 1] === silentProperties[0]) return fixer.replaceText(silentProperties[0], "silent: true");
              const text = source.getText(options).slice(0, -1).trimEnd();
              return fixer.replaceText(options, `${text}${options.properties.length && !text.endsWith(",") ? "," : ""} silent: true }`);
            }
            return fixer.replaceText(options, `{ ...(${source.getText(options)}), silent: true }`);
          },
        });
      },
    };
  },
});
