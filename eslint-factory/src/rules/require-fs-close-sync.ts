import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { createFsSyncMethodResolver } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const FS_METHODS = new Set(["openSync", "closeSync"]);

type ScopeType = ReturnType<TSESLint.SourceCode["getScope"]>;
type ScopeVariable = ScopeType["variables"][number];

type OpenDescriptor = {
  variable: ScopeVariable;
  openCall: TSESTree.CallExpression;
  fdName: string;
  closed: boolean;
};

type FunctionFrame = {
  opens: OpenDescriptor[];
};

function findVariableByName(sourceCode: Readonly<TSESLint.SourceCode>, node: TSESTree.Node, varName: string): ScopeVariable | undefined {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
  while (scope) {
    const variable = scope.set.get(varName);
    if (variable) return variable;
    scope = scope.upper;
  }
  return undefined;
}

function getFdIdentifierFromOpenCall(node: TSESTree.CallExpression): string | null {
  const parent = node.parent;

  if (parent?.type === AST_NODE_TYPES.VariableDeclarator && parent.init === node && parent.id.type === AST_NODE_TYPES.Identifier) {
    return parent.id.name;
  }

  if (parent?.type === AST_NODE_TYPES.AssignmentExpression && parent.operator === "=" && parent.right === node && parent.left.type === AST_NODE_TYPES.Identifier) {
    return parent.left.name;
  }

  return null;
}

export const requireFsCloseSyncRule = createRule({
  name: "require-fs-close-sync",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require file descriptors returned by fs.openSync(...) in actions/setup/js scripts to be closed with fs.closeSync(fd) in the same enclosing function. " +
        "An unclosed descriptor leaks a file handle for the process lifetime and can eventually trigger EMFILE failures in unrelated I/O. " +
        "Scope: this rule checks variable-bound openSync calls (`const fd = fs.openSync(...)` and `fd = fs.openSync(...)`) and accepts any fs.closeSync(fd) within the same enclosing function body. " +
        "It intentionally does not analyze destructured bindings, inline argument forms, or cross-function close patterns.",
    },
    schema: [],
    messages: {
      missingCloseSync: "File descriptor '{{fdName}}' from fs.openSync(...) is never closed in this function. " + "Add fs.closeSync({{fdName}}) to avoid leaking file descriptors (EMFILE risk).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const resolveFsMethod = createFsSyncMethodResolver(sourceCode, FS_METHODS, { allowUnboundFsIdentifier: true });
    const frameStack: FunctionFrame[] = [];

    function pushFrame() {
      frameStack.push({ opens: [] });
    }

    function popFrame() {
      const frame = frameStack.pop();
      if (!frame) return;

      for (const openDescriptor of frame.opens) {
        if (openDescriptor.closed) continue;
        context.report({
          node: openDescriptor.openCall,
          messageId: "missingCloseSync",
          data: { fdName: openDescriptor.fdName },
        });
      }
    }

    return {
      Program() {
        pushFrame();
      },
      "Program:exit"() {
        popFrame();
      },
      FunctionDeclaration() {
        pushFrame();
      },
      "FunctionDeclaration:exit"() {
        popFrame();
      },
      FunctionExpression() {
        pushFrame();
      },
      "FunctionExpression:exit"() {
        popFrame();
      },
      ArrowFunctionExpression() {
        pushFrame();
      },
      "ArrowFunctionExpression:exit"() {
        popFrame();
      },
      CallExpression(node: TSESTree.CallExpression) {
        const methodName = resolveFsMethod(node);
        if (!methodName) return;

        if (methodName === "openSync") {
          const fdName = getFdIdentifierFromOpenCall(node);
          if (!fdName) return;
          const variable = findVariableByName(sourceCode, node, fdName);
          if (!variable) return;
          const frame = frameStack[frameStack.length - 1];
          if (!frame) return;
          frame.opens.push({ variable, openCall: node, fdName, closed: false });
          return;
        }

        if (methodName === "closeSync") {
          const firstArg = node.arguments[0];
          if (!firstArg || firstArg.type !== AST_NODE_TYPES.Identifier) return;
          const variable = findVariableByName(sourceCode, firstArg, firstArg.name);
          if (!variable) return;

          const candidates: OpenDescriptor[] = [];
          for (const frame of frameStack) {
            for (const openDescriptor of frame.opens) {
              if (openDescriptor.closed) continue;
              if (openDescriptor.variable !== variable) continue;
              if (openDescriptor.openCall.range[0] >= node.range[0]) continue;
              candidates.push(openDescriptor);
            }
          }

          if (candidates.length === 0) return;

          let candidateToClose = candidates[0];
          for (const candidate of candidates) {
            if (candidate.openCall.range[0] > candidateToClose.openCall.range[0]) {
              candidateToClose = candidate;
            }
          }
          candidateToClose.closed = true;
        }
      },
    };
  },
});
