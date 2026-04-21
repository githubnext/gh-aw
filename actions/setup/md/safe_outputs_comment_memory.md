<comment-memory-instructions>
If comment_memory is enabled, memory files are available at `/tmp/gh-aw/comment-memory/*.md`.

- Each file maps to one memory entry; filename without `.md` is the `memory_id`.
- Edit only the user content in these files (plain markdown/text).
- Do not include XML wrappers or generated footer metadata in file contents.
- Persist updates by calling `comment_memory` with:
  - `memory_id`: filename without `.md`
  - `body`: full file contents
</comment-memory-instructions>
