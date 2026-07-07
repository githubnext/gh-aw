import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

interface CatchFrame {
  varName: string;
  aliases: Set<string>;
}

/**
 * Returns the catch variable name referenced as the first argument of a
 * `getErrorMessage(catchVar)` call, or null if the call does not match.
 */
function getErrorMessageArg(node: TSESTree.CallExpression): string | null {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.Identifier || callee.name !== "getErrorMessage") return null;
  if (node.arguments.length < 1) return null;
  const firstArg = node.arguments[0];
  if (firstArg.type !== AST_NODE_TYPES.Identifier) return null;
  return firstArg.name;
}

/**
 * Returns the first identifier name from `trackedNames` referenced in `node`,
 * or null if none are found. Used to detect whether the Error message
 * references the caught variable (or a direct alias of it).
 */
function findTrackedIdentifierReference(node: TSESTree.Expression, trackedNames: ReadonlySet<string>): string | null {
  if (node.type === AST_NODE_TYPES.Identifier && trackedNames.has(node.name)) return node.name;
  if (node.type === AST_NODE_TYPES.CallExpression) {
    const gem = getErrorMessageArg(node);
    if (gem && trackedNames.has(gem)) return gem;
    // Recurse into all arguments
    for (const arg of node.arguments) {
      if (arg.type === "SpreadElement") continue;
      const match = findTrackedIdentifierReference(arg, trackedNames);
      if (match) return match;
    }
  }
  if (node.type === AST_NODE_TYPES.TemplateLiteral) {
    for (const expr of node.expressions) {
      const match = findTrackedIdentifierReference(expr, trackedNames);
      if (match) return match;
    }
  }
  if (node.type === AST_NODE_TYPES.BinaryExpression) {
    const left = node.left;
    const right = node.right;
    if (left.type !== AST_NODE_TYPES.PrivateIdentifier) {
      const leftMatch = findTrackedIdentifierReference(left, trackedNames);
      if (leftMatch) return leftMatch;
    }
    return findTrackedIdentifierReference(right, trackedNames);
  }
  if (node.type === AST_NODE_TYPES.LogicalExpression) {
    const leftMatch = findTrackedIdentifierReference(node.left, trackedNames);
    if (leftMatch) return leftMatch;
    return findTrackedIdentifierReference(node.right, trackedNames);
  }
  if (node.type === AST_NODE_TYPES.ConditionalExpression) {
    const testMatch = findTrackedIdentifierReference(node.test, trackedNames);
    if (testMatch) return testMatch;
    const consequentMatch = findTrackedIdentifierReference(node.consequent, trackedNames);
    if (consequentMatch) return consequentMatch;
    return findTrackedIdentifierReference(node.alternate, trackedNames);
  }
  if (node.type === AST_NODE_TYPES.MemberExpression) {
    if (node.object.type !== AST_NODE_TYPES.Super) {
      const objectMatch = findTrackedIdentifierReference(node.object, trackedNames);
      if (objectMatch) return objectMatch;
    }
    if (node.computed) {
      return findTrackedIdentifierReference(node.property as TSESTree.Expression, trackedNames);
    }
    return null;
  }
  return null;
}

function collectCatchAliases(catchBody: TSESTree.BlockStatement, catchVarName: string): Set<string> {
  const aliases = new Set<string>();
  for (const statement of catchBody.body) {
    if (statement.type !== AST_NODE_TYPES.VariableDeclaration || statement.kind !== "const") continue;
    for (const decl of statement.declarations) {
      if (decl.id.type !== AST_NODE_TYPES.Identifier) continue;
      if (!decl.init || decl.init.type !== AST_NODE_TYPES.Identifier) continue;
      if (decl.init.name !== catchVarName) continue;
      aliases.add(decl.id.name);
    }
  }
  return aliases;
}

/**
 * Returns true when the second argument of `new Error(msg, options)` already
 * contains a `cause` property, regardless of the property value expression.
 * This avoids false positives for wrapped causes like `{ cause: ensureError(err) }`.
 * Accepts the forms:
 *   new Error(msg, { cause: catchVar })
 *   new Error(msg, { cause: catchVar, ... })
 */
function hasCauseProperty(optionsArg: TSESTree.Expression): boolean {
  if (optionsArg.type !== AST_NODE_TYPES.ObjectExpression) return false;
  return optionsArg.properties.some(prop => {
    if (prop.type !== AST_NODE_TYPES.Property) return false;
    const key = prop.key;
    const isKeyNamed = (key.type === AST_NODE_TYPES.Identifier && key.name === "cause") || (key.type === AST_NODE_TYPES.Literal && key.value === "cause");
    if (!isKeyNamed) return false;
    return true;
  });
}

