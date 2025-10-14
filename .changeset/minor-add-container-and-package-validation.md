---
"gh-aw": minor
---

Add container image and runtime package validation to --validate flag

Enhances the `--validate` flag to perform additional validation checks beyond GitHub Actions schema validation:

- **Container images**: Validates Docker container images used in MCP configurations are accessible
- **npm packages**: Validates packages referenced with `npx` exist on the npm registry
- **Python packages**: Validates packages referenced with `pip`, `pip3`, `uv`, or `uvx` exist on PyPI

The validator provides early detection of non-existent Docker images, typos in package names, and missing dependencies during compilation, giving immediate feedback to workflow authors before runtime failures occur.
