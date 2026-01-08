---
"gh-aw": patch
---

Handle 502 Bad Gateway errors in assign_to_agent handler by treating them as success. The cloud gateway may return 502 errors during agent assignment, but the assignment typically succeeds despite the error. The handler now logs 502 errors for troubleshooting but does not fail the workflow.
