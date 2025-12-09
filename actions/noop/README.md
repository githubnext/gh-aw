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


## Known Limitations

**Note**: This generated action currently uses `require()` statements for dependencies, which means it is not fully self-contained. This action is intended as a reference implementation that demonstrates the structure and functionality of the JavaScript module.

To make this action production-ready, convert it to use the FILES embedding pattern (see `setup-safe-outputs/src/index.js` for an example) or ensure all required dependencies are available at runtime.

## License

MIT
