# Test Expressions File

This file is imported at runtime and contains GitHub Actions expressions.

## Safe Expressions

- **Actor**: ${{ github.actor }}
- **Repository**: ${{ github.repository }}
- **Run ID**: ${{ github.run_id }}
- **Run Number**: ${{ github.run_number }}
- **Workflow**: ${{ github.workflow }}

## Context Information

Triggered by: ${{ github.actor }}
Repository Owner: ${{ github.repository_owner }}
Server URL: ${{ github.server_url }}

All of these expressions should be rendered with actual values at runtime.
