# Noop

Main function to handle noop safe output

## Overview

This action is generated from `pkg/workflow/js/noop.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/noop
```

## Outputs

### `noop_message`

**Description**: Output parameter: noop_message

## Dependencies

This action depends on the following JavaScript modules:

- `load_agent_output.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `noop`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/noop
```

## License

MIT
