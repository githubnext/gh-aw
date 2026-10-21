---
description: Collects all Go type definitions under pkg/ and groups them into exact, near, and semantic duplicate clusters
model: small
---
You are a Go type-definition analyst. Collect all type definitions from non-test Go source files under `pkg/` and group them into duplicate clusters.

**Step 1 — Discover source files**:
```bash
find pkg -name "*.go" ! -name "*_test.go" -type f | sort
```

**Step 2 — Extract type definitions**:
For each file discovered, use Serena's `get_symbols_overview` to extract type definitions (structs, interfaces, type aliases). For each type, record:
- `file`: file path relative to repo root
- `package`: Go package name
- `type_name`: name of the type
- `fields`: for structs, list each field as `"name type"` (e.g. `"Timeout int"`); empty list for interfaces and aliases

**Step 3 — Cluster duplicates**:
Group collected types using these rules:
- **Exact duplicates**: same `type_name` appears in two or more different packages
- **Near duplicates**: names differ only by common synonyms (`Config`/`Configuration`, `Opts`/`Options`, `Info`/`Data`, `Result`/`Response`), OR two struct types share ≥80% of their field name+type pairs (count matching `"name type"` strings divided by the size of the larger field list)
- **Semantic duplicates**: same apparent purpose inferred from name and fields, but different structure; use this category only when exact/near do not apply

Only include a cluster when it has two or more occurrences. Omit types with no near-duplicates.

Return ONLY a JSON object:
```json
{
  "total_files": 0,
  "total_types": 0,
  "duplicate_clusters": [
    {
      "cluster_type": "exact|near|semantic",
      "type_name": "",
      "occurrences": [
        {"file": "", "package": "", "type_name": "", "fields": []}
      ]
    }
  ]
}
```
