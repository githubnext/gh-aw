> [!WARNING]
> <details>
> <summary>Cache Configuration Problem: cache miss detected despite cache-memory being configured.</summary>
> 
> The agent reported a cache miss (`missing_data` with `reason: cache_memory_miss`) even though cache-memory is configured and was available. This likely indicates the prompt is misconfigured and the agent cannot locate the correct file path within the cache directory.
> 
> Review the [cache-memory configuration](https://github.github.com/gh-aw/reference/cache-memory/) and ensure the agent prompt correctly references files inside the cache directory.
> 
> **File naming convention:** Cache files are stored at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/` for additional caches. Use descriptive file and directory names with subdirectories for organization.
> 
> </details>
