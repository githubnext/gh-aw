---
"gh-aw": minor
---

Extract and display premium model information and request consumption from Copilot CLI logs

Enhanced the Copilot log parser to extract and display premium request information from agent stdio logs. Users can now see which AI model was used, whether it requires a premium subscription, any cost multipliers that apply, and how many premium requests were consumed. This information is now surfaced directly in the GitHub Actions step summary, making it easily accessible without needing to download and manually parse log files.
