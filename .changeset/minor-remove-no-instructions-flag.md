---
"gh-aw": minor
---

Remove instruction file writing from compile command and remove --no-instructions flag

The `compile` command no longer writes instruction files to `.github/instructions/` and `.github/prompts/` - this behavior is now exclusive to the `init` command. Additionally, the `--no-instructions` flag has been completely removed from the CLI as it no longer serves any purpose. This is a breaking change for users who were relying on `compile` to update instruction files or using the `--no-instructions` flag.
