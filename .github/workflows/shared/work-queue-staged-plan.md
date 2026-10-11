---
# Shared contract for dispatchers with trusted, pre-staged work-queue plans.
---

The trusted preparation step stages exactly one `submit` followed by one bounded
`dispatch_next`. Do not submit, dispatch, edit intent files, reconstruct the
plan, or call ordinary safe outputs such as `noop` or `report_incomplete`.
Read and explain the snapshot only. Trusted processing verifies the exact batch
before admission and launch; an unassigned dispatcher has no worker Claim.
