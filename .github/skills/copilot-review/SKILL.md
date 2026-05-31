---
name: copilot-review
description: Teach Copilot how to plan, address, and respond to pull request review feedback.
---

# Copilot Review Skill

Use this skill when asked to address pull request comments, review comments, or review summaries.

## Scope

Process feedback only from these sources:

- GitHub Copilot actors
- GitHub Actions actors
- Team members

Ignore comments and reviews from non-team members.

## Reviewer Eligibility

Treat feedback as in-scope only when the author is one of the following:

- `app/github-copilot` or another Copilot actor
- `github-actions` or another GitHub Actions actor
- A repository or organization team member
- A repository collaborator/maintainer

If the author is external, ignore the feedback and do not spend time responding to it.

## Required Workflow

### 1. Collect all feedback first

Before making changes, gather all pull request discussion in one pass:

- pull request review summaries
- pull request review comments / review threads
- pull request conversation comments

Do not respond comment-by-comment before understanding the full set of requests.

### 2. Filter to allowed reviewers

Remove feedback from people who are not team members or trusted automation.

Keep only comments and reviews from the allowed reviewer set above.

### 3. Bucket the feedback

Group the remaining feedback into clear buckets such as:

- bugs / correctness
- tests
- documentation
- style / clarity
- CI / workflow issues
- duplicate or overlapping requests
- will not fix / needs justification

Create a short plan that covers every bucket before editing code.

### 4. Resolve each bucket

For every bucket, decide whether to:

- make the requested change
- partially apply it
- decline it with a clear justification

Do not silently ignore in-scope feedback.

### 5. Validate before replying

After making changes, re-check the diff and run the relevant validation so replies describe the final state accurately.

### 6. Reply to every in-scope review comment

Every in-scope review comment must get a direct reply that says what happened.

Each reply should briefly state one of:

- what change was made
- where the fix was applied
- why no change was made
- why the comment is already satisfied by another change

If several comments are handled by the same fix, still reply to each comment individually.

### 7. Resolve threads after answering

If a review thread has been fully addressed and the tooling supports it:

- reply with the action taken
- resolve the thread

Do not resolve a thread without answering it first.

## Response Rules

- Answer every in-scope review comment.
- Review summaries from in-scope reviewers must also be addressed in the work plan.
- Keep replies concise, specific, and action-oriented.
- Mention file names or behavior changes when helpful.
- When declining a request, explain why it is being ignored.
- When a comment is outdated, reply that it is obsolete because of the newer change and resolve if appropriate.

## Planning Standard

Before editing, produce a compact internal checklist that maps:

- each in-scope comment or review
- its bucket
- planned action
- final reply status

Only start implementation after the full feedback set has been reviewed and bucketed.

## Completion Standard

The task is complete only when all of the following are true:

- all in-scope comments and reviews were collected
- non-team-member feedback was ignored
- each in-scope item was resolved by code changes or explicit justification
- every in-scope review comment received a reply describing the action taken
- addressed threads were resolved when possible