export const requireErrorCauseInRethrowRule = createRule({
  name: "require-error-cause-in-rethrow",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require `{ cause: err }` when rethrowing a new Error inside a catch block that already references the caught variable. " +
        "Omitting { cause } silently discards the original stack trace and error chain, making post-mortem debugging harder.",
    },
    schema: [],
    messages: {
      missingCause: "`new Error(...)` inside catch ({{catchVar}}) references {{catchVar}} but omits `{ cause: {{catchVar}} }` — the original stack trace will be lost. Add `{ cause: {{catchVar}} }` as the second argument.",
      addCause: "Add `{ cause: {{catchVar}} }` as the second argument to preserve the original error chain.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const catchStack: CatchFrame[] = [];

    /** Returns the innermost active catch frame, or null. */
    function innermostCatch(): CatchFrame | null {
      if (catchStack.length === 0) return null;
      return catchStack[catchStack.length - 1] ?? null;
    }

    /**
     * Returns true if `node` is syntactically inside the try/catch clause
     * referenced by the current `catchStack` top-frame — i.e., we have not crossed
     * an intervening function boundary that would create a new execution context.
     */
    function isInsideCatchBody(node: TSESTree.Node): boolean {
      const frame = innermostCatch();
      if (!frame) return false;
      const ancestors = sourceCode.getAncestors(node);
      // Walk from innermost ancestor outward.
      // If we cross a non-arrow function boundary, the catch clause no longer
      // protects the node (deferred execution).
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const a = ancestors[i];
        if (a.type === AST_NODE_TYPES.FunctionDeclaration || a.type === AST_NODE_TYPES.FunctionExpression || a.type === AST_NODE_TYPES.ArrowFunctionExpression) {
          // Crossed a function boundary — outer catch no longer in scope for execution
          return false;
        }
        if (a.type === AST_NODE_TYPES.CatchClause) {
          return true;
        }
      }
      return false;
    }

    return {
      CatchClause(node) {
        const param = node.param;
        if (!param || param.type !== AST_NODE_TYPES.Identifier) {
          // Bare catch {} or destructured — push empty sentinel so CatchClause:exit still pops
          catchStack.push({ varName: "", aliases: new Set() });
          return;
        }
        catchStack.push({ varName: param.name, aliases: collectCatchAliases(node.body, param.name) });
      },

      "CatchClause:exit"() {
        catchStack.pop();
      },

      NewExpression(node) {
        if (node.parent?.type !== AST_NODE_TYPES.ThrowStatement || node.parent.argument !== node) return;

        // Only flag `new Error(...)` — not subclasses like `new TypeError(...)`.
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.Identifier || callee.name !== "Error") return;

        const frame = innermostCatch();
        if (!frame || !frame.varName) return;
        if (!isInsideCatchBody(node)) return;

        const catchVarName = frame.varName;
        const trackedNames = new Set<string>([catchVarName, ...frame.aliases]);
        const args = node.arguments;

        // Must have at least a message argument that references the catch variable.
        if (args.length === 0) return;
        const msgArg = args[0];
        if (msgArg.type === "SpreadElement") return;

        const causeIdentifier = findTrackedIdentifierReference(msgArg, trackedNames);
        if (!causeIdentifier) return;

        // If a second argument exists, check that it contains { cause: catchVar }.
        if (args.length >= 2) {
          const secondArg = args[1];
          if (secondArg.type !== "SpreadElement" && hasCauseProperty(secondArg)) {
            return; // Already has cause — no violation
          }
        }

        context.report({
          node,
          messageId: "missingCause",
          data: { catchVar: causeIdentifier },
          suggest: [
            {
              messageId: "addCause" as const,
              data: { catchVar: causeIdentifier },
              fix(fixer) {
                // If there's already a second arg, replace it with an object that adds cause.
                if (args.length >= 2) {
                  const secondArg = args[1];
                  if (secondArg.type === "SpreadElement") return null;
                  if (secondArg.type === AST_NODE_TYPES.ObjectExpression) {
                    // Add cause property to the existing object
                    if (secondArg.properties.length === 0) {
                      return fixer.replaceText(secondArg, `{ cause: ${causeIdentifier} }`);
                    }
                    const firstProp = secondArg.properties[0];
                    return fixer.insertTextBefore(firstProp, `cause: ${causeIdentifier}, `);
                  }
                  return null;
                }
                // No second argument — append `, { cause: catchVar }` before closing paren
                const lastArg = args[args.length - 1];
                if (!lastArg || lastArg.type === "SpreadElement") return null;
                return fixer.insertTextAfter(lastArg, `, { cause: ${causeIdentifier} }`);
              },
            },
          ],
        });
      },
    };
  },
});
