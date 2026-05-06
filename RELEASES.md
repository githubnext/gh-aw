# Releases

This file documents the internal release flow for `github/gh-aw`.

## Release flow

1. Start the release from `.github/workflows/release.md` using
   `workflow_dispatch` and select `release_type` (`patch`,
   `minor`, or `major`).
2. The workflow computes the next semver tag, creates the tag,
   builds artifacts, and creates a GitHub release as a
   **prerelease** with `latest=false`.
3. During the release workflow, hand off to
   [`github/gh-aw-actions`](https://github.com/github/gh-aw-actions/actions/runs/25454542215/agentic_workflow):
   run the required sync flow, merge the generated PR in
   `github/gh-aw-actions`, verify the tag exists there, then
   continue and finish the `gh-aw` release workflow.

## Promotion cadence

- A new release is first floated as a prerelease for a few days.
- On Monday, promote the last known good prerelease to
  **latest**.
- That promoted release becomes the current latest stable result.

## Versioning policy

The team follows semantic versioning on a best-effort basis.
