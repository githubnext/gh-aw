# Safe Inputs Copy Action

This action copies safe-inputs MCP server files to the agent environment.

## Description

The safe-inputs MCP server provides read operations for GitHub Actions workflows. This action copies all necessary JavaScript files to the agent environment so they can be used by the workflow.

## Usage

```yaml
- name: Setup Safe Inputs Files
  uses: ./actions/safe-inputs-copy
  with:
    # Destination directory for safe-inputs files
    # Default: /tmp/gh-aw/safe-inputs
    destination: /tmp/gh-aw/safe-inputs
```

## Inputs

### `destination`

**Optional** Destination directory for safe-inputs files.

Default: `/tmp/gh-aw/safe-inputs`

## Outputs

### `files-copied`

The number of files copied to the destination directory.

## Example

```yaml
steps:
  - uses: actions/checkout@v4
  
  - name: Setup Safe Inputs Files
    uses: ./actions/safe-inputs-copy
    
  - name: Use Safe Inputs
    run: |
      node /tmp/gh-aw/safe-inputs/safe_inputs_mcp_server.cjs
```

## Development

This action is built from source files in `src/` using the build tooling:

```bash
make actions-build
```

The build process embeds all required JavaScript files into the bundled `index.js`.
