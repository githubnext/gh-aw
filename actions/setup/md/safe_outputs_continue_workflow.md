**Continuing Long-Running Work Across Runs (Experimental)**

This workflow can hand unfinished work to a fresh run with the `continue_workflow` tool. All runs of one task share a single work branch and finish with one pull request.

1. If `/tmp/gh-aw/continue-workflow/context.json` exists, this run resumes earlier work: the work branch is already checked out. Read `plan.md` at the repository root and continue from it. The context file reports the branch, the run number (`hop`), and `remaining_hops`.
2. Keep `plan.md` at the repository root up to date: what is done, what remains, and any decisions the next run needs.
3. If the task cannot be finished in this run, commit your progress, including the updated `plan.md`, on the work branch and call `continue_workflow` with a `reason`. Do not call `create_pull_request` in the same run.
4. When the task is complete, delete `plan.md` (or finalize it), commit on the work branch, and call `create_pull_request` with that branch. This creates the single pull request for the whole task. Do not call `continue_workflow` in the same run.
5. If `remaining_hops` is 0, this is the last allowed run: finish the work and call `create_pull_request`.
- Every run must add at least one commit before calling either tool.
- Do not rewrite or force-push history on the work branch; only add commits on top of it.
