import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFsSyncTryCatchRule } from "./require-fs-sync-try-catch";

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

describe("require-fs-sync-try-catch", () => {
  it("valid: fs.readFileSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [
        `try { const x = fs.readFileSync(path, "utf8"); } catch (e) {}`,
        `try { return fs.readFileSync(path, "utf8"); } catch (e) {}`,
        `function f() { try { fs.readFileSync(path); } catch (e) {} }`,
        `try { const x = fs["readFileSync"](path, "utf8"); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: fs.writeFileSync and fs.appendFileSync inside try block pass", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [`try { fs.writeFileSync(path, data); } catch (e) {}`, `try { fs.appendFileSync(path, data); } catch (e) {}`, `try { fs["writeFileSync"](path, data); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: other fs methods not in scope are ignored", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [
        `fs.existsSync(path);`,
        `fs.mkdirSync(dir, { recursive: true });`,
        `fs.unlinkSync(path);`,
        `fs.statSync(path);`,
        `fs.readdirSync(dir);`,
        `fs.rmSync(dir, { recursive: true });`,
        // Non-fs objects with same method names are ignored
        `mockFs.readFileSync(path);`,
        `storage.writeFileSync(path, data);`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured fs bindings inside try block pass", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [
        `const { readFileSync } = require("fs"); try { readFileSync(path, "utf8"); } catch (e) {}`,
        `const { appendFileSync } = require("node:fs"); try { appendFileSync(p, data); } catch (e) {}`,
        `const { appendFileSync: append } = require("fs"); try { append(p, data); } catch (e) {}`,
        `const { appendFileSync } = require("custom-logger"); appendFileSync(filePath, data);`,
      ],
      invalid: [],
    });
  });

  it("valid: member-expression alias inside try block passes", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [`const fs = require("fs"); const append = fs.appendFileSync; try { append(p, data); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: fs inside try block (ES module) passes", () => {
    esmRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [
        `try { const x = fs.readFileSync(path, "utf8"); } catch (e) {}`,
        `import { readFileSync } from "node:fs"; try { readFileSync(path, "utf8"); } catch (e) {}`,
        `import { writeFileSync as write } from "fs"; try { write(path, data); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: try/catch/finally is treated as protective", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [`try { fs.writeFileSync(path, data); } catch (e) {} finally { cleanup(); }`, `try { fs.readFileSync(path, "utf8"); } catch (e) { handleErr(e); } finally { releaseLock(); }`],
      invalid: [],
    });
  });

  it("invalid: try/finally without catch is not protective", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `try { fs.writeFileSync(path, data); } finally { cleanup(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "writeFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { try {\n  fs.writeFileSync(path, data);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.writeFileSync call.\n  throw new Error(\n    "fs.writeFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} } finally { cleanup(); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try { fs.readFileSync(path, "utf8"); } finally { releaseLock(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { try {\n  fs.readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} } finally { releaseLock(); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: synchronous callbacks inside try block are protected", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [
        // Array map is synchronous — try block is genuinely protective
        `try { const results = paths.map(p => fs.readFileSync(p, "utf8")); } catch (e) {}`,
        `try { items.forEach(p => { fs.writeFileSync(p, data); }); } catch (e) {}`,
        // Locally-defined nextTick can be synchronous, so the surrounding try is protective
        `try { const nextTick = fn => fn(); nextTick(() => { fs.readFileSync(path, "utf8"); }); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare fs.readFileSync is flagged with correct message and suggestion", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const content = fs.readFileSync(filePath, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "filePath" },
              suggestions: [],
            },
          ],
        },
        {
          code: `fs.readFileSync(configPath, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "configPath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.readFileSync(configPath, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: bare fs.writeFileSync is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.writeFileSync(outputPath, JSON.stringify(data));`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "writeFileSync", arg: "outputPath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.writeFileSync(outputPath, JSON.stringify(data));\n} catch (err) {\n  // TODO: handle I/O failure for this fs.writeFileSync call.\n  throw new Error(\n    "fs.writeFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: bare fs.appendFileSync is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.appendFileSync(logPath, line + "\\n");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "logPath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.appendFileSync(logPath, line + "\\n");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it('invalid: computed fs["readFileSync"] and aliased fs member access are flagged when not in try block', () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const data = fs["readFileSync"](path, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [],
            },
          ],
        },
        {
          code: `const fileSystem = require("fs"); fileSystem.readFileSync(path, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fileSystem = require("fs"); try {\n  fileSystem.readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.readFileSync in deferred callback is not protected by surrounding try", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        // EventEmitter .on — callback fires asynchronously
        {
          code: `try { emitter.on("data", () => { fs.readFileSync(path, "utf8"); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { emitter.on("data", () => { try {\n  fs.readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // setTimeout — callback fires asynchronously
        {
          code: `try { setTimeout(() => { fs.writeFileSync(p, d); }, 0); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "writeFileSync", arg: "p" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { setTimeout(() => { try {\n  fs.writeFileSync(p, d);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.writeFileSync call.\n  throw new Error(\n    "fs.writeFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }, 0); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // process.nextTick — callback runs after the surrounding try has returned
        {
          code: `try { process.nextTick(() => { fs.readFileSync(path, "utf8"); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { process.nextTick(() => { try {\n  fs.readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // async function bodies are still synchronous relative to their own frame
        {
          code: `async function load() { fs.readFileSync(path, "utf8"); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function load() { try {\n  fs.readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
        // new Promise executor — Promise captures throws
        {
          code: `try { new Promise(resolve => { fs.readFileSync(path); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { new Promise(resolve => { try {\n  fs.readFileSync(path);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.readFileSync inside if-branch without surrounding try is flagged (ES module)", () => {
    esmRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `if (cond) {\n  const raw = fs.readFileSync(p, "utf8");\n}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "p" },
              suggestions: [],
            },
          ],
        },
        {
          code: `import { readFileSync } from "node:fs"; readFileSync(p, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "p" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import { readFileSync } from "node:fs"; try {\n  readFileSync(p, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { appendFileSync as append } from "fs"; append(logPath, line);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "logPath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import { appendFileSync as append } from "fs"; try {\n  append(logPath, line);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { writeFileSync } from "node:fs"; writeFileSync(outputPath, data);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "writeFileSync", arg: "outputPath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import { writeFileSync } from "node:fs"; try {\n  writeFileSync(outputPath, data);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.writeFileSync call.\n  throw new Error(\n    "fs.writeFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: unsupported positions and multi-line expressions handle suggestions safely", () => {
    esmRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `export default fs.readFileSync(config, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "config" },
              suggestions: [],
            },
          ],
        },
        {
          code: `fs.readFileSync(\n  config,\n  "utf8",\n);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "config" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.readFileSync(\n    config,\n    "utf8",\n  );\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured fs binding (shorthand) not in try is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          // Regression: was a live false negative in actions/setup/js/action_setup_otlp.cjs.
          code: `const { appendFileSync } = require("fs"); appendFileSync(filePath, \`\${key}=\${value}\\n\`);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "filePath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { appendFileSync } = require("fs"); try {\n  appendFileSync(filePath, \`\${key}=\${value}\\n\`);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured fs binding via node:fs not in try is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { readFileSync } = require("node:fs"); readFileSync(path, "utf8");`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "readFileSync", arg: "path" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { readFileSync } = require("node:fs"); try {\n  readFileSync(path, "utf8");\n} catch (err) {\n  // TODO: handle I/O failure for this fs.readFileSync call.\n  throw new Error(\n    "fs.readFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: renamed destructured fs binding not in try is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { appendFileSync: append } = require("fs"); append(filePath, data);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "filePath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { appendFileSync: append } = require("fs"); try {\n  append(filePath, data);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: member-expression alias (const alias = fs.method) not in try is flagged", () => {
    cjsRuleTester.run("require-fs-sync-try-catch", requireFsSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); const append = fs.appendFileSync; append(filePath, data);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "filePath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); const append = fs.appendFileSync; try {\n  append(filePath, data);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); const append = fs.appendFileSync; function inner() { const fs = {}; append(filePath, data); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "appendFileSync", arg: "filePath" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); const append = fs.appendFileSync; function inner() { const fs = {}; try {\n  append(filePath, data);\n} catch (err) {\n  // TODO: handle I/O failure for this fs.appendFileSync call.\n  throw new Error(\n    "fs.appendFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
