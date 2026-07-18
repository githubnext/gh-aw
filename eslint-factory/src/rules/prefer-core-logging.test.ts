import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { preferCoreLoggingRule } from "./prefer-core-logging";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("prefer-core-logging", () => {
  it("uses the correct docs URL", () => {
    expect(preferCoreLoggingRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#prefer-core-logging");
  });

  it("hasSuggestions enabled", () => {
    expect(preferCoreLoggingRule.meta.hasSuggestions).toBe(true);
  });

  it("invalid: plain console.log with no core in scope is now flagged", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `console.log("hello");`,
          errors: [
            { messageId: "preferCoreLogging", data: { method: "log", replacement: "core.info" }, suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `"hello"` }, output: `core.info("hello");` }] },
          ],
        },
        {
          code: `console.error("bad thing");`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "error", replacement: "core.error" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.error", args: `"bad thing"` }, output: `core.error("bad thing");` }],
            },
          ],
        },
        {
          code: `const foo = "bar"; console.log(foo);`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "log", replacement: "core.info" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `foo` }, output: `const foo = "bar"; core.info(foo);` }],
            },
          ],
        },
      ],
    });
  });

  it("valid: core.info calls are fine", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [
        `const core = require("@actions/core"); core.info("hello");`,
        `const core = require("@actions/core"); core.error("bad");`,
        `const core = require("@actions/core"); core.warning("warn");`,
        `const core = require("@actions/core"); core.debug("debug");`,
      ],
      invalid: [],
    });
  });

  it("invalid: console.log in a function where core is not declared", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `function helper() { console.log("no core here"); }`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "log", replacement: "core.info" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `"no core here"` }, output: `function helper() { core.info("no core here"); }` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.log when core is in scope via require", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `const core = require("@actions/core"); console.log("hello");`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "log", replacement: "core.info" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `"hello"` }, output: `const core = require("@actions/core"); core.info("hello");` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.error when core is in scope", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `const core = require("@actions/core"); console.error("bad thing");`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "error", replacement: "core.error" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.error", args: `"bad thing"` }, output: `const core = require("@actions/core"); core.error("bad thing");` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.warn when core is in scope", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `const core = require("@actions/core"); console.warn("warning");`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "warn", replacement: "core.warning" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.warning", args: `"warning"` }, output: `const core = require("@actions/core"); core.warning("warning");` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.debug when core is in scope", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `const core = require("@actions/core"); console.debug("verbose");`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "debug", replacement: "core.debug" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.debug", args: `"verbose"` }, output: `const core = require("@actions/core"); core.debug("verbose");` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.log when core is a function parameter", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `async function run(core) { console.log("done"); }`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "log", replacement: "core.info" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `"done"` }, output: `async function run(core) { core.info("done"); }` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: console.log with multi-arg call when core is in scope", () => {
    ruleTester.run("prefer-core-logging", preferCoreLoggingRule, {
      valid: [],
      invalid: [
        {
          code: `const core = require("@actions/core"); const someVar = 1; console.log("value:", someVar);`,
          errors: [
            {
              messageId: "preferCoreLogging",
              data: { method: "log", replacement: "core.info" },
              suggestions: [{ messageId: "replaceWithCoreMethod", data: { replacement: "core.info", args: `"value:", someVar` }, output: `const core = require("@actions/core"); const someVar = 1; core.info("value:", someVar);` }],
            },
          ],
        },
      ],
    });
  });
});
