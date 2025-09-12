---
description: GitHub Action Script Source Code
applyTo: "pkg/workflow/js/*.cjs"
---

This JavaScript file will be run using the GitHub Action `actions/github-script@v7` which provides the `@actions/core`, `@actions/github` packages for logging errors and setting action status.

- do not add import or require for `@actions/core`
- reference: 
  - https://github.com/actions/toolkit/blob/main/packages/core/README.md
  - https://github.com/actions/toolkit/blob/main/packages/github/README.md

## Common errors

- catch handler: check if error is an instance of Error before accessing message property

```js
catch (error) {
  core.setFailed(error instanceof Error ? error : String(error));
}
```

## Typechecking

Run `make js` to run the typescript compiler.