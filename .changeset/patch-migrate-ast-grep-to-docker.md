---
"gh-aw": patch
---

Update ast-grep shared workflow to use mcp/ast-grep docker image

Migrates the ast-grep shared workflow from npm-based installation to the official mcp/ast-grep Docker image from Docker Hub. This provides better isolation, consistency, and follows the Docker-based pattern used by other shared workflows.
