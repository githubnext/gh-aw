import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noAsyncForEachCallbackRule } from "./no-async-foreach-callback";

const ruleTester = new RuleTester({ languageOptions: { ecmaVersion: 2022, sourceType: "commonjs" } });

describe("no-async-foreach-callback", () => {
  it("uses the correct docs URL", () => {
    expect(noAsyncForEachCallbackRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-async-foreach-callback");
  });

  it("valid and invalid cases", () => {
    ruleTester.run("no-async-foreach-callback", noAsyncForEachCallbackRule, {
      valid: [
        `items.forEach(x => use(x));`,
        `Promise.all(items.map(async x => fetchIt(x)));`,
        `items.forEach(function (x) { use(x); });`,
        `items.forEach(handler);`,
        `items.forEach();`,
        `items[method](async x => {});`,
        `items[\`map\`](async x => {});`,
        `items[\`forEach\`](x => use(x));`,
        `items[\`for\${method}\`](async x => {});`,
        `function handler(x) { use(x); } items.forEach(handler);`,
        `const handler = x => use(x); items.forEach(handler);`,
        `function run(handler) { items.forEach(handler); }`,
        `async function handler() {} function run(handler) { items.forEach(handler); }`,
        `async function handler() {} handler = syncHandler; items.forEach(handler);`,
        `const handler = async () => {}; handler = syncHandler; items.forEach(handler);`,
        `let handler = async () => {}; items.forEach(handler);`,
        `var handler = async () => {}; items.forEach(handler);`,
        `async function handler() {} function handler() {} items.forEach(handler);`,
        `const { handler } = callbacks; items.forEach(handler);`,
        `const [handler] = [async () => {}]; items.forEach(handler);`,
        `const handler = factory(); items.forEach(handler);`,
        `const handler = async () => {}; { const handler = () => {}; items.forEach(handler); }`,
        { code: `import handler from "handlers"; items.forEach(handler);`, languageOptions: { sourceType: "module" } },
      ],
      invalid: [
        { code: `items.forEach(async x => { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `a.b.forEach(async function (x) { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `items["forEach"](async x => { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `items['forEach'](async function (x) { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `items[\`forEach\`](async x => { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `items[\`for\\u0045ach\`](async function (x) { await f(x); });`, errors: [{ messageId: "asyncForEach" }] },
        { code: `const handler = async x => { await f(x); }; items[\`forEach\`](handler);`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
        { code: `async function handler(x) { await f(x); } items.forEach(handler);`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
        { code: `const handler = async x => { await f(x); }; items["forEach"](handler);`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
        { code: `const handler = async function (x) { await f(x); }; function run() { items.forEach(handler); }`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
        { code: `items.forEach(handler); async function handler() {}`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
        { code: `function run() { const handler = async () => {}; { items.forEach(handler); } }`, errors: [{ messageId: "asyncForEach", type: "Identifier" }] },
      ],
    });
  });
});
