---
"gh-aw": patch
---

Fix compiler issue generating invalid lock files due to heredoc delimiter

Fixed a critical bug in the workflow compiler where using single-quoted heredoc delimiters (`<< 'EOF'`) prevented GitHub Actions expressions from being evaluated in MCP server configuration files. Changed to unquoted delimiters (`<< EOF`) to allow proper expression evaluation at runtime. This fix affects all generated workflow lock files and ensures MCP configurations are correctly populated with environment variables.
