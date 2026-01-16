# Test Unsafe Expressions File

This file should be rejected because it contains unsafe expressions.

## Unsafe Expressions

- **Token**: ${{ secrets.GITHUB_TOKEN }}

This should fail at runtime.
