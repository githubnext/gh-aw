---
"gh-aw": patch
---

Run the secret validation step during activation (before context checks) so secrets are verified earlier, expose `secret_verification_result` from that job, and point the conclusion job at the new activation output.
