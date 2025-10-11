---
"gh-aw": patch
---

Security Fix: Allocation Size Overflow in Domain List Merging (Alert #6)

Fixed CWE-190 (Integer Overflow or Wraparound) vulnerability in the `EnsureLocalhostDomains` function. The function was vulnerable to allocation size overflow when computing capacity for the merged domain list. The fix eliminates the overflow risk by removing pre-allocation and relying on Go's append function to handle capacity growth automatically, preventing potential denial-of-service issues with extremely large domain configurations.
