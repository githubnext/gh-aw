import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireNanCheckAfterEnvNumericParseRule } from "./require-nan-check-after-env-numeric-parse";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

const esmRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
  },
});

describe("require-nan-check-after-env-numeric-parse", () => {
  it("uses the correct docs URL", () => {
    expect(requireNanCheckAfterEnvNumericParseRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-nan-check-after-env-numeric-parse");
  });

  it("valid: parseInt from process.env validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const maxRuns = parseInt(process.env.MAX_RUNS, 10); if (Number.isNaN(maxRuns)) throw new Error("invalid MAX_RUNS");`, `const port = parseInt(process.env.PORT, 10); if (!Number.isNaN(port)) startServer(port);`],
      invalid: [],
    });
  });

  it("valid: parseInt from process.env validated with global isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const maxRuns = parseInt(process.env.MAX_RUNS, 10); if (isNaN(maxRuns)) throw new Error("invalid MAX_RUNS");`],
      invalid: [],
    });
  });

  it("valid: Number.parseInt from process.env validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const port = Number.parseInt(process.env.PORT, 10); if (Number.isNaN(port)) throw new Error("invalid PORT");`],
      invalid: [],
    });
  });

  it("valid: Number.parseFloat from process.env validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const delay = Number.parseFloat(process.env.DELAY); if (Number.isNaN(delay)) throw new Error("invalid DELAY");`],
      invalid: [],
    });
  });

  it("valid: Number() from process.env validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const runId = Number(process.env.RUN_ID); if (Number.isNaN(runId)) throw new Error("invalid RUN_ID");`],
      invalid: [],
    });
  });

  it("valid: parseFloat from process.env validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const rate = parseFloat(process.env.RATE); if (Number.isNaN(rate)) throw new Error("invalid RATE");`],
      invalid: [],
    });
  });

  it("valid: parseInt without process.env is not flagged", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const x = parseInt("42", 10);`, `const y = parseInt(someVariable, 10);`, `const z = Number("42");`],
      invalid: [],
    });
  });

  it("valid: env access validated with isNaN using logical fallback pattern", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const count = parseInt(process.env.COUNT || "0", 10); if (Number.isNaN(count)) throw new Error();`, `const count = parseInt(process.env.COUNT ?? "0", 10); if (Number.isNaN(count)) throw new Error();`],
      invalid: [],
    });
  });

  it("valid: env access validated with isNaN using optional chaining pattern", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const count = parseInt(process.env.COUNT?.trim(), 10); if (Number.isNaN(count)) throw new Error();`],
      invalid: [],
    });
  });

  it("valid: env access validated with isNaN using ternary pattern", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const delay = parseInt(process.env.DELAY ? process.env.DELAY : "1000", 10); if (Number.isNaN(delay)) throw new Error();`],
      invalid: [],
    });
  });

  it("invalid: parseInt from process.env without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const maxRuns = parseInt(process.env.MAX_RUNS, 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "maxRuns" } }],
        },
        {
          code: `const port = parseInt(process.env.PORT, 10); startServer(port);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "port" } }],
        },
      ],
    });
  });

  it("invalid: parseFloat from process.env without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const rate = parseFloat(process.env.RATE);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "rate" } }],
        },
      ],
    });
  });

  it("invalid: Number.parseInt from process.env without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const port = Number.parseInt(process.env.PORT, 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "port" } }],
        },
      ],
    });
  });

  it("invalid: Number.parseFloat from process.env without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const delay = Number.parseFloat(process.env.DELAY);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "delay" } }],
        },
      ],
    });
  });

  it("invalid: Number() from process.env without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const runId = Number(process.env.RUN_ID);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "runId" } }],
        },
      ],
    });
  });

  it("invalid: logical fallback env access without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const maxFileSize = parseInt(process.env.MAX_FILE_SIZE || "1000", 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "maxFileSize" } }],
        },
        {
          code: `const timeout = parseInt(process.env.TIMEOUT ?? "5000", 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "timeout" } }],
        },
      ],
    });
  });

  it("invalid: optional chaining env access without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const port = parseInt(process.env.PORT?.trim(), 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "port" } }],
        },
      ],
    });
  });

  it("invalid: ternary-wrapped env access without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const configuredDelay = parseInt(process.env.DELAY ? process.env.DELAY : "1000", 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "configuredDelay" } }],
        },
      ],
    });
  });

  it("invalid: multiple unvalidated env parse declarations are each reported", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `
const maxFileSize = parseInt(process.env.MAX_FILE_SIZE, 10);
const maxFileCount = parseInt(process.env.MAX_FILE_COUNT, 10);
          `.trim(),
          errors: [
            { messageId: "requireNaNCheck", data: { name: "maxFileSize" } },
            { messageId: "requireNaNCheck", data: { name: "maxFileCount" } },
          ],
        },
      ],
    });
  });

  it("valid: ESM import style — validated with Number.isNaN", () => {
    esmRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [`const count = parseInt(process.env.COUNT, 10); if (Number.isNaN(count)) throw new Error();`],
      invalid: [],
    });
  });

  it("invalid: ESM import style — missing NaN check", () => {
    esmRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [],
      invalid: [
        {
          code: `const count = parseInt(process.env.COUNT, 10);`,
          errors: [{ messageId: "requireNaNCheck", data: { name: "count" } }],
        },
      ],
    });
  });
  it("valid: Number.isFinite guard (build_checkout_manifest.cjs pattern)", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [
        `const count = Number.parseInt(process.env.GH_AW_CHECKOUT_MANIFEST_COUNT || "0", 10); if (!Number.isFinite(count) || count < 0) { throw new Error("invalid"); }`,
        `const runId = parseInt(process.env.GITHUB_RUN_ID || "0", 10); if (Number.isFinite(runId)) { use(runId); }`,
        `const configuredDelay = Number.parseInt(process.env.DELAY_MS || "", 10); const delay = Number.isFinite(configuredDelay) && configuredDelay >= 0 ? configuredDelay : 100;`,
        `const count = parseInt(process.env.COUNT, 10); if (isFinite(count)) { use(count); }`,
      ],
      invalid: [],
    });
  });

  it("valid: truthiness guard on parsed value", () => {
    cjsRuleTester.run("require-nan-check-after-env-numeric-parse", requireNanCheckAfterEnvNumericParseRule, {
      valid: [
        `const runId = Number(process.env.GITHUB_RUN_ID || 0); if (!runId) { return; } use(runId);`,
        `const port = parseInt(process.env.PORT, 10); if (port) { listen(port); }`,
        `const port = parseInt(process.env.PORT, 10); const value = port ? port : 8080;`,
      ],
      invalid: [],
    });
  });
});
