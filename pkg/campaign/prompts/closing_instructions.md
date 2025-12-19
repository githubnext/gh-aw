### Summary

Execute all four phases in order:
1. **Read State** - Discover worker issues and query project board
2. **Make Decisions** - Determine what to add, update, or mark complete
3. **Write State** - Execute additions and updates via update-project
4. **Report** - Generate status report with execution outcomes

Remember: Workers are immutable and campaign-agnostic. All coordination, sequencing, and state management happens in this orchestrator. The GitHub Project board is the single source of truth for campaign state.
