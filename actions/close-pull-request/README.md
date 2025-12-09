# Close Pull Request Output

Output for closing a GitHub pull request without merging, with a comment

## Overview

This action is generated from `pkg/workflow/js/close_pull_request.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/close_pull_request
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Dependencies

This action depends on the following JavaScript modules:

- `close_entity_helpers.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `close_pull_request`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/close_pull_request
```

## License

MIT
