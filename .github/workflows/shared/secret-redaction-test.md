---
secret-masking:
  steps:
    - name: Redact dummy password pattern
      if: always()
      run: |
        echo "Searching for dummy password patterns in /tmp/gh-aw/"
        find /tmp/gh-aw -type f -exec sed -i 's/password123/REDACTED/g' {} + 2>/dev/null || true
        echo "Secret masking complete"
---

This shared workflow provides additional secret redaction steps to mask dummy password patterns in generated files.
