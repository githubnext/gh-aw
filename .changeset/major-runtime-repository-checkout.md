---
"gh-aw": major
---

Add runtime repository-list checkouts for scheduled workers, with the `repo` alias, mandatory scoped wildcard-capable `allowed-repos`, per-repository paths and manifests, and fail-closed visibility checks that block private or internal targets in public workflows.

Runtime checkout entries default to a maximum of 10 unique selected repositories. Set `max-repos` to a positive literal integer to change the cap. Oversized lists fail before metadata requests or checkout instead of being truncated; merged entries retain the smallest effective limit.

Runtime lists allow only public targets by default. Set the literal boolean `allow-private-repos: true` to permit allowlisted private/internal targets from a private/internal hosting repository. Public hosts always reject non-public targets, even with this opt-in enabled. The flag requires `allowed-repos`; every merged entry must explicitly opt in. Ordinary static checkout behavior is unchanged.

**Breaking change:** GitHub Actions expressions in `checkout.repository` or `checkout.repo` now require `allowed-repos`, even for one repository, except for the standalone `${{ github.repository }}` expression. Add an explicit allowlist and a literal, non-root `path` prefix for other expressions; each repository is checked out under `prefix/owner/repo`. To keep a checkout of the workflow repository at the workspace root, omit `repository` or use `${{ github.repository }}` without an allowlist. Repository selectors cannot contain wildcards; only scoped allowlist patterns support them. Rejected selections fail the job and are reported through the existing agent failure issue settings.

Runtime repository-list checkouts reject `lfs: true` at compilation and runtime, including in private workflows, because repository-controlled LFS endpoints can bypass the repository allowlist. Disable LFS for runtime lists; static checkout LFS support is unchanged.

Runtime repository-list checkouts also reject enabled submodules at compilation and runtime for all workflow host visibilities because repository-controlled submodule URLs can bypass the allowlist using the checkout token. Omit `submodules` or set it to `false`; static checkout submodule support is unchanged.

Agent checkout finalization cleans and verifies every expanded repository and submodule config using validated checkout paths, regardless of prefix depth, closing a gap in bounded credential discovery. Missing or unverifiable configs fail closed; safe-output write credentials remain available.

`allowed-repos` expressions must be standalone references to repository or organization variables. Replace caller-controlled or computed policies with literal patterns or `${{ vars.ALLOWED_REPOS }}`; repository-selection expressions remain supported. Safe-output jobs build their own checkout manifests and use the effective PR checkout credentials, including global safe-output GitHub App tokens unless overridden per entry.
