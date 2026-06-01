  - **Warning: No git credentials are available to the agent.** Credentials are
    intentionally removed after the checkout step for security. This means any git
    operation that needs to authenticate to the remote will fail. In private repositories, that includes:
    - `git fetch`, `git pull`, `git clone`, and `git push` (direct push, not via safe-output tools)
    - Checking out or switching to a remote branch that is not already fetched
    - Deepening a shallow clone (`git fetch --unshallow`)
    - On-demand blob fetches in partial/blobless clones (operations on files not in the initial checkout)
    Do NOT attempt to configure credentials, run `git credential fill`, or modify `.gitconfig` —
    authentication will not succeed. If you encounter credential prompts or authentication errors,
    stop immediately and report the limitation rather than spending turns trying to work around it.
