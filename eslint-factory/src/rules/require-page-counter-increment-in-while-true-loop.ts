import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const requirePageCounterIncrementInWhileTrueLoopRule = createRule({
  name: "require-page-counter-increment-in-while-true-loop",
  meta: {
    type: "problem",
    docs: {
      description: "Require page counters used by manual `while (true)` pagination loops to be incremented or reassigned.",
    },
    schema: [],
    messages: {
      requirePageCounterIncrement: "Page counter '{{name}}' is used in this `while (true)` loop but is never incremented or reassigned. The loop can repeatedly fetch the same page.",
    },
  },
  defaultOptions: [],
  create(context) {
    const assignments: TSESTree.AssignmentExpression[] = [];
    const breaks: TSESTree.BreakStatement[] = [];
    const functions: (TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression)[] = [];
    const identifiers: TSESTree.Identifier[] = [];
    const updates: TSESTree.UpdateExpression[] = [];
    const sourceCode = context.sourceCode;

    function isWithin(node: TSESTree.Node, container: TSESTree.Node): boolean {
      return node.range[0] >= container.range[0] && node.range[1] <= container.range[1];
    }

    function isInNestedFunction(node: TSESTree.Node, loop: TSESTree.WhileStatement): boolean {
      return functions.some(fn => isWithin(fn, loop.body) && isWithin(node, fn));
    }

    function getCountersImmediatelyBefore(loop: TSESTree.WhileStatement): TSESTree.VariableDeclarator[] {
      const parent = loop.parent;
      if (parent?.type !== "BlockStatement" && parent?.type !== "Program") return [];

      const index = parent.body.indexOf(loop);
      const previous = parent.body[index - 1];
      if (previous?.type !== "VariableDeclaration" || previous.kind !== "let") return [];

      return previous.declarations.filter(declaration => declaration.id.type === "Identifier" && declaration.init?.type === "Literal" && typeof declaration.init.value === "number");
    }

    function isCounterIdentifier(identifier: TSESTree.Identifier, counter: TSESTree.VariableDeclarator): boolean {
      if (
        (identifier.parent?.type === "Property" && identifier.parent.key === identifier && !identifier.parent.computed && !identifier.parent.shorthand) ||
        (identifier.parent?.type === "MemberExpression" && identifier.parent.property === identifier && !identifier.parent.computed)
      ) {
        return false;
      }

      const counterIdentifier = counter.id as TSESTree.Identifier;
      const counterVariable = sourceCode.getDeclaredVariables(counter).find(variable => variable.identifiers.includes(counterIdentifier));
      if (!counterVariable || identifier.name !== counterIdentifier.name) return false;

      for (let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(identifier); scope; scope = scope.upper) {
        const variable = scope.set.get(identifier.name);
        if (variable) return variable === counterVariable;
      }
      return false;
    }

    function hasCounterReference(loop: TSESTree.WhileStatement, counter: TSESTree.VariableDeclarator): boolean {
      return identifiers.some(node => isWithin(node, loop.body) && isCounterIdentifier(node, counter));
    }

    function hasBreak(loop: TSESTree.WhileStatement): boolean {
      return breaks.some(node => isWithin(node, loop.body) && !isInNestedFunction(node, loop));
    }

    function isCounterAdvanceAssignment(node: TSESTree.AssignmentExpression, counter: TSESTree.VariableDeclarator): boolean {
      if (node.operator === "+=") return true;
      if (node.operator !== "=" || node.right.type === "Literal" || (node.right.type === "Identifier" && isCounterIdentifier(node.right, counter))) return false;
      return node.right.type !== "BinaryExpression" || node.right.operator !== "-" || ![node.right.left, node.right.right].some(operand => operand.type === "Identifier" && isCounterIdentifier(operand, counter));
    }

    function hasCounterAdvance(loop: TSESTree.WhileStatement, counter: TSESTree.VariableDeclarator): boolean {
      return (
        updates.some(node => node.operator === "++" && node.argument.type === "Identifier" && isCounterIdentifier(node.argument, counter) && isWithin(node, loop.body) && !isInNestedFunction(node, loop)) ||
        assignments.some(node => node.left.type === "Identifier" && isCounterIdentifier(node.left, counter) && isCounterAdvanceAssignment(node, counter) && isWithin(node, loop.body) && !isInNestedFunction(node, loop))
      );
    }

    return {
      AssignmentExpression(node) {
        assignments.push(node);
      },
      ArrowFunctionExpression(node) {
        functions.push(node);
      },
      BreakStatement(node) {
        breaks.push(node);
      },
      FunctionDeclaration(node) {
        functions.push(node);
      },
      FunctionExpression(node) {
        functions.push(node);
      },
      Identifier(node) {
        identifiers.push(node);
      },
      UpdateExpression(node) {
        updates.push(node);
      },
      "WhileStatement:exit"(node) {
        if (node.test.type !== "Literal" || node.test.value !== true || node.body.type !== "BlockStatement" || !hasBreak(node)) return;

        for (const counter of getCountersImmediatelyBefore(node)) {
          const name = (counter.id as TSESTree.Identifier).name;
          if (hasCounterReference(node, counter) && !hasCounterAdvance(node, counter)) {
            context.report({
              node: counter,
              messageId: "requirePageCounterIncrement",
              data: { name },
            });
          }
        }
      },
    };
  },
});
