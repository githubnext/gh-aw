---
"gh-aw": major
---

Add runtime repository-list checkouts for scheduled workers, with the `repo` alias, mandatory scoped wildcard-capable `allowed-repos`, per-repository paths and manifests, and fail-closed visibility checks that block private or internal targets in public workflows.

**Breaking change:** GitHub Actions expressions in `checkout.repository` or `checkout.repo` now require `allowed-repos`, even for one repository, except for the standalone `${{ github.repository }}` expression. Add an explicit allowlist and a literal, non-root `path` prefix for other expressions; each repository is checked out under `prefix/owner/repo`. To keep a checkout of the workflow repository at the workspace root, omit `repository` or use `${{ github.repository }}` without an allowlist. Repository selectors cannot contain wildcards; only scoped allowlist patterns support them. Rejected selections fail the job and are reported through the existing agent failure issue settings.

Runtime repository-list checkouts reject `lfs: true` at compilation and runtime, including in private workflows, because repository-controlled LFS endpoints can bypass the repository allowlist. Disable LFS for runtime lists; static checkout LFS support is unchanged.

Runtime repository-list checkouts also reject enabled submodules at compilation and runtime for all workflow host visibilities because repository-controlled submodule URLs can bypass the allowlist using the checkout token. Omit `submodules` or set it to `false`; static checkout submodule support is unchanged.

Agent checkout finalization cleans and verifies every expanded repository and submodule config using validated checkout paths, regardless of prefix depth, closing a gap in bounded credential discovery. Missing or unverifiable configs fail closed; safe-output write credentials remain available.
