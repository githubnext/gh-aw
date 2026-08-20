import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noJsonStringifySetOrMapRule } from "./no-json-stringify-set-or-map";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-json-stringify-set-or-map", () => {
  it("valid: JSON.stringify on plain objects/arrays/other variables is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [
        `const obj = { a: 1 }; JSON.stringify(obj);`,
        `JSON.stringify([1, 2, 3]);`,
        `const data = fetchData(); JSON.stringify(data);`,
        `const arr = Array.from(new Set([1, 2])); JSON.stringify(arr);`,
        `const obj = Object.fromEntries(new Map()); JSON.stringify(obj);`,
      ],
      invalid: [],
    });
  });

  it("valid: JSON.stringify on a converted Set/Map is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [`const s = new Set([1, 2]); JSON.stringify(Array.from(s));`, `const m = new Map(); JSON.stringify(Object.fromEntries(m));`, `JSON.stringify([...new Set([1, 2])]);`],
      invalid: [],
    });
  });

  it("valid: let-declared Set/Map bindings are not tracked (could be reassigned)", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [`let s = new Set([1, 2]); s = computeSomethingElse(); JSON.stringify(s);`],
      invalid: [],
    });
  });

  it("valid: locally declared or imported Set/Map/JSON shadowing the globals is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [
        `class Set { constructor() { this.items = [1]; } } JSON.stringify(new Set());`,
        `class Map { constructor() { this.entries = []; } } const m = new Map(); JSON.stringify(m);`,
        `function Set() { this.items = [1]; } const s = new Set(); JSON.stringify(s);`,
        `const { Set } = require("immutable"); const s = new Set([1, 2]); JSON.stringify(s);`,
        `const JSON = { stringify: v => v }; const s = new Set([1, 2]); JSON.stringify(s);`,
      ],
      invalid: [],
    });
  });

  it("invalid: JSON.stringify on a const Set binding", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [],
      invalid: [
        {
          code: `const cliServers = new Set(["a", "b"]); JSON.stringify(cliServers);`,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
      ],
    });
  });

  it("invalid: JSON.stringify on a const Map binding", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [],
      invalid: [
        {
          code: `const cache = new Map([["a", 1]]); core.info(JSON.stringify(cache));`,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
      ],
    });
  });

  it("invalid: JSON.stringify on an inline new Set/Map construction", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [],
      invalid: [
        {
          code: `JSON.stringify(new Set([1, 2, 3]));`,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
        {
          code: `JSON.stringify(new Map([["k", "v"]]));`,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
      ],
    });
  });

  it("invalid: same-name bindings in different scopes only flag the Set/Map binding", () => {
    cjsRuleTester.run("no-json-stringify-set-or-map", noJsonStringifySetOrMapRule, {
      valid: [],
      invalid: [
        {
          code: `
            function withSet() {
              const seen = new Set([1, 2]);
              JSON.stringify(seen);
            }
            function withObject() {
              const seen = { other: true };
              JSON.stringify(seen);
            }
          `,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
        {
          code: `
            function withObject() {
              const seen = { other: true };
              JSON.stringify(seen);
            }
            function withSet() {
              const seen = new Set([1, 2]);
              JSON.stringify(seen);
            }
          `,
          errors: [{ messageId: "jsonStringifySetOrMap" }],
        },
      ],
    });
  });
});
