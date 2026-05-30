# Sparse Checkout, Blobless Clones, and Credential Lifecycle

This document explains how `actions/checkout@v6` interacts with sparse-checkout and blobless clones in gh-aw compiled workflows, and why the agent job and safe_outputs job require different treatment.

## Background: Clone Modes

Git supports three orthogonal mechanisms for reducing clone size:

| Mechanism | Flag | Effect | Offline checkout? |
|-----------|------|--------|-------------------|
| Shallow clone | `--depth=N` | Downloads only last N commits (commit + tree + blob objects) | Yes — all blobs present for commits in window |
| Sparse checkout | `sparse-checkout` | Filters which paths appear in working tree; does not affect object fetching | Yes — purely a working-tree filter |
| Blobless/partial clone | `--filter=blob:none` | Omits blob objects entirely; configures a "promisor remote" for lazy fetches | **No** — any operation needing file content triggers a network fetch |

**Key insight**: `--filter=blob:none` is the only setting that prevents offline git operations. Shallow and sparse are both safe for agents.

## How `actions/checkout@v6` Introduces Blobless

When `sparse-checkout` is specified with a non-zero `fetch-depth`, `actions/checkout@v6` automatically adds `--filter=blob:none` as a bandwidth optimization. This configures the repository as a partial clone with a "promisor remote" in `.git/config`:

```ini
[remote "origin"]
    partialclonefilter = blob:none
    promisor = true
```

Once configured, git will lazily fetch missing blobs from the promisor remote on demand — but only if credentials are available.

## Agent Job: Why Blobless Breaks

Agent jobs always set `persist-credentials: false` to prevent credential exfiltration. After `actions/checkout` completes, no credentials remain on disk. The credential lifecycle is:

1. **`actions/checkout`** — uses its internal token to clone and materialize working tree blobs for the sparse cone. Sets `persist-credentials: false`, so credentials are removed after the step.
2. **Agent execution** — the agent performs arbitrary git operations (`git checkout`, `git show`, `git diff`, etc.). Any operation touching a blob not in the initial sparse cone triggers a lazy fetch from the promisor remote — which fails because no credentials exist.

This means `git checkout <fetched-branch>` prompts for a username or fails with "unable to read tree", even though the branch ref is correctly available as `origin/<branch>`.

### Fix: `filter: ''`

The compiled workflow passes `filter: ''` (empty string) in the `actions/checkout` `with:` block when sparse-checkout is configured. This prevents `actions/checkout` from adding `--filter=blob:none`, ensuring the repository is never configured as a partial clone. All blobs within the shallow window are downloaded upfront.

This is only emitted when sparse-checkout patterns are present, since that is the condition under which `actions/checkout@v6` would automatically add the blobless filter.

**Size impact**: For a large repository (e.g. github/github) with a typical sparse-checkout pattern, removing blobless increases the initial checkout from ~500 MB to ~1.3 GB (~800 MB extra, ~8-15 seconds on a GH runner). This is far preferable to agent failures and 4+ minutes of wasted retry time.

## Safe Outputs Job: Why Blobless is Safe

The safe_outputs job has a different credential lifecycle that makes blobless clones safe:

1. **`actions/checkout`** with `persist-credentials: false` — checks out the base branch. `actions/checkout` uses its own internal token to materialize working tree blobs during this step, so the initial checkout succeeds.

2. **"Configure Git credentials" step** — runs immediately after checkout and embeds the token directly in the remote URL:
   ```bash
   git remote set-url origin "https://x-access-token:${GIT_TOKEN}@${SERVER_URL}/${REPO_NAME}.git"
   ```
   From this point forward, any lazy blob fetch from the promisor remote will authenticate successfully via the embedded URL credentials.

3. **`applyBundleToBranch`** — applies the agent's git bundle, then calls `git checkout <branch>` and `git reset --hard`. These operations may trigger lazy blob fetches, but they succeed because the remote URL has embedded credentials from step 2.

4. **Push** — uses either the GraphQL signed-commits API (no git needed) or `git push` with the same token-embedded remote URL.

### The Fetch Refs Optimization

The safe_outputs job's "Fetch additional refs" step (in `compiler_safe_outputs_steps.go`) explicitly uses `--filter=blob:none` when fetching refs declared in `checkout.fetch`. This is correct because:

- These refs are fetched solely to make bundle prerequisite commits reachable locally
- They are never checked out in the safe_outputs job
- Their blob objects are genuinely unnecessary
- Credentials are available via the embedded remote URL if needed

## Summary Table

| Job | `persist-credentials` | Credentials after checkout | Blobless safe? | Action taken |
|-----|----------------------|---------------------------|----------------|--------------|
| Agent | `false` | None | **No** | Emit `filter: ''` to prevent blobless |
| Safe outputs | `false` | Re-added via `git remote set-url` | Yes | Allow blobless (bandwidth optimization) |
| Activation | `false` | None | Yes* | No action needed |

\* The activation job only checks out `.github` and `.agents` folders — it never performs arbitrary git operations on other branches.

## Related

- Issue: [#35947](https://github.com/github/gh-aw/issues/35947)
- Source (agent job): `pkg/workflow/checkout_step_generator.go`
- Source (safe_outputs job): `pkg/workflow/compiler_safe_outputs_steps.go`
- Source (bundle application): `actions/setup/js/create_pull_request.cjs` (`applyBundleToBranch`)
