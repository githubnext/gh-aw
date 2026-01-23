---
on: workflow_dispatch
engine: copilot
tools:
  cache-memory: true
imports:
  - shared/mcp/chroma.md
---

# Test Chroma MCP Integration

This workflow tests the Chroma MCP server integration with cache-memory.

**Tasks:**
1. List available embedding functions using `mcp_known_embedding_functions`
2. Create a test collection called "test-docs"
3. Add a few sample documents to the collection
4. Query the collection for semantic search
5. Verify the data persists in cache-memory
