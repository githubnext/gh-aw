import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { preferStructuredCloneRule } from "./prefer-structured-clone";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("prefer-structured-clone", () => {
  it("uses the correct docs URL", () => {
    expect(preferStructuredCloneRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#prefer-structured-clone");
  });

  it("valid: unrelated JSON.parse / JSON.stringify usages are accepted", () => {
    cjsRuleTester.run("prefer-structured-clone", preferStructuredCloneRule, {
      valid: [
        `const data = JSON.parse(rawText);`,
        `const text = JSON.stringify(obj);`,
        `structuredClone(obj);`,
        // Replacer/indent argument changes stringify semantics; excluded to avoid false positives.
        `const clone = JSON.parse(JSON.stringify(obj, null, 2));`,
        // Custom reviver on parse changes semantics too; still matched today since only the
        // parse call itself is checked for a single stringify argument — but the inner
        // stringify call must have exactly one argument, so this is out of scope here.
        `const clone = JSON.parse(JSON.stringify(obj), reviver);`,
      ],
      invalid: [],
    });
  });

  it("invalid: JSON.parse(JSON.stringify(x)) is flagged and suggests structuredClone(x)", () => {
    cjsRuleTester.run("prefer-structured-clone", preferStructuredCloneRule, {
      valid: [],
      invalid: [
        {
          code: `const clone = JSON.parse(JSON.stringify(tool));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const clone = structuredClone(tool);`,
                },
              ],
            },
          ],
        },
        {
          code: `const runs = state.runs.map(run => JSON.parse(JSON.stringify(run)));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const runs = state.runs.map(run => structuredClone(run));`,
                },
              ],
            },
          ],
        },
        {
          code: `const clone = JSON["parse"](JSON["stringify"](tool));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const clone = structuredClone(tool);`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: no suggestion when the cloned identifier carries function-valued properties", () => {
    cjsRuleTester.run("prefer-structured-clone", preferStructuredCloneRule, {
      valid: [],
      invalid: [
        {
          // Shape of actions/setup/js/safe_outputs_tools_loader.cjs: the JSON round-trip
          // intentionally drops `tool.handler`, which is re-attached afterwards.
          // structuredClone would throw DataCloneError here.
          code: [
            `function attachHandlers(tools, handlers) {`,
            `  tools.forEach(tool => {`,
            `    tool.handler = args => handlers.defaultHandler(args);`,
            `  });`,
            `}`,
            `function register(tool) {`,
            `  const toolToRegister = JSON.parse(JSON.stringify(tool));`,
            `  if (tool.handler) {`,
            `    toolToRegister.handler = tool.handler;`,
            `  }`,
            `  return toolToRegister;`,
            `}`,
          ].join("\n"),
          errors: [{ messageId: "preferStructuredClone", suggestions: [] }],
        },
        {
          code: [`function register(tool) {`, `  if (typeof tool.handler === "function") {`, `  }`, `  return JSON.parse(JSON.stringify(tool));`, `}`].join("\n"),
          errors: [{ messageId: "preferStructuredClone", suggestions: [] }],
        },
        {
          code: [`const config = { handler: () => {} };`, `const clone = JSON.parse(JSON.stringify(config));`].join("\n"),
          errors: [{ messageId: "preferStructuredClone", suggestions: [] }],
        },
        {
          // JSON-sourced tool (actions/setup/js/generate_safe_outputs_tools.cjs shape):
          // no function-valued property anywhere, so the suggestion is still offered.
          code: [`const tools = JSON.parse(readFileSync(path, "utf8"));`, `for (const tool of tools) {`, `  const enhancedTool = JSON.parse(JSON.stringify(tool));`, `}`].join("\n"),
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: [`const tools = JSON.parse(readFileSync(path, "utf8"));`, `for (const tool of tools) {`, `  const enhancedTool = structuredClone(tool);`, `}`].join("\n"),
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
