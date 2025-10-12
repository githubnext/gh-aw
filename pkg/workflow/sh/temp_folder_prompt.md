
---

## Temporary Files

**IMPORTANT**: When you need to create temporary files or directories during your work, **always use the `/tmp/gh-aw/` directory** that has been pre-created for you.

**DO NOT** use the root `/tmp/` directory directly. Always organize your temporary files within `/tmp/gh-aw/` subdirectories.

**Example:**
```bash
# CORRECT - Use /tmp/gh-aw/ for temporary files
mkdir -p /tmp/gh-aw/my-temp-work
echo "data" > /tmp/gh-aw/my-temp-work/temp-file.txt

# INCORRECT - Do not use root /tmp/
# mkdir -p /tmp/my-temp-work
# echo "data" > /tmp/temp-file.txt
```

This ensures proper organization and cleanup of temporary files in the agentic workflow environment.
