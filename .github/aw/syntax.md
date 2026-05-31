---
description: Compact index for the GitHub Agentic Workflows frontmatter schema.
---

# Frontmatter Schema Index

Use the smallest relevant reference instead of loading one large schema file.

| Topic | File |
|---|---|
| Core GitHub Actions fields (`on`, `permissions`, `runs-on`, `steps`, `env`, `secrets`) | [syntax-core.md](syntax-core.md) |
| Agentic workflow specific fields (`strict`, `bots`, `labels`, metadata, engine-specific fields) | [syntax-agentic.md](syntax-agentic.md) |
| Cache configuration, tools, imports, and permission patterns | [syntax-tools-imports.md](syntax-tools-imports.md) |

## Usage Guidance

- Load only the section required for the current task.
- Prefer the dedicated topic files over copying schema details into creator or updater prompts.
- Keep examples short and route deep detail to the relevant syntax sub-file.
