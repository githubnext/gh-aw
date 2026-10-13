---
"gh-aw": major
---

Add runtime repository-list checkouts for scheduled workers, with the `repo` alias, mandatory scoped wildcard-capable `allowed-repos`, per-repository paths and manifests, and fail-closed visibility checks that block private or internal targets in public workflows.

**Breaking change:** GitHub Actions expressions in `checkout.repository` or `checkout.repo` now require `allowed-repos`, even for one repository. Add an explicit allowlist and a literal, non-root `path` prefix; each repository is checked out under `prefix/owner/repo`. To keep a checkout of the workflow repository at the workspace root, omit `repository` instead of using `${{ github.repository }}`. Repository selectors cannot contain wildcards; only scoped allowlist patterns support them. Rejected selections fail the job and are reported through the existing agent failure issue settings.

Runtime repository-list checkouts reject `lfs: true` at compilation and runtime, including in private workflows, because repository-controlled LFS endpoints can bypass the repository allowlist. Disable LFS for runtime lists; static checkout LFS support is unchanged.
