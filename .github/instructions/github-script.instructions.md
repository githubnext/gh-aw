---
description: GitHub Action Script Source Code
applyTo: "pkg/workflow/js/*.cjs"
---

This JavaScript file will be run using the GitHub Action `actions/github-script@v7` which provides the `@actions/core`, `@actions/github` packages for logging errors and setting action status.

- do not add import or require for `@actions/core`
- reference: 
  - https://github.com/actions/toolkit/blob/main/packages/core/README.md
  - https://github.com/actions/toolkit/blob/main/packages/github/README.md

## Best practices

- use `core.info`, `core.warning`, `core.error` for logging, not `console.log` or `console.error`
- use `core.debug` for verbose debug logging, which can be enabled in action settings
- use `core.setOutput` to set action outputs
- use `core.exportVariable` to set environment variables for subsequent steps
- use `core.getInput` to get action inputs, with `required: true` for mandatory inputs
- use `core.setFailed` to mark the action as failed with an error message

## Common errors

- catch handler: check if error is an instance of Error before accessing message property

```js
catch (error) {
  core.setFailed(error instanceof Error ? error : String(error));
}
```

- `core.setFailed` also calls `core.error`, so do not call both

## Typechecking

Run `make js` to run the typescript compiler.

Run `make fmt:cjs` after editing to format the file.