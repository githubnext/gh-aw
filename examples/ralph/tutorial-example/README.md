# Tutorial Example: User Profile Management

This directory contains a complete Ralph Loop example demonstrating an autonomous AI agent working through a feature implementation over multiple iterations.

## Overview

This example shows how a Ralph Loop workflow:
1. Works through a structured PRD (Product Requirements Document)
2. Implements features incrementally
3. Learns from failures
4. Maintains progress across iterations
5. Only commits working, tested code

## Files in This Example

### Core Files
- **`feature-prd.md`** - Human-readable PRD with user stories and acceptance criteria
- **`prd.json`** - Structured JSON version for programmatic processing
- **`progress.txt`** - Detailed log of each iteration with learnings
- **`AGENTS.md`** - Project-specific agent instructions (agent context)

### Iteration Examples
- **`iterations/`** - Snapshots showing code at different stages

## The Story

### Feature: User Profile Management
A typical web application feature: users need to view and update their profiles, including uploading avatars.

### Iteration Breakdown

**Iteration 1: View Profile (✅ Success)**
- Implemented GET /api/profile endpoint
- Added authentication and response formatting
- Tests passed, committed successfully
- Time: 30 minutes

**Iteration 2: Error Handling (✅ Success)**
- Added 404 handling for missing users
- Discovered soft-delete pattern in codebase
- Tests passed, committed successfully
- Time: 30 minutes

**Iteration 3: Update Profile (⚠️ Partial)**
- Implemented PUT /api/profile endpoint
- Email validation working
- Discovered username uniqueness race condition
- Did NOT commit (test failure)
- Time: 45 minutes

**Iteration 4: Fix Uniqueness (❌ Failed)**
- Attempted to add database UNIQUE constraint
- Migration failed due to existing duplicate data
- Learned: need data cleanup before constraint
- Did NOT commit
- Time: 30 minutes

**Iteration 5: (Next)...**
- Would add data cleanup migration
- Apply UNIQUE constraint
- Update error handling
- Complete US-002

## Key Learnings from This Example

### 1. Iterative Progress Works
The agent made real progress in iterations 1-2, completing an entire user story with tests. When it hit a complex problem (race condition), it didn't get stuck—it documented the issue in progress.txt for the next iteration.

### 2. Failures Provide Context
Iteration 4's failure revealed important information:
- Test database has duplicate usernames
- Need data migration, not just schema migration
- The agent learned the database state matters

This failure made the *next* iteration smarter.

### 3. No Broken Commits
Despite 2 iterations that didn't fully succeed, the agent never committed broken code. The test-before-commit discipline kept the main branch clean.

### 4. Progress.txt is Critical
Each iteration adds detailed notes:
- What was attempted
- What worked/failed
- Why it failed
- What to try next

This creates a "knowledge base" that prevents repeated mistakes.

### 5. Realistic Complexity
This isn't a toy example. The username race condition is a real-world problem that requires:
- Database-level constraints
- Data migration strategy
- Error handling at the app level

The Ralph Loop handles real development complexity.

## Using This Example

### 1. Study the PRD Structure
Look at `feature-prd.md` and `prd.json` to see how requirements are broken down:
- User stories with clear scope
- Specific, testable acceptance criteria
- Time estimates
- Technical requirements

### 2. Follow the Iteration Log
Read `progress.txt` chronologically to see:
- How the agent approaches each task
- How it handles failures
- How it documents learnings
- How knowledge accumulates

### 3. Understand the Pattern
```
Read PRD + progress.txt → 
Select next incomplete task → 
Implement → 
Test → 
Success? (Yes: Commit & update PRD, No: Document in progress.txt) → 
Repeat
```

### 4. Apply to Your Project
Key requirements for successful Ralph Loop:
- ✅ Well-structured PRD with testable criteria
- ✅ Automated tests that can verify success
- ✅ Project instructions in AGENTS.md
- ✅ Progress tracking (progress.txt)
- ✅ State management (prd.json)

## File Structure Details

```
tutorial-example/
├── README.md                    # This file
├── feature-prd.md               # Human-readable requirements
├── prd.json                     # Machine-readable task tracking
├── progress.txt                 # Iteration history and learnings
├── AGENTS.md                    # Project-specific instructions
└── iterations/                  # Code snapshots (if included)
    ├── iteration-1/             # After successful profile GET
    ├── iteration-2/             # After error handling
    └── iteration-3/             # After partial update implementation
```

## Adapting This Pattern

### For Your Language/Framework
- Python → FastAPI/Flask endpoints (shown here)
- Node.js → Express/Fastify routes
- Go → net/http or Gin handlers
- Java → Spring Boot controllers

The pattern works regardless of stack.

### For Your Domain
- REST APIs → Endpoints and tests (shown here)
- CLI tools → Commands and integration tests
- Libraries → Public APIs and unit tests
- Infrastructure → Terraform/config and validation

The pattern adapts to any domain with testable outcomes.

### For Your Scale
- Single feature → One PRD (shown here)
- Epic/milestone → Multiple PRDs in sequence
- Multi-repo → Campaign with Ralph Loop workers

Start small, scale up as confidence builds.

## Common Questions

**Q: Why didn't the agent fix the race condition immediately?**  
A: Real software problems often require investigation. The agent correctly identified the issue, documented it, and set up the next iteration with enough context to solve it. This is realistic and effective.

**Q: What if the agent gets stuck in a loop?**  
A: Set `max_iterations` in the workflow (e.g., 20). If stuck, progress.txt will show repeated attempts with the same approach—that's your signal to intervene and add guidance to AGENTS.md.

**Q: How much supervision is needed?**  
A: Early iterations: frequent checks. After patterns stabilize: spot checks. Before merging: full PR review. The workflow provides all history for review.

**Q: What about complex features requiring design decisions?**  
A: Add decision criteria to the PRD or AGENTS.md. For truly complex decisions, break the PRD into phases with human gates between them.

## Next Steps

1. **Read the PRD** (`feature-prd.md`) to understand the feature
2. **Study the progress log** (`progress.txt`) to see iteration patterns
3. **Review the workflow** (in main README.md) to see how it all connects
4. **Try it yourself** with a simple feature in your repository

## Related Documentation

- [Ralph Loop Guide](../README.md) - Complete pattern documentation
- [Tutorial](../../../../docs/src/content/docs/guides/ralph-loop.md) - Step-by-step setup
- [Safe Outputs](https://githubnext.github.io/gh-aw/reference/safe-outputs/) - Git operations reference
- [ResearchPlanAssign](https://githubnext.github.io/gh-aw/guides/researchplanassign/) - Combine with planning

---

> [!NOTE]
> This is a realistic example showing both successes and failures. Real software development involves iteration, learning, and recovery from setbacks. The Ralph Loop embraces this reality.
