import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireHttpResponseErrorListenerRule } from "./require-http-response-error-listener";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-http-response-error-listener", () => {
  it("valid: response callbacks that attach an 'error' listener pass", () => {
    cjsRuleTester.run("require-http-response-error-listener", requireHttpResponseErrorListenerRule, {
      valid: [
        `const http = require("http"); http.request(options, res => { res.on("data", () => {}); res.on("error", reject); });`,
        `const https = require("https"); https.get(url, res => { res.once("error", err => { reject(err); }); res.on("end", () => {}); });`,
        `const http = require("node:http"); http.request(options, function (res) { res.on("error", reject); });`,
        `const https = require("node:https"); https.request(options, res => { res.on("end", () => {}); if (res.statusCode !== 200) { res.resume(); } res.on("error", reject); });`,
      ],
      invalid: [],
    });
  });

  it("invalid: response callbacks without an 'error' listener are reported", () => {
    cjsRuleTester.run("require-http-response-error-listener", requireHttpResponseErrorListenerRule, {
      valid: [],
      invalid: [
        {
          code: `const http = require("http"); const req = http.request(options, res => { let data = ""; res.on("data", chunk => { data += chunk; }); res.on("end", () => { resolve(data); }); }); req.on("error", reject);`,
          errors: [{ messageId: "missingResponseErrorListener" }],
        },
        {
          code: `const https = require("https"); https.get(url, res => { res.on("end", () => {}); });`,
          errors: [{ messageId: "missingResponseErrorListener" }],
        },
        {
          code: `const http = require("node:http"); http.request(options, function (res) { res.resume(); });`,
          errors: [{ messageId: "missingResponseErrorListener" }],
        },
      ],
    });
  });

  it("valid: identifiers not bound to Node's http/https modules are ignored", () => {
    cjsRuleTester.run("require-http-response-error-listener", requireHttpResponseErrorListenerRule, {
      valid: [
        `const http = require("./my-http-helper.cjs"); http.request(options, res => { res.on("end", () => {}); });`,
        `const https = { request(options, cb) {} }; https.request(options, res => { res.on("data", () => {}); });`,
        `http.request(options, res => { res.on("end", () => {}); });`,
      ],
      invalid: [],
    });
  });

  it("valid: request calls without a response callback are ignored", () => {
    cjsRuleTester.run("require-http-response-error-listener", requireHttpResponseErrorListenerRule, {
      valid: [`const http = require("http"); const req = http.request(options); req.on("error", reject);`, `const https = require("https"); https.get(url);`],
      invalid: [],
    });
  });

  it("valid: nested error listener registrations count", () => {
    cjsRuleTester.run("require-http-response-error-listener", requireHttpResponseErrorListenerRule, {
      valid: [`const http = require("http"); http.request(options, res => { setImmediate(() => { res.on("error", reject); }); res.on("end", () => {}); });`],
      invalid: [],
    });
  });
});
