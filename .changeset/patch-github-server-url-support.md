---
"gh-aw": patch
---

Use GITHUB_SERVER_URL instead of hardcoded https://github.com in safe output JavaScript files

Updated all safe output JavaScript files to use the `GITHUB_SERVER_URL` environment variable with `"https://github.com"` as a fallback. This enables GitHub Agentic Workflows to work seamlessly with GitHub Enterprise Server instances while maintaining full backward compatibility with GitHub.com and test environments.
