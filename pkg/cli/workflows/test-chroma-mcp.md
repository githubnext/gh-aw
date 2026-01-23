---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
tools:
  cache-memory: true
imports:
  - shared/mcp/chroma.md
---

# Test Chroma MCP Integration

This workflow demonstrates the Chroma MCP server integration with persistent cache-memory storage.

Chroma provides vector database and semantic search capabilities for AI agents. This workflow shows how to:
- Create and manage collections
- Add documents with embeddings
- Perform semantic search queries
- Persist data across workflow runs using cache-memory

**Test Tasks:**

1. **Initialize**: List available embedding functions using `mcp_known_embedding_functions`
2. **Create Collection**: Create a test collection called "test-docs" with an appropriate embedding function
3. **Add Documents**: Add sample documents to the collection with IDs like:
   - doc1: "GitHub Actions is a CI/CD platform"
   - doc2: "Chroma is a vector database for AI applications"
   - doc3: "MCP enables standardized AI tool integration"
4. **Semantic Search**: Query the collection for "What is an AI database?" and verify it returns relevant results
5. **Verify Persistence**: Check that the collection exists and contains the documents

**Expected Behavior:**
- The collection and documents should persist in `/tmp/gh-aw/cache-memory/` 
- On subsequent runs, the data should be available from the cache
- Semantic search should return relevant documents based on meaning, not just keywords
