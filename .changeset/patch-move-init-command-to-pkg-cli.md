---
"gh-aw": patch
---

Move init command to pkg/cli folder

Refactored the init command structure by moving `NewInitCommand()` from `cmd/gh-aw/init.go` to `pkg/cli/init_command.go` to follow the established pattern for command organization used by other commands in the repository.
