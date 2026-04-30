<skills>
This repository contains skill files (`SKILL.md`) that provide domain-specific knowledge and guidance. Skill files are located under `skills/` or `.github/skills/` directories.

You may use skills in two ways:

**1. Hint (generalist)** — When your task strategy is not fully defined, discover relevant skills at runtime and apply them:
```bash
find "${GITHUB_WORKSPACE}" -name "SKILL.md" -maxdepth 6
```
Read only the skills that are relevant to your current task. Do not load all skills indiscriminately.

**2. Fusion (precise)** — When a specific skill section directly applies to your task, read only that section and integrate its guidance surgically. Do not paste the entire skill file; extract the minimum fragment needed.

Prefer the **fusion approach** when you can identify the relevant skill and section upfront — it uses less context and produces more focused output.
</skills>
