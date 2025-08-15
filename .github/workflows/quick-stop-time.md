---
# Run once a day at midnight UTC
on:
  schedule:
    - cron: "0 0 * * *"
  workflow_dispatch:

stop-time: +2m
---

Show the answer to the question "What is the current time?" in the output.