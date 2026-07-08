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

Disallow template literals with interpolations or string concatenation expressions as the route argument of Octokit `github` / `octokit` / `githubClient` / `octokitClient` `.request()` calls.

Using an interpolated route bypasses Octokit's typed route dispatch, can silently produce malformed paths when values contain special characters, and prevents static analysis of the route string.

**Flagged forms:**
- `` github.request(`GET /repos/${owner}/${repo}`, ...) `` — template literal with interpolations.
- `github.request("GET /repos/" + owner + "/" + repo, ...)` — string concatenation.

**Safe alternative:**
```js
github.request("GET /repos/{owner}/{repo}", { owner, repo });
```

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
