---
"gh-aw": patch
---

Convert TypeScript safe output files to CommonJS and remove TypeScript compilation

This PR simplifies the build process by converting TypeScript-based safe output files to pure CommonJS format (`.cjs`) and removing TypeScript compilation from the build pipeline. The changes include:

- Converting 4 TypeScript files (`add_labels`, `create_issue`, `create_discussion`, `collect_ndjson_output`) to CommonJS
- Removing TypeScript compilation targets from the Makefile
- Updating Go embed directives to reference `.cjs` files
- Removing post-processing scripts no longer needed for TypeScript cleanup

The conversion maintains identical runtime behavior while reducing the codebase by ~1,890 lines and simplifying build dependencies. Type checking is still performed via JSDoc comments and TypeScript's `checkJs` mode.
