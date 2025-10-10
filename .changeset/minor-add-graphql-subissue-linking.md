---
"gh-aw": minor
---

Add GraphQL sub-issue linking and optional parent parameter to create-issue safe output

This enhancement adds proper sub-issue linking using GitHub's GraphQL API when the `create-issue` safe output creates issues in the context of a parent issue. The implementation includes:

- Native GitHub sub-issue relationships using the `addSubIssue` GraphQL mutation
- Optional `parent` parameter allowing agents to explicitly specify parent issue numbers
- Graceful fallback to comment-based linking when GraphQL sub-issue linking fails
- Full backward compatibility with existing workflows

Workflows that create sub-issues (like `.github/workflows/plan.md` and `.github/workflows/dev.md`) will now properly establish parent-child relationships in GitHub's issue tracker, making it easier to track hierarchical task breakdowns and navigate between related issues.
