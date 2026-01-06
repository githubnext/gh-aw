---
# Secure Artifact Upload Template
on:
  push:
    branches: [main]
permissions:
  contents: read
---

# Build and Upload Artifacts

Build artifacts and upload them securely with proper validation.

## Task

1. Build the project
2. Validate artifacts for secrets  
3. Upload with exclusions and retention

## Build Steps

```bash
echo "Building..."
npm run build
```

## Pre-Upload Validation

Scan for secrets:

```bash
if grep -rE "(SECRET|TOKEN|PASSWORD)" dist/ 2>/dev/null; then
  echo "Error: Secrets found"
  exit 1
fi
```

## Upload

Use secure configuration:

```yaml
- uses: actions/upload-artifact@v4
  with:
    name: build-artifacts
    path: dist/**
    exclude: |
      **/.env*
      **/*.key
    retention-days: 7
```

See full template documentation for complete examples.
