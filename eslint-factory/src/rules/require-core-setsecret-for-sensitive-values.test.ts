import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireCoreSetSecretForSensitiveValuesRule } from "./require-core-setsecret-for-sensitive-values";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-core-setsecret-for-sensitive-values", () => {
  it("uses the correct docs URL", () => {
    expect(requireCoreSetSecretForSensitiveValuesRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-core-setsecret-for-sensitive-values");
  });

  it("accepts sensitive values registered through supported core forms", () => {
    ruleTester.run("require-core-setsecret-for-sensitive-values", requireCoreSetSecretForSensitiveValuesRule, {
      valid: [
        `const token = process.env.GITHUB_TOKEN; core.setSecret(token);`,
        `const apiKey = process.env.API_KEY; core["setSecret"](apiKey.trim());`,
        `const credential = payload.client_secret; const c = core; c.setSecret(credential);`,
        `const { setSecret: mask } = core; const password = config.password; mask(password);`,
        `const { GITHUB_TOKEN: value } = process.env; coreObj.setSecret(value);`,
        `const { client_secret: value } = JSON.parse(raw); core.setSecret(value);`,
        `let accessToken; accessToken = response.access_token; core.setSecret(accessToken);`,
      ],
      invalid: [],
    });
  });

  it("reports sensitive environment, input, parsed, and derived values that are not masked", () => {
    ruleTester.run("require-core-setsecret-for-sensitive-values", requireCoreSetSecretForSensitiveValuesRule, {
      valid: [],
      invalid: [
        {
          code: `const value = process.env.GITHUB_TOKEN;`,
          errors: [{ messageId: "requireSetSecret", data: { name: "value" } }],
        },
        {
          code: `const apiKey = core.getInput("api-key");`,
          errors: [{ messageId: "requireSetSecret", data: { name: "apiKey" } }],
        },
        {
          code: `const credential = JSON.parse(raw).credential;`,
          errors: [{ messageId: "requireSetSecret", data: { name: "credential" } }],
        },
        {
          code: `const safeValue = Buffer.from(authToken).toString("base64");`,
          errors: [{ messageId: "requireSetSecret", data: { name: "safeValue" } }],
        },
        {
          code: `const { CLIENT_SECRET: value } = process.env;`,
          errors: [{ messageId: "requireSetSecret", data: { name: "value" } }],
        },
        {
          code: `const { password: value } = JSON.parse(raw);`,
          errors: [{ messageId: "requireSetSecret", data: { name: "value" } }],
        },
        {
          code: `let result; result = payload.access_token;`,
          errors: [{ messageId: "requireSetSecret", data: { name: "result" } }],
        },
        {
          code: `const token = process.env.GITHUB_TOKEN; core.setSecret(other, token);`,
          errors: [{ messageId: "requireSetSecret", data: { name: "token" } }],
        },
      ],
    });
  });

  it("ignores token accounting and boolean secret-presence checks", () => {
    ruleTester.run("require-core-setsecret-for-sensitive-values", requireCoreSetSecretForSensitiveValuesRule, {
      valid: [
        `const inputTokens = usage.input_tokens || 0;`,
        `const tokenCount = estimateTokens(text);`,
        `const tokenThreshold = Number(process.env.LONG_RUN_TOKEN_THRESHOLD);`,
        `const hasToken = !!process.env.GITHUB_TOKEN;`,
        `const usingCustomToken = process.env.GITHUB_TOKEN !== undefined;`,
        `const GITHUB_TOKEN_CONFIG_KEY = "github-token";`,
        `const octokit = github.getOctokit(token);`,
        `const validationResult = validateToken(token);`,
        `const renderTokenError = () => "missing token";`,
        `const value = response.status;`,
      ],
      invalid: [],
    });
  });

  it("does not confuse masking a shadowed binding with masking the sensitive value", () => {
    ruleTester.run("require-core-setsecret-for-sensitive-values", requireCoreSetSecretForSensitiveValuesRule, {
      valid: [],
      invalid: [
        {
          code: `const token = process.env.GITHUB_TOKEN; function mask(token) { core.setSecret(token); }`,
          errors: [{ messageId: "requireSetSecret", data: { name: "token" } }],
        },
      ],
    });
  });
});
