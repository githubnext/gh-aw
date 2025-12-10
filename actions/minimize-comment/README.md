# Minimize Comment Output

Output for minimizing (hiding) a comment on a GitHub issue, pull request, or discussion

## Overview

This action is generated from `pkg/workflow/js/minimize_comment.cjs` and provides functionality for GitHub Agentic Workflows.

## Usage

```yaml
- uses: ./actions/minimize_comment
  with:
    token: 'value'  # GitHub token for API authentication
```

## Inputs

### `token`

**Description**: GitHub token for API authentication

**Required**: true

## Outputs

### `comment_id`

**Description**: Output parameter: comment_id

### `is_minimized`

**Description**: Output parameter: is_minimized

## Dependencies

This action depends on the following JavaScript modules:

- `load_agent_output.cjs`

## Development

### Building

To build this action, you need to:

1. Update the dependency mapping in `pkg/cli/actions_build_command.go` for `minimize_comment`
2. Run `make actions-build` to bundle the JavaScript dependencies
3. The bundled `index.js` will be generated and committed

### Testing

Test this action by creating a workflow:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: ./actions/minimize_comment
```

## License

MIT
