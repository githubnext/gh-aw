---
"gh-aw": patch
---
Move the generated run info/workflow overview step into the activation job and upload `aw_info.json` alongside `prompt.txt` in the new `activation` artifact so prompt generation and logging can access the data earlier.
