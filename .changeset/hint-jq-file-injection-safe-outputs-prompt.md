---
"gh-aw": patch
---
Add jq file-injection hint to the safe outputs CLI proxy instruction. Agents can use `jq -Rs` to read a local file and inject its content as the `body` field without re-embedding the entire text in the model context.
