---
"gh-aw": patch
---

Add git patch preview in fallback issue messages

When the create_pull_request safe output handler fails to push changes or create a PR, it now includes a preview of the git patch (max 500 lines) in the fallback issue message. This improves debugging by providing immediate visibility into the changes that failed to be pushed or converted to a PR.
