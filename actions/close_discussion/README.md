# Close Discussion

Closes GitHub discussions

## Overview

This action is generated from `pkg/workflow/js/close_discussion.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/close_discussion
```

## Outputs

### `comment_url`

**Description**: Output parameter: comment_url

### `discussion_number`

**Description**: Output parameter: discussion_number

### `discussion_url`

**Description**: Output parameter: discussion_url

## Dependencies

This action depends on the following JavaScript modules:

- `generate_footer.cjs`
- `get_repository_url.cjs`
- `get_tracker_id.cjs`
- `load_agent_output.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `close_discussion`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/close_discussion
```


## Known Limitations

**Note**: This generated action currently uses `require()` statements for dependencies, which means it is not fully self-contained. This action is intended as a reference implementation that demonstrates the structure and functionality of the JavaScript module.

To make this action production-ready, convert it to use the FILES embedding pattern (see `setup-safe-outputs/src/index.js` for an example) or ensure all required dependencies are available at runtime.

## License

MIT
