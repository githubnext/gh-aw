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
