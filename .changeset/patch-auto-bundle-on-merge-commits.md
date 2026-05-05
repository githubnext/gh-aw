---
"gh-aw": patch
---

`push_to_pull_request_branch`: when `patch-format` is not explicitly configured and the incremental range (`origin/<branch>..<branch>`) contains a merge commit, automatically use `bundle` transport instead of the default `am` transport. `git am` cannot apply merge commits, so without this fallback long-running PR branches that periodically merge their base branch locally would fail with add/add conflicts on every push attempt. Set `patch-format: am` explicitly to opt out of the auto-fallback. The bundle transport now also records the incremental net `diff_size`, so `max-patch-size` is validated against the actual change size rather than the (often much larger) bundle artifact size.
