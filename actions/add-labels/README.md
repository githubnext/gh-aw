# Add Issue Label Output

Output for adding labels to an issue or pull request

## Overview

This action is generated from `pkg/workflow/js/add_labels.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/add_labels
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Outputs

### `labels_added`

**Description**: Output parameter: labels_added

## Dependencies

This action depends on the following JavaScript modules:

- `safe_output_processor.cjs`
- `safe_output_validator.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `add_labels`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/add_labels
```

## License

MIT
