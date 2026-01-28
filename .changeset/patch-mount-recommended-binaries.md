---
"gh-aw": patch
---
Add the audited essential and common binaries (cat, curl, date, find, gh, grep, jq, yq, cp, cut, diff, head, ls, mkdir, rm, sed, sort, tail, wc, which) as read-only mounts inside the AWF agent container so workflows can rely on the expected utilities.
