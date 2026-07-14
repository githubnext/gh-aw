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
