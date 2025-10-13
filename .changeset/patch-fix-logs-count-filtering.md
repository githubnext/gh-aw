---
"gh-aw": patch
---

Fix logs command to fetch all runs when date filters are specified

The `logs` command's `--count` parameter was limiting the number of logs downloaded, not the number of matching logs returned after filtering. This caused incomplete results when using date filters like `--start-date -24h`.

Modified the algorithm to always limit downloads inline based on remaining count needed, ensuring the count parameter correctly limits the final output after applying all filters. Also increased the default count from 20 to 100 for better coverage and updated documentation to clarify the behavior.
