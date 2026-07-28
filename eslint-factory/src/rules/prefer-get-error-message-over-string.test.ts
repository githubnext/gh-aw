import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { preferGetErrorMessageOverStringRule } from "./prefer-get-error-message-over-string";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("prefer-get-error-message-over-string", () => {
  it("valid: String(err) not flagged when getErrorMessage is unavailable", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [`try { f(); } catch (err) { console.log(\`failed: \${String(err)}\`); }`, `p.catch(err => \`error: \${String(err)}\`);`],
      invalid: [],
    });
  });

  it("valid: non-caught variable String() interpolation is not flagged", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [`const { getErrorMessage } = require("./error_helpers.cjs"); const name = "world"; const s = \`Hello \${String(name)}\`;`],
      invalid: [],
    });
  });

  it("valid: getErrorMessage(err) usage is not flagged", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [`const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { console.log(\`failed: \${getErrorMessage(err)}\`); }`],
      invalid: [],
    });
  });

  it("invalid: String(err) flagged in catch block when getErrorMessage is imported", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { throw new Error(\`Failed: \${String(err)}\`, { cause: err }); }`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  output: `const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { throw new Error(\`Failed: \${getErrorMessage(err)}\`, { cause: err }); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: String(err) flagged in .catch(fn) rejection handler when getErrorMessage is imported", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require("./error_helpers.cjs"); p.catch(err => log(\`failed: \${String(err)}\`));`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  output: `const { getErrorMessage } = require("./error_helpers.cjs"); p.catch(err => log(\`failed: \${getErrorMessage(err)}\`));`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: String(err) in a tagged template is not flagged", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [`const { getErrorMessage } = require("./error_helpers.cjs"); try { f(); } catch (err) { tag\`failed: \${String(err)}\`; }`],
      invalid: [],
    });
  });

  it("valid: String(metadata) for a non-first rejection handler parameter is not flagged", () => {
    cjsRuleTester.run("prefer-get-error-message-over-string", preferGetErrorMessageOverStringRule, {
      valid: [`const { getErrorMessage } = require("./error_helpers.cjs"); p.catch((err, metadata) => log(\`info: \${String(metadata)}\`));`],
      invalid: [],
    });
  });
});
