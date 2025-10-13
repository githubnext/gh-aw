---
"gh-aw": minor
---

Add support for multiple cache-memory configurations with array notation and optional descriptions

Implemented support for multiple cache-memory configurations with a simplified, unified array-based structure. This feature allows workflows to define multiple caches using array notation, each with a unique ID and optional description. The implementation maintains full backward compatibility with existing single-cache configurations (boolean, nil, or object notation).

Key features:
- Unified array structure for all cache configurations
- Support for multiple caches with explicit IDs
- Optional description field for each cache
- Backward compatibility with existing workflows
- Smart path handling for single cache with ID "default"
- Duplicate ID validation at compile time
- Import support for shared workflows
