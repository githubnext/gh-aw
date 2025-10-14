---
"gh-aw": minor
---

Clear terminal in watch mode before recompiling

Adds automatic terminal clearing when files are modified in `--watch` mode, improving readability by removing cluttered output from previous compilations. The new `ClearScreen()` function uses ANSI escape sequences and only clears when stdout is a TTY, ensuring compatibility with pipes, redirects, and CI/CD environments.
