---
"gh-aw": patch
---

Fixed `push_signed_commits.cjs` to refuse unsigned `git push` fallback when the commit range contains a structurally unsignable commit (merge commit, symlink mode 120000, submodule mode 160000, or executable bit mode 100755).

Previously, the `catch` block unconditionally fell back to a plain `git push` for all errors, including the pre-flight refusals for commit shapes that `createCommitOnBranch` cannot represent. On repositories with the "Commits must have verified signatures" branch protection rule, this resulted in a rejected push and an inconsistent state where the action had already decided to attempt an unsigned push against the user's explicit policy.

The fix introduces a `PushSignedCommitsUnsupportedShape` sentinel error class. Pre-flight checks now throw this class instead of a plain `Error`, and the catch block re-throws with a clear error message when it encounters one, skipping the unsigned `git push` entirely. Transient GraphQL failures (e.g. on GHES) continue to fall back to `git push` as before.
