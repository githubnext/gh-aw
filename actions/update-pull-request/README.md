# Update Pull Request Output

Output for updating an existing pull request's title and/or body. Supports replace, append, or prepend operations for body updates. Note: The JavaScript validation ensures at least one of title or body is provided.

## Overview

This action is generated from `pkg/workflow/js/update_pull_request.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/update_pull_request
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Dependencies

This action depends on the following JavaScript modules:

- `update_pr_description_helpers.cjs`
- `update_runner.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `update_pull_request`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/update_pull_request
```

## License

MIT
