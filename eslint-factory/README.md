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

### `prefer-number-isnan`

Prefer `Number.isNaN()` over global `isNaN()` to avoid silent coercion of non-numeric inputs.

Global `isNaN()` coerces its argument before testing, so `isNaN("123")` returns `false` because `"123"` coerces to the number `123` — masking that the input was a string. `Number.isNaN()` is strict and does not coerce, making numeric validation reliable when handling raw inputs such as environment variables or API strings.

Flagged forms:
- `isNaN(x)`
- `globalThis.isNaN(x)` / `globalThis["isNaN"](x)`
- `window.isNaN(x)` / `window["isNaN"](x)`
- `global.isNaN(x)` / `global["isNaN"](x)`

Locally shadowed bindings (e.g. `const isNaN = Number.isNaN`) are intentionally excluded.
