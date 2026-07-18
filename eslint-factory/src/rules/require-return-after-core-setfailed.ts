import { AST_NODE_TYPES, AST_TOKEN_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier, isDestructuredCoreMethodIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

/**
 * Returns true when the statement unconditionally transfers control out of the
 * current block: `return`, `throw`, `break`, `continue`, or `process.exit(...)`.
 *
 * Known limitation: `break` and `continue` only exit the innermost loop/switch
 * iteration — they do not prevent post-loop or post-switch statements from
 * running in a failed state. The rule accepts them to cover the common loop
 * guard pattern; detecting post-loop continuation is out of scope.
 */
function isControlTransfer(node: TSESTree.Statement): boolean {
  const matchesControlTransferType = node.type === AST_NODE_TYPES.ReturnStatement || node.type === AST_NODE_TYPES.ThrowStatement || node.type === AST_NODE_TYPES.BreakStatement || node.type === AST_NODE_TYPES.ContinueStatement;
  if (matchesControlTransferType) {
    return true;
  }
  // process.exit(...)
  if (node.type === AST_NODE_TYPES.ExpressionStatement && node.expression.type === AST_NODE_TYPES.CallExpression) {
    const callee = node.expression.callee;
    if (
      callee.type === AST_NODE_TYPES.MemberExpression &&
      !callee.computed &&
      callee.object.type === AST_NODE_TYPES.Identifier &&
      callee.object.name === "process" &&
      callee.property.type === AST_NODE_TYPES.Identifier &&
      callee.property.name === "exit"
    ) {
      return true;
    }
  }
  return false;
}

/**
 * Checks a list of sequential statements for `core.setFailed(...)` calls that
 * are not immediately followed by a control-transfer statement.
 */
function checkStatementList(stmts: TSESTree.Statement[], isSetFailed: (node: TSESTree.Statement) => boolean, report: (node: TSESTree.Statement, next: TSESTree.Statement) => void): void {
  for (let i = 0; i < stmts.length; i++) {
    const stmt = stmts[i];
    if (!isSetFailed(stmt)) continue;
    const next = stmts[i + 1];
    if (next && !isControlTransfer(next)) {
      report(stmt, next);
    }
  }
}

function isExecutableStatement(node: TSESTree.ProgramStatement): node is TSESTree.Statement {
  return (
    node.type !== AST_NODE_TYPES.ImportDeclaration &&
    node.type !== AST_NODE_TYPES.ExportAllDeclaration &&
    node.type !== AST_NODE_TYPES.ExportDefaultDeclaration &&
    node.type !== AST_NODE_TYPES.ExportNamedDeclaration &&
    node.type !== AST_NODE_TYPES.TSModuleDeclaration
  );
}

function isCallExpressionStatement(node: TSESTree.Statement): boolean {
  return node.type === AST_NODE_TYPES.ExpressionStatement && node.expression.type === AST_NODE_TYPES.CallExpression;
}

function isLoopStatement(node: TSESTree.Node): node is TSESTree.WhileStatement | TSESTree.DoWhileStatement | TSESTree.ForStatement | TSESTree.ForInStatement | TSESTree.ForOfStatement {
  return (
    node.type === AST_NODE_TYPES.WhileStatement || node.type === AST_NODE_TYPES.DoWhileStatement || node.type === AST_NODE_TYPES.ForStatement || node.type === AST_NODE_TYPES.ForInStatement || node.type === AST_NODE_TYPES.ForOfStatement
  );
}

/**
 * When `core.setFailed()` is the last statement in a nested block, searches
 * ancestor statement lists for the first subsequent statement that would execute
 * after the nested block exits. Returns that statement, or `null` if no such
 * statement exists before a function boundary.
 *
 * This detects patterns such as:
 *   if (x) { core.setFailed(...); }  // setFailed is last in the if-block
 *   doMore();                         // still runs in a failed state
 */
function findContinuationOutsideBlock(setFailedNode: TSESTree.Statement, ancestors: TSESTree.Node[]): TSESTree.Statement | null {
  let childNode: TSESTree.Node = setFailedNode;

  for (let i = ancestors.length - 1; i >= 0; i--) {
    const ancestor = ancestors[i];

    // Stop at function boundaries — the caller's scope is never reached
    if (ancestor.type === AST_NODE_TYPES.FunctionDeclaration || ancestor.type === AST_NODE_TYPES.FunctionExpression || ancestor.type === AST_NODE_TYPES.ArrowFunctionExpression) {
      return null;
    }

    let stmts: readonly TSESTree.Statement[] | null = null;
    if (ancestor.type === AST_NODE_TYPES.BlockStatement) {
      stmts = ancestor.body;
    } else if (ancestor.type === AST_NODE_TYPES.SwitchCase) {
      stmts = ancestor.consequent;
    } else if (ancestor.type === AST_NODE_TYPES.Program) {
      stmts = ancestor.body.filter(isExecutableStatement);
    }

    if (stmts !== null) {
      const idx = stmts.findIndex(s => s === childNode);
      if (idx >= 0 && idx < stmts.length - 1) {
        return stmts[idx + 1];
      }
    }

    // Loop back-edges: reaching the end of an enclosing loop body may begin the
    // next iteration, so execution can continue even without later siblings.
    const isEnclosingLoopWithBody =
      (ancestor.type === AST_NODE_TYPES.WhileStatement ||
        ancestor.type === AST_NODE_TYPES.DoWhileStatement ||
        ancestor.type === AST_NODE_TYPES.ForStatement ||
        ancestor.type === AST_NODE_TYPES.ForInStatement ||
        ancestor.type === AST_NODE_TYPES.ForOfStatement) &&
      ancestor.body === childNode;
    if (isEnclosingLoopWithBody) {
      return ancestor;
    }

    // Switch fall-through: after one case is exhausted, control may continue
    // into subsequent cases when no terminating statement is encountered.
    if (ancestor.type === AST_NODE_TYPES.SwitchStatement && childNode.type === AST_NODE_TYPES.SwitchCase) {
      const currentIndex = ancestor.cases.findIndex(testCase => testCase === childNode);
      if (currentIndex >= 0) {
        for (let nextCaseIndex = currentIndex + 1; nextCaseIndex < ancestor.cases.length; nextCaseIndex++) {
          const nextCase = ancestor.cases[nextCaseIndex];
          const nextStmt = nextCase.consequent[0];
          if (nextStmt) {
            return nextStmt;
          }
        }
      }
    }

    childNode = ancestor;
  }

  return null;
}

export const requireReturnAfterCoreSetFailedRule = createRule({
  name: "require-return-after-core-setfailed",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require a return, throw, break, continue, or process.exit() statement immediately after core.setFailed() to prevent execution from continuing after a failure is declared. " +
        "core.setFailed() only marks the action as failed at the end; it does not stop execution. " +
        "Note: break and continue are accepted as control-transfer statements within loop/switch bodies, but they do not prevent post-loop or post-switch code from running in a failed state (known limitation).",
    },
    schema: [],
    messages: {
      missingReturnAfterSetFailed:
        "core.setFailed() does not stop execution — add a control-transfer statement (for example: return, throw, break, continue, or process.exit(...)) immediately after to prevent the action from continuing in a failed state.",
      addReturn: "Add 'return;' after core.setFailed() to stop execution.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    /**
     * Returns true when the statement is a call to core.setFailed(...) in any
     * recognized form:
     *  - Direct non-computed: core.setFailed(...)
     *  - Computed string literal: core["setFailed"](...)
     *  - Aliased object: const c = core; c.setFailed(...)
     *  - Destructured binding: const { setFailed } = core; setFailed(...)
     */
    function isCoreSetFailedStatement(node: TSESTree.Statement): node is TSESTree.ExpressionStatement {
      if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
      const expr = node.expression;
      if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
      const callee = expr.callee;

      if (callee.type === AST_NODE_TYPES.MemberExpression) {
        const obj = callee.object;
        const prop = callee.property;
        const isNonComputedSetFailed = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "setFailed";
        const isComputedSetFailed = callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "setFailed";
        if ((isNonComputedSetFailed || isComputedSetFailed) && obj.type === AST_NODE_TYPES.Identifier) {
          if (isCoreLikeIdentifier(obj.name)) return true;
          if (isCoreAliasIdentifier(obj, sourceCode)) return true;
        }
      }

      if (callee.type === AST_NODE_TYPES.Identifier) {
        if (isDestructuredCoreMethodIdentifier(callee, "setFailed", sourceCode)) return true;
      }

      return false;
    }

    function isInsideFunctionLike(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];
        const isFunctionLike = ancestor.type === AST_NODE_TYPES.FunctionDeclaration || ancestor.type === AST_NODE_TYPES.FunctionExpression || ancestor.type === AST_NODE_TYPES.ArrowFunctionExpression;
        if (isFunctionLike) {
          return true;
        }
      }
      return false;
    }

    function report(node: TSESTree.Statement, next: TSESTree.Statement): void {
      const parent = node.parent;
      const isBlockTerminalCallCleanup = parent?.type === AST_NODE_TYPES.BlockStatement && isCallExpressionStatement(next) && parent.body.at(-2) === node && parent.body.at(-1) === next;

      context.report({
        node,
        messageId: "missingReturnAfterSetFailed",
        suggest:
          isInsideFunctionLike(node) && !isBlockTerminalCallCleanup
            ? [
                {
                  messageId: "addReturn",
                  fix(fixer) {
                    const isOnSameLine = next.loc.start.line === node.loc.end.line;
                    if (isOnSameLine) {
                      return fixer.insertTextBefore(next, "return; ");
                    }
                    const line = sourceCode.lines[next.loc.start.line - 1] ?? "";
                    const indent = /^(\s*)/.exec(line)?.[1] ?? "";
                    return fixer.insertTextBefore(next, `return;\n${indent}`);
                  },
                },
              ]
            : undefined,
      });
    }

    // Used for cross-block fall-through: inserts return; after the setFailed node
    // inside its block rather than before a statement in an outer block.
    function getDirectControlBodyContainer(node: TSESTree.Statement): TSESTree.IfStatement | TSESTree.WhileStatement | TSESTree.DoWhileStatement | TSESTree.ForStatement | TSESTree.ForInStatement | TSESTree.ForOfStatement | null {
      const parent = node.parent;
      if (parent?.type === AST_NODE_TYPES.IfStatement && (parent.consequent === node || parent.alternate === node)) {
        return parent;
      }
      if (parent !== null && isLoopStatement(parent) && parent.body === node) {
        return parent;
      }
      return null;
    }

    function reportNested(node: TSESTree.Statement): void {
      context.report({
        node,
        messageId: "missingReturnAfterSetFailed",
        suggest: isInsideFunctionLike(node)
          ? [
              {
                messageId: "addReturn",
                fix(fixer) {
                  const directControlBodyContainer = getDirectControlBodyContainer(node);
                  if (directControlBodyContainer !== null) {
                    return fixer.replaceText(node, `{ ${sourceCode.getText(node)} return; }`);
                  }

                  const nextTokenOrComment = sourceCode.getTokenAfter(node, { includeComments: true });
                  const hasTrailingCommentOnSameLine =
                    nextTokenOrComment !== null && (nextTokenOrComment.type === AST_TOKEN_TYPES.Line || nextTokenOrComment.type === AST_TOKEN_TYPES.Block) && nextTokenOrComment.loc.start.line === node.loc.end.line;
                  if (hasTrailingCommentOnSameLine) {
                    const line = sourceCode.lines[node.loc.start.line - 1] ?? "";
                    const indent = /^(\s*)/.exec(line)?.[1] ?? "";
                    return fixer.insertTextAfter(nextTokenOrComment, `\n${indent}return;`);
                  }

                  const nextToken = sourceCode.getTokenAfter(node);
                  if (nextToken && nextToken.loc.start.line === node.loc.end.line) {
                    // Next token (e.g. closing '}') is on the same line — insert inline
                    return fixer.insertTextAfter(node, " return;");
                  }
                  const line = sourceCode.lines[node.loc.start.line - 1] ?? "";
                  const indent = /^(\s*)/.exec(line)?.[1] ?? "";
                  return fixer.insertTextAfter(node, `\n${indent}return;`);
                },
              },
            ]
          : undefined,
      });
    }

    function checkNestedContinuation(stmt: TSESTree.Statement): void {
      const ancestors = sourceCode.getAncestors(stmt);
      const continuation = findContinuationOutsideBlock(stmt, ancestors);
      if (continuation !== null && !isControlTransfer(continuation)) {
        reportNested(stmt);
      }
    }

    function checkDirectControlBody(stmt: TSESTree.Statement | null): void {
      if (stmt !== null && stmt.type !== AST_NODE_TYPES.BlockStatement && isCoreSetFailedStatement(stmt)) {
        checkNestedContinuation(stmt);
      }
    }

    return {
      IfStatement(node: TSESTree.IfStatement) {
        for (const branch of [node.consequent, node.alternate]) {
          checkDirectControlBody(branch);
        }
      },
      // Check statement blocks: if body, else body, while body, function body, etc.
      BlockStatement(node: TSESTree.BlockStatement) {
        checkStatementList(node.body, isCoreSetFailedStatement, report);

        // Cross-block fall-through: when core.setFailed() is the last statement of
        // this block, check whether any enclosing block has subsequent statements.
        // Example (invalid): if (x) { core.setFailed(...); }  doMore();
        const lastStmt = node.body[node.body.length - 1];
        if (lastStmt && isCoreSetFailedStatement(lastStmt)) {
          checkNestedContinuation(lastStmt);
        }
      },
      DoWhileStatement(node: TSESTree.DoWhileStatement) {
        checkDirectControlBody(node.body);
      },
      ForInStatement(node: TSESTree.ForInStatement) {
        checkDirectControlBody(node.body);
      },
      ForOfStatement(node: TSESTree.ForOfStatement) {
        checkDirectControlBody(node.body);
      },
      ForStatement(node: TSESTree.ForStatement) {
        checkDirectControlBody(node.body);
      },
      // Handle single-statement arrow functions (no braces) — rare but safe to skip
      // The main case is BlockStatement above.
      SwitchCase(node: TSESTree.SwitchCase) {
        checkStatementList(node.consequent, isCoreSetFailedStatement, report);
      },
      WhileStatement(node: TSESTree.WhileStatement) {
        checkDirectControlBody(node.body);
      },
      Program(node: TSESTree.Program) {
        checkStatementList(node.body.filter(isExecutableStatement), isCoreSetFailedStatement, report);
      },
    };
  },
});
