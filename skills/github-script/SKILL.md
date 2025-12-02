---
name: github-script
description: Best practices for writing JavaScript code for GitHub Actions using github-script
---

# GitHub Action Script Best Practices

This skill provides guidelines for writing JavaScript files that run using the GitHub Action `actions/github-script@v8`.

## Important Notes

- This action provides `@actions/core` and `@actions/github` packages globally
- Do not add import or require for `@actions/core` 
- Reference documentation:
  - https://github.com/actions/toolkit/blob/main/packages/core/README.md
  - https://github.com/actions/toolkit/blob/main/packages/github/README.md

## Best Practices

- Use `core.info`, `core.warning`, `core.error` for logging, not `console.log` or `console.error`
- Use `core.setOutput` to set action outputs
- Use `core.exportVariable` to set environment variables for subsequent steps
- Use `core.getInput` to get action inputs, with `required: true` for mandatory inputs
- Use `core.setFailed` to mark the action as failed with an error message

## Step Summary

Use `core.summary.*` function to write output the step summary file.

- Use `core.summary.addRaw()` to add raw Markdown content (GitHub Flavored Markdown supported)
- Make sure to call `core.summary.write()` to flush pending writes
- Summary function calls can be chained, e.g. `core.summary.addRaw(...).addRaw(...).write()`

## Common Errors

- Avoid `any` type as much as possible, use specific types or `unknown` instead
- Catch handler: check if error is an instance of Error before accessing message property

```js
catch (error) {
  core.setFailed(error instanceof Error ? error : String(error));
}
```

- `core.setFailed` also calls `core.error`, so do not call both

## Typechecking

Run `make js` to run the typescript compiler.

Run `make lint-cjs` to lint the files.

Run `make fmt-cjs` after editing to format the file.
