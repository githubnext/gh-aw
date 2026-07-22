# ESLint Factory

This project hosts custom ESLint linters for `/actions/setup/js`.

## Goals

- Mine recurring JavaScript/TypeScript defects in `actions/setup/js`.
- Implement custom ESLint rules in TypeScript.
- Compile rules to `dist/` and run them against `actions/setup/js` scripts.

## Commands

- `npm run build` — compile rule sources.
- `npm run lint:setup-js` — build and lint all `../actions/setup/js/**/*.cjs` files.
- `npm run lint:setup-js:changed` — build and lint `../actions/setup/js/*.cjs` files.

## Rules

| Rule | Description |
|---|---|
| [`no-core-exportvariable-non-string`](#no-core-exportvariable-non-string) | Require explicit string values for `core.exportVariable` calls |
| [`no-core-setoutput-non-string`](#no-core-setoutput-non-string) | Require explicit string values for `core.setOutput` calls |
| [`no-github-request-interpolated-route`](#no-github-request-interpolated-route) | Disallow interpolated route arguments in Octokit `.request()` calls |
| [`no-json-stringify-error`](#no-json-stringify-error) | Disallow `JSON.stringify()` on caught error variables |
| [`no-throw-plain-object`](#no-throw-plain-object) | Disallow throwing plain object literals |
| [`no-unsafe-catch-error-property`](#no-unsafe-catch-error-property) | Disallow unsafe property access on `catch` error bindings |
| [`no-unsafe-promise-catch-error-property`](#no-unsafe-promise-catch-error-property) | Disallow unsafe property access in promise rejection handlers |
| [`prefer-get-error-message`](#prefer-get-error-message) | Prefer `getErrorMessage(err)` over the inline ternary pattern |
| [`prefer-core-logging`](#prefer-core-logging) | Prefer `@actions/core` logging over `console.log` / `console.info` / `console.debug` |
| [`prefer-number-isnan`](#prefer-number-isnan) | Prefer `Number.isNaN()` over global `isNaN()` |
| [`require-async-entrypoint-catch`](#require-async-entrypoint-catch) | Require `.catch(...)` on bare async entrypoint calls |
| [`require-await-core-summary-write`](#require-await-core-summary-write) | Require `await` on `core.summary.write()` calls |
| [`require-error-cause-in-rethrow`](#require-error-cause-in-rethrow) | Require `{ cause: err }` when rethrowing inside a `catch` block |
| [`require-fs-io-try-catch`](#require-fs-io-try-catch) | Require try/catch around `fs.statSync`, `readdirSync`, `copyFileSync`, `unlinkSync`, and `renameSync` |
| [`require-fs-sync-try-catch`](#require-fs-sync-try-catch) | Require try/catch around `fs.readFileSync`, `writeFileSync`, and `appendFileSync` |
| [`require-json-parse-try-catch`](#require-json-parse-try-catch) | Require try/catch around `JSON.parse(...)` calls |
| [`require-mkdirsync-try-catch`](#require-mkdirsync-try-catch) | Require try/catch around `fs.mkdirSync` calls |
| [`require-new-url-try-catch`](#require-new-url-try-catch) | Require try/catch around `new URL(variable)` calls |
| [`require-parseInt-radix`](#require-parseInt-radix) | Require an explicit radix argument to `parseInt()` |
| [`require-return-after-core-setfailed`](#require-return-after-core-setfailed) | Require a control-transfer statement after `core.setFailed()` |
| [`require-execsync-try-catch`](#require-execsync-try-catch) | Require try/catch around `execSync(...)` calls from `child_process` |
| [`require-execfilesync-try-catch`](#require-execfilesync-try-catch) | Require try/catch around `execFileSync(...)` calls from `child_process` |
| [`require-spawnsync-error-check`](#require-spawnsync-error-check) | Require checking `result.error` after `spawnSync` calls |

### `no-github-request-interpolated-route`

Disallow template literals with interpolations or string concatenation expressions as the route argument of Octokit `.request()` calls.

Using an interpolated route bypasses Octokit's typed route dispatch, can silently produce malformed paths when values contain special characters, and prevents static analysis of the route string.

**Detected Octokit clients:**
- Well-known names: `github`, `octokit`, `githubClient`, `octokitClient`.
- `context.github` — the GitHub context object's client property.
- Identifiers initialized by calling `getOctokit(...)` directly or via known module objects (`github.getOctokit(...)`, `actions.getOctokit(...)`). (Known module object names currently: `github`, `actions`.)
- Simple `const` aliases of any of the above:
  `const gh = github`, `const client = getOctokit(token)`, `const myClient = context.github`.

**Flagged forms:**
- `` github.request(`GET /repos/${owner}/${repo}`, ...) `` — template literal with interpolations.
- `github.request("GET /repos/" + owner + "/" + repo, ...)` — string concatenation.
- `` github.request(`POST ${endpoint}`, ...) `` — opaque whole-route helper; thread a typed route from the caller instead of interpolating the entire path.
- `` context.github.request(`GET /repos/${owner}/${repo}`, ...) `` — `context.github` client.
- `` const gh = github; gh.request(`GET /repos/${owner}/${repo}`, ...) `` — aliased client.
- `` const client = getOctokit(token); client.request(`GET /repos/${owner}/${repo}`, ...) `` — `getOctokit` result alias.

**Out of scope:**
- `this.github.request(...)` — `this`-based member expressions are not resolved.
- `github.request(route, ...)` — variable indirection for the route argument is not resolved.
- `github.request("GET /repos/".concat(owner), ...)` — `.concat()`-built routes are not inspected.
- `github.request("GET /repos" + "/{owner}/{repo}", ...)` — compile-time constant concatenations are accepted.

**Safe alternative:**
```js
github.request("GET /repos/{owner}/{repo}", { owner, repo });
```

For helpers that receive the entire route as a parameter, there is no mechanical `{owner}` / `{repo}` rewrite. Pass a typed route string from the caller instead of interpolating `POST ${endpoint}` or `"POST " + endpoint` at the helper call site.

### `no-core-exportvariable-non-string`

Require `core.exportVariable(name, value)` calls to pass an explicit string value for the targeted low-false-positive cases: numeric literals, boolean literals, `null`, `undefined`, and `.length` member accesses.

Why: GitHub Actions environment variables exported by `core.exportVariable` are always strings. Relying on implicit coercion can silently emit `"null"`, `"undefined"`, `"true"`, or other unintended values into downstream expressions that read the variable.

**Detected forms:**
- `core.exportVariable("MY_VAR", 42)` — numeric literal value.
- `core.exportVariable("MY_FLAG", true)` / `...false` — boolean literal value.
- `core.exportVariable("MY_VAR", null)` / `...undefined` — null or undefined value.
- `core.exportVariable("MY_COUNT", items.length)` — `.length` member access.
- `core["exportVariable"]("MY_VAR", 42)` — computed string-literal property access.
- `coreObj.exportVariable("MY_VAR", 42)` — the `coreObj` alias for `@actions/core`.

**Out of scope:**
- Variable references (e.g. `core.exportVariable("MY_VAR", someVariable)`) — the rule does not resolve variable types.
- Methods other than `exportVariable` — use `no-core-setoutput-non-string` for `setOutput`.
- Objects whose name is not in the known `@actions/core` alias list (`core`, `coreObj`).

**Safe alternatives:**
- `core.exportVariable("MY_COUNT", String(items.length))` — explicit coercion.
- `core.exportVariable("MY_VAR", "")` — empty-string semantics when `null` / `undefined` is intended to mean "not set".

### `no-json-stringify-error`

Disallow `JSON.stringify()` on caught error variables. `Error` properties (`message`, `stack`, etc.) are non-enumerable, so `JSON.stringify(err)` silently produces `{}`.

**Detected scopes:**
- `try { } catch (err) { }` — catch-clause bindings.
- `p.catch(err => ...)` — inline arrow or function callbacks passed as the first argument to `.catch()`.
- `p.then(onFulfilled, err => ...)` — inline rejection handlers passed as the **second** argument to `.then()`, which are semantically equivalent to `.catch()`.

**Out of scope:** named-reference handlers such as `p.catch(handler)` or `p.then(ok, handler)` — the rule does not follow references across files or scopes.

Flagged forms:
- `JSON.stringify(err)` where `err` is a catch-clause or inline rejection-handler parameter.
- `JSON.stringify(err, null, 2)` (with replacer/space arguments).

Safe alternatives:
- `getErrorMessage(err)` from `error_helpers.cjs` (auto-suggested fix).
- `JSON.stringify({ message: err.message, stack: err.stack })` — explicitly serializing safe string properties.

### `prefer-number-isnan`

Prefer `Number.isNaN()` over global `isNaN()` to avoid silent coercion of non-numeric inputs.

Global `isNaN()` coerces its argument before testing, so `isNaN("123")` returns `false` because `"123"` coerces to the number `123` — masking that the input was a string. `Number.isNaN()` is strict and does not coerce, making numeric validation reliable when handling raw inputs such as environment variables or API strings.

Flagged forms:
- `isNaN(x)`
- `globalThis.isNaN(x)` / `globalThis["isNaN"](x)`
- `window.isNaN(x)` / `window["isNaN"](x)`
- `global.isNaN(x)` / `global["isNaN"](x)`

Locally shadowed bindings (e.g. `const isNaN = Number.isNaN`) are intentionally excluded.

### `no-throw-plain-object`

Disallow throwing plain object literals (`throw { ... }`). Plain objects lack a `.stack` trace and a meaningful `.message` string, making errors hard to debug and incompatible with catch-clause error utilities such as `getErrorMessage`.

**Detected forms:**
- `throw { message: "not found" }` — object literal with a `message` property.
- `throw { code: 500, message: "internal" }` — object literal with extra fields.
- `throw {}` — empty object literal.
- `throw { ...base, code: 1 }` — spread elements or computed keys (no autofix suggestion, only an error).

**Out of scope:**
- `throw err` — identifier references are not checked.
- `throw new Error(...)` — `Error` constructor calls are always accepted.
- `throw Object.assign(new Error(...), { ... })` — already in the recommended form.

**JSON-RPC exemption:** Objects that match the JSON-RPC error shape `{ code: <negative integer literal>, message: <any>, data?: <any> }` are intentionally exempt. These are deliberately thrown at protocol boundaries where the receiver reads `code`, `message`, and `data` directly rather than a stack trace. Only keys from `{ code, message, data }` are allowed; extra keys, a positive `code`, a fractional `code`, or a missing `message` disqualify the exemption.

**Safe alternatives:**
- `throw new Error(message)` — minimal form.
- `throw Object.assign(new Error(message), { code, ... })` — attaches extra context while preserving the stack trace.

The rule provides an autofix suggestion for plain-key objects: it extracts the `message` property as the `Error` argument and collects remaining properties into `Object.assign(...)`.

### `no-core-setoutput-non-string`

Require `core.setOutput(name, value)` calls to pass an explicit string value for the targeted low-false-positive cases: numeric literals, boolean literals, `null`, `undefined`, and `.length` member accesses.

Why: GitHub Actions step outputs are strings. Relying on implicit coercion can silently emit `"null"`, `"undefined"`, `"true"`, or other unintended values into downstream expressions.

Typical fixes:
- `core.setOutput("count", String(count))`
- `core.setOutput("optional", "")` when empty-string semantics are intended for `null` / `undefined`

### `no-unsafe-catch-error-property`

Disallow direct access to `.message`, `.stack`, `.code`, `.status`, `.cause`, or `.name` on a `catch (err)` binding unless the code first proves the thrown value is safe to inspect.

Accepted guards:
- `getErrorMessage(err)`
- `err instanceof Error`
- `typeof err === "object" && err !== null`

Why: JavaScript can throw non-`Error` values, so `err.message` is not always safe.

### `no-unsafe-promise-catch-error-property`

Disallow the same unsafe error-property accesses inside inline promise rejection handlers such as `.catch(err => ...)`.

This rule mirrors `no-unsafe-catch-error-property`, but for promise rejection values rather than `catch` clauses. Truthiness checks such as `err && err.message` are recognized for the accessed property.

### `prefer-get-error-message`

Prefer `getErrorMessage(err)` over the repeated pattern `err instanceof Error ? err.message : String(err)`.

Why: `getErrorMessage(err)` centralizes safe error extraction and also sanitizes HTML error-page responses in the gh-aw runtime helpers.

### `require-async-entrypoint-catch`

Require bare calls to module-scope async entrypoints such as `main()` to be chained with `.catch(...)` when they are invoked outside an async context.

Flagged form:
- `main();`

Safe alternatives:
- `main().catch(err => { ... });`
- `await main();` when already inside an async function

### `require-await-core-summary-write`

Require `core.summary.write()` (including known aliases and fluent `core.summary.*().write()` chains) to be awaited when used as a bare expression.

Why: `core.summary.write()` returns a promise. Dropping it can truncate or lose the step summary if the process exits first.

Intentional exception:
- `void core.summary.write()` is treated as an explicit deliberate discard marker.

### `require-error-cause-in-rethrow`

Require rethrown `new Error(...)` values inside a `catch` block to preserve the original failure with `{ cause: err }` when the new message already references the caught error or a direct alias of it.

Flagged form:
- `throw new Error(\`failed: ${getErrorMessage(err)}\`);`

Safe alternative:
- `throw new Error(\`failed: ${getErrorMessage(err)}\`, { cause: err });`

### `require-fs-io-try-catch`

Require `fs.statSync`, `fs.readdirSync`, `fs.copyFileSync`, `fs.unlinkSync`, and `fs.renameSync` calls to be wrapped in `try/catch`.

Why: these synchronous filesystem methods throw on missing files, permission errors (`EACCES`), busy resources (`EBUSY`), and other I/O failures. An unhandled throw crashes the action without surfacing a useful diagnostic message.

**Detected forms:**
- `fs.statSync(path)` — direct call on a known `require("fs")` result.
- `fs["readdirSync"](dir)` — computed string-literal property access.
- `const { unlinkSync } = require("fs"); unlinkSync(path)` — destructured binding from `require("fs")` or `require("node:fs")`.
- ESM namespace imports: `import * as fs from "fs"; fs.copyFileSync(src, dest)`.
- ESM named imports: `import { renameSync } from "fs"; renameSync(src, dest)`.
- Bare unbound identifiers: `statSync(path)` when `statSync` is not a locally bound variable.

**Out of scope:**
- Objects whose `require` source is not the Node `fs` / `node:fs` module.
- `try { ... } finally { ... }` without a `catch` clause is still flagged.

**Safe alternative:**
```js
try {
  fs.statSync(filePath);
} catch (err) {
  throw new Error("fs.statSync failed: " + (err instanceof Error ? err.message : String(err)), { cause: err });
}
```

### `require-fs-sync-try-catch`

Require `fs.readFileSync`, `fs.writeFileSync`, and `fs.appendFileSync` calls to be wrapped in `try/catch`.

Why: these synchronous filesystem calls throw on missing files, permission errors, and disk failures, which otherwise crash the action without useful context.

Current scope:
- direct `fs.readFileSync(...)`
- known `require("fs")` aliases
- destructured aliases such as `const { readFileSync } = require("fs")`

### `require-json-parse-try-catch`

Require `JSON.parse(...)` calls to be wrapped in `try/catch`.

Why: malformed JSON should produce a controlled failure path in runtime scripts rather than an uncaught exception.

Out of scope:
- aliased or destructured `JSON.parse` references such as `const parse = JSON.parse`

### `require-parseInt-radix`

Require `parseInt()` to include an explicit radix argument.

Flagged forms:
- `parseInt(value)`
- `Number.parseInt(value)`
- `globalThis.parseInt(value)`

Why: omitting the radix allows implicit base detection, which can silently accept prefixes such as `0x`.

### `require-mkdirsync-try-catch`

Require `fs.mkdirSync` calls to be wrapped in `try/catch`.

Why: `mkdirSync` throws synchronously on permission errors, invalid paths, or unexpected filesystem state. An unhandled throw crashes the action without surfacing a useful diagnostic.

**Detected forms:**
- `fs.mkdirSync(dir)` / `fs.mkdirSync(dir, { recursive: true })` — direct calls on a known `require("fs")` result.
- `fs["mkdirSync"](dir, ...)` — computed string-literal property access.
- `const { mkdirSync } = require("fs"); mkdirSync(dir)` — destructured binding from `require("fs")` or `require("node:fs")`.
- ESM namespace imports: `import * as fs from "fs"; fs.mkdirSync(dir)`.
- ESM named imports: `import { mkdirSync } from "fs"; mkdirSync(dir)`.

**Out of scope:**
- Objects whose `require` source is not the Node `fs` / `node:fs` module (e.g. `mockFs.mkdirSync`, `storage.mkdirSync`, or `const fs = require("mock-fs"); fs.mkdirSync`).
- Other `fs` methods such as `existsSync` — use `require-fs-sync-try-catch` for `readFileSync`, `writeFileSync`, and `appendFileSync`; use `require-fs-io-try-catch` for `statSync`, `readdirSync`, `copyFileSync`, `unlinkSync`, and `renameSync`.
- `try { ... } finally { ... }` without a `catch` clause is still flagged.

**Safe alternative:**
```js
try {
  fs.mkdirSync(dir, { recursive: true });
} catch (err) {
  throw new Error("fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)), { cause: err });
}
```

### `require-new-url-try-catch`

Require `new URL(variable)` calls to be wrapped in `try/catch`.

Why: the `URL` constructor throws a `TypeError` when given an invalid or relative URL string, which crashes the action with an unhelpful uncaught exception.

**Detected forms:**
- `new URL(urlStr)` — first argument is a runtime-dynamic expression.
- `new URL(process.env.GITHUB_SERVER_URL)` — environment variable reference.
- `` new URL(`https://${host}/path`) `` — template literal with expressions.
- `new URL(host + "/x")` — string concatenation containing a variable.
- `new URL("/path", base)` — dynamic second (base) argument.
- `new URL()` — zero arguments (always throws `TypeError` at runtime).

**Out of scope (not flagged):**
- `new URL("https://github.com")` — compile-time constant string literal or static concatenation.
- `` new URL(`https://github.com/static`) `` — template literal with no expressions.
- `new URL("https://github.com" + "/owner/repo")` — concatenation of string literals only.
- `new URL(import.meta.url)` — `import.meta.url` is always a valid absolute URL in ES modules.
- `new URL("./relative/path", import.meta.url)` — `import.meta.url` as the base is safe.
- `function parse(URL, value) { return new URL(value); }` — `URL` shadowed by a local binding is not the global constructor.
- Calls already inside a `try` block with a `catch` clause.

**Known limitation — no autofix for `VariableDeclaration`:** when the flagged `new URL(...)` appears as the initializer of a variable declaration (`const u = new URL(urlStr)`), the rule reports the error but emits no autofix suggestion. Wrapping that statement in `try { ... } catch { ... }` would move subsequent uses of `u` outside the `try` block, leaving them referencing an undeclared binding. Only `ExpressionStatement` and `ReturnStatement` positions receive an autofix suggestion.

**Safe alternative:**
```js
try {
  const u = new URL(urlStr);
  // use u here
} catch (err) {
  throw new Error("URL constructor call failed: " + (err instanceof Error ? err.message : String(err)), { cause: err });
}
```

### `require-return-after-core-setfailed`

Require a `return`, `throw`, `break`, `continue`, or `process.exit()` statement immediately after `core.setFailed()` to prevent execution from continuing after a failure is declared.

Why: `core.setFailed()` only marks the action as failed at the end of the run — it does **not** stop execution. Any code that follows continues to run in a failed state, which can produce misleading output or unexpected side effects.

**Detected forms:**
- `core.setFailed(...)` — direct non-computed call on a known `@actions/core` alias (`core`, `coreObj`).
- `core["setFailed"](...)` — computed string-literal property access.
- `const c = core; c.setFailed(...)` — single-assignment alias for a core-like object.
- `const { setFailed } = core; setFailed(...)` — destructured binding from a core-like object (including renamed destructuring such as `const { setFailed: sf } = core`).

**Accepted control-transfer statements:**
- `return` / `return value`
- `throw new Error(...)`
- `process.exit(...)`
- `break` (inside a loop or switch body)
- `continue` (inside a loop body)

**Out of scope:**
- Calls on unrecognized objects: `other.setFailed("bad")` is not flagged.
- `setFailed("bad")` as a bare identifier call (not destructured from a core alias) is not flagged.

**Known limitation:** `break` and `continue` are accepted as control-transfer statements within loop or switch bodies, but they do not prevent post-loop or post-switch statements from running in a failed state. Detecting that kind of continuation is out of scope.

**Cross-block fall-through:** the rule also flags `core.setFailed(...)` that is the last statement of a nested block when the enclosing block has a subsequent statement that would execute unconditionally after the nested block exits:
```js
// Flagged — doMore() runs after setFailed even though they are in different blocks
function f() {
  if (!ok) {
    core.setFailed("msg");
  }
  doMore();
}
```

**Safe alternative:**
```js
if (!ok) {
  core.setFailed("msg");
  return;
}
doMore(); // only reached when ok is true
```

### `require-spawnsync-error-check`

Require `spawnSync` result variables to check `result.error` in addition to `result.status`.

Why: when `spawnSync` cannot spawn the child process (e.g. `ENOENT`, `ETIMEDOUT`), `result.status` is `null` and `result.error` holds the actual `Error`. Checking only `result.status` silently swallows spawn-level failures or reports a misleading "exit null" diagnostic.

**Detected forms:**
- `const result = spawnSync(cmd, args)` — unqualified `spawnSync` identifier.
- `const result = childProcess.spawnSync(...)` — `childProcess` namespace alias.
- `const result = child_process.spawnSync(...)` — `child_process` namespace alias.
- Object-destructuring: `const { status, error } = spawnSync(...)` — the destructured `error` binding must appear in a guard position.
- Renamed destructuring: `const { error: spawnError } = spawnSync(...)` — the renamed binding must appear in a guard position.
- String-literal key: `const { "error": spawnError } = spawnSync(...)`.

**Accepted guard positions for `.error`:**
- `if (result.error) throw result.error;` — direct truthiness check as an `if` test.
- `if (result.error !== undefined) throw result.error;` — binary comparison as an `if` test.
- `if (result.status !== 0 || result.error) throw result.error;` — `.error` on the right side of `||` where the full expression is the `if` test.
- `throw result.error;` / `return result.error;` — direct throw or return.
- `const e = result.error; if (e) throw e;` — single-assignment immutable alias that is later guarded.

**Out of scope (not recognized as valid guards):**
- `result.error && result.error.message` — right-hand side of `&&` is not an independent guard.
- `result.error || new Error("fallback")` — `||` right-hand operand is not a guard.
- `result.error ?? null` — nullish coalescing is not a guard.
- `core.info(String(result.error))` — logging without a conditional check does not count.
- `AssignmentExpression` forms (`result = spawnSync(...)`) and inline chains (`spawnSync(...).status`) are not analyzed.
- Passing the result object to a helper function that internally checks `.error` is not recognized.
- Mutable aliases (`let e = result.error; e = undefined; if (e) throw e`) are rejected because the original value may have been discarded before the guard.

### `prefer-core-logging`

Prefer `@actions/core` logging methods (`core.info`, `core.debug`) over `console.log`, `console.info`, and `console.debug`.

`core.*` logging methods integrate with the GitHub Actions annotation system (errors and warnings appear as file annotations in the UI) and produce structured log output. `global.core` is always available via `shim.cjs` in the Node.js context and via `github-script` in the Actions context.

**Covered methods and their replacements:**

| `console.*` method | Suggested replacement |
|---|---|
| `console.log` | `core.info` |
| `console.info` | `core.info` |
| `console.debug` | `core.debug` |

**Intentionally excluded: `console.error` and `console.warn`**

`console.error` and `console.warn` write to **`process.stderr`**, while `core.error` and `core.warning` emit GitHub Actions workflow commands to **`process.stdout`**. For processes that own stdout as a data/protocol channel — such as stdio MCP servers and transports — replacing stderr logging with stdout logging would corrupt the JSON-RPC stream. Because the stream change is not behavior-preserving, the rule never reports `console.error` or `console.warn` and offers no suggestion to replace them.


### `require-execsync-try-catch`

Require `execSync` calls sourced from `child_process` to be wrapped in `try/catch`.

Why: `execSync` throws an `Error` containing child-process result fields (for example `status`, `signal`, `stdout`, `stderr`) when the child process exits with a non-zero status code or is killed by a signal. An unhandled throw crashes the action without surfacing a useful diagnostic.

**Detected forms:**
- `const { execSync } = require("child_process"); execSync(...)` — destructured.
- `const cp = require("child_process"); cp.execSync(...)` — namespace access.
- `const run = cp.execSync; run(...)` — aliased via member expression.
- `import { execSync } from "child_process"; execSync(...)` — ESM named import.
- Both `"child_process"` and `"node:child_process"` specifiers are recognized.

**Not flagged:**
- `execSync` from any module other than `child_process` / `node:child_process`.
- Calls already inside an enclosing `try { ... } catch { ... }` block.

### `require-execfilesync-try-catch`

Require `execFileSync` calls sourced from `child_process` to be wrapped in `try/catch`.

Why: `execFileSync` has identical throw-on-failure semantics to `execSync` — it throws an `Error` containing child-process result fields (for example `status`, `signal`, `stdout`, `stderr`) when the child process exits with a non-zero status code or is killed by a signal. An unhandled throw crashes the action without surfacing a useful diagnostic.

**Detected forms:**
- `const { execFileSync } = require("child_process"); execFileSync(...)` — destructured.
- `const cp = require("child_process"); cp.execFileSync(...)` — namespace access.
- `const run = cp.execFileSync; run(...)` — aliased via member expression.
- `import { execFileSync } from "child_process"; execFileSync(...)` — ESM named import.
- Both `"child_process"` and `"node:child_process"` specifiers are recognized.

**Not flagged:**
- `execFileSync` from any module other than `child_process` / `node:child_process`.
- Calls already inside an enclosing `try { ... } catch { ... }` block.

**Out of scope:** `execFile` (the async, callback-based sibling) is intentionally excluded. The async form accepts a callback and does not throw synchronously; errors are delivered through the callback or the returned `ChildProcess` event emitter, so a synchronous try/catch provides no protection.
