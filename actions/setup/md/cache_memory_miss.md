> [!WARNING]
> <details>
> <summary>Cache Configuration Problem: cache miss detected after cache restore succeeded.</summary>
> 
> The agent reported a cache miss (`missing_data` with `reason: cache_memory_miss`) after the workflow successfully restored cache-memory for this run. This likely indicates the prompt is misconfigured and the agent cannot locate the correct file path within the cache directory.
> 
> This warning is shown only when a cache restore matched an existing key. Cache misses can be expected on first runs and on branches where `actions/cache` has no visible entries due to branch scoping.
> 
> Review the [cache-memory configuration](https://github.github.com/gh-aw/reference/cache-memory/) and ensure the agent prompt correctly references files inside the cache directory.
> 
> **File naming convention:** Cache files are stored at `/tmp/gh-aw/cache-memory/` (default) or `/tmp/gh-aw/cache-memory-{id}/` for additional caches. Use descriptive file and directory names with subdirectories for organization.
> 
> </details>
