import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noCaughtErrorInterpolationRule } from "./no-caught-error-interpolation";

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

describe("no-caught-error-interpolation", () => {
  it("valid: non-caught variable interpolation is not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [`const err = new Error("test"); const msg = \`Error: \${err}\`;`, `const name = "world"; const s = \`Hello \${name}\`;`, `fetch(url).then(r => \`status: \${r}\`);`],
      invalid: [],
    });
  });

  it("valid: getErrorMessage(err) interpolation is not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [
        `try { f(); } catch (err) { console.log(\`failed: \${getErrorMessage(err)}\`); }`,
        `try { f(); } catch (err) { core.setFailed(\`Error: \${getErrorMessage(err)}\`); }`,
        `p.catch(err => { log(\`failed: \${getErrorMessage(err)}\`); });`,
      ],
      invalid: [],
    });
  });

  it("valid: String(err) interpolation is not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [`try { f(); } catch (err) { console.log(\`failed: \${String(err)}\`); }`, `p.catch(err => \`error: \${String(err)}\`);`],
      invalid: [],
    });
  });

  it("valid: err.message interpolation is not flagged (covered by no-unsafe-catch-error-property)", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [`try { f(); } catch (err) { console.log(\`failed: \${err.message}\`); }`, `try { f(); } catch (err) { console.log(\`stack: \${err.stack}\`); }`],
      invalid: [],
    });
  });

  it("valid: err.toString() interpolation is not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [`try { f(); } catch (err) { console.log(\`failed: \${err.toString()}\`); }`],
      invalid: [],
    });
  });

  it("valid: tagged template expression is not flagged (tag receives raw values)", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [
        // Tagged templates pass values to the tag as-is; no unsafe coercion occurs
        `try { f(); } catch (err) { tag\`msg: \${err}\`; }`,
        `p.catch(err => format\`error: \${err}\`);`,
      ],
      invalid: [],
    });
  });

  it("valid: .catch() with zero params is not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [`fetch(url).catch(() => { log("caught"); });`],
      invalid: [],
    });
  });

  it("valid: destructured catch param elements are not flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [
        // `message` is already a string extracted from the error object
        `try { f(); } catch ({ message }) { log(\`failed: \${message}\`); }`,
        `try { f(); } catch ({ message, code }) { log(\`\${code}: \${message}\`); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare catch variable in template literal is flagged (try/catch)", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(\`failed: \${err}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.log(\`failed: \${String(err)}\`); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try { f(); } catch (error) { core.setFailed(\`Action failed: \${error}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "error" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "error" },
                  output: `try { f(); } catch (error) { core.setFailed(\`Action failed: \${String(error)}\`); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: bare catch variable flagged with getErrorMessage suggestion when in scope", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { console.log(\`failed: \${err}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { console.log(\`failed: \${getErrorMessage(err)}\`); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: bare .catch() rejection handler variable is flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `fetch(url).catch(err => { log(\`network error: \${err}\`); });`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `fetch(url).catch(err => { log(\`network error: \${String(err)}\`); });`,
                },
              ],
            },
          ],
        },
        {
          code: `p.then(null, err => { log(\`rejected: \${err}\`); });`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.then(null, err => { log(\`rejected: \${String(err)}\`); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: multiple bare error interpolations in same template are all flagged", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { log(\`got \${err} at step, original: \${err}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { log(\`got \${String(err)} at step, original: \${err}\`); }`,
                },
              ],
            },
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { log(\`got \${err} at step, original: \${String(err)}\`); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: closure inside catch body still flags outer catch variable", () => {
    cjsRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { setTimeout(function() { log(\`msg: \${err}\`); }, 0); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { setTimeout(function() { log(\`msg: \${String(err)}\`); }, 0); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try { f(); } catch (err) { [1].forEach(x => { log(\`\${err}\`); }); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { [1].forEach(x => { log(\`\${String(err)}\`); }); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: ESM import of getErrorMessage triggers getErrorMessage suggestion", () => {
    esmRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `import { getErrorMessage } from "./error_helpers.js"; try { f(); } catch (err) { log(\`failed: \${err}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `import { getErrorMessage } from "./error_helpers.js"; try { f(); } catch (err) { log(\`failed: \${getErrorMessage(err)}\`); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: ESM import context is also flagged", () => {
    esmRuleTester.run("no-caught-error-interpolation", noCaughtErrorInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (e) { throw new Error(\`wrapper: \${e}\`); }`,
          errors: [
            {
              messageId: "bareErrorInterpolation",
              data: { errorVar: "e" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "e" },
                  output: `try { f(); } catch (e) { throw new Error(\`wrapper: \${String(e)}\`); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
