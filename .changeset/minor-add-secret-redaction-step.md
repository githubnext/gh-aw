---
"gh-aw": minor
---

Add secret redaction step before artifact upload in agentic workflows

This adds a JavaScript step that redacts secrets before uploading artifacts. The implementation collects secrets from workflow files, uses exact string matching for precise redaction, and displays the first 3 characters of redacted values followed by asterisks for better debugging while maintaining security. The redaction step processes files in /tmp and runs automatically before artifact uploads.
