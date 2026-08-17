# Meet Your Squad 🕵️

**Universe:** The Usual Suspects

Your team was cast for **gh-aw** — a Go CLI extension (`gh aw`) that compiles Markdown + YAML frontmatter agentic workflows into standard GitHub Actions workflows.

## The Team

| Name | Role | Specialty | How to talk to them |
|------|------|-----------|----------------------|
| Keaton | Lead / Architect | Architecture, module boundaries, release strategy | `@copilot` mention or `squad:keaton` label |
| Fenster | Compiler & Workflow Engine Engineer | `pkg/parser`, workflow compiler, safe-outputs, engine adapters | `squad:fenster` label |
| McManus | CI/CD & Actions Platform Engineer | `actions/`, `Makefile`, `Dockerfile`, run/audit tooling | `squad:mcmanus` label |
| Hockney | Test Engineer | Unit/integration tests, `make test-unit`, coverage | `squad:hockney` label |
| Verbal | DevRel / Docs | `docs/`, README, skills, changelog | `squad:verbal` label |
| Kujan | Security Engineer | Injection prevention, action pinning, permission scoping | `squad:kujan` label |

## Always-On Support

| Name | Role | Notes |
|------|------|-------|
| Scribe | Session Logger | Records decisions and orchestration logs |
| Ralph | Work Monitor | Keeps the work queue moving |
| Rai | RAI Reviewer | Responsible AI review gate |
| Fact Checker | Fact Checker | Verifies claims, plays devil's advocate |

## How to Work With Your Squad

- Assign issues with a `squad:{name}` label (color `9B8FCC`) to route work to a specific team member.
- Use `/squad research`, `/squad plan`, `/squad triage` on any issue to kick off the planning lifecycle.
- Use `/squad implement` to dispatch an isolated implementation worker for an issue or epic.
- See `.squad/routing.md` for the full domain-to-agent routing table.

## What Happened Here

**Repo analysis:**
- **Languages/frameworks:** Go (~2,850 files), Markdown/YAML workflow definitions, Starlight docs (MDX).
- **Structure:** `pkg/` (parser, cli, github, linters, etc.), `cmd/`, `actions/` (custom GitHub Actions), `docs/` (Starlight site), `.github/skills/` (Copilot skill library), `linters/`.
- **CI/CD:** `Makefile`-driven build/test/lint, `Dockerfile`, GitHub Actions workflows.
- **Testing:** Extensive Go unit tests with an impacted-first `test-unit` target.
- **Docs:** Large Starlight documentation site plus 100+ Copilot skill files.

**Rationale:** A compiler-heavy Go CLI project needs a dedicated compiler/engine specialist (Fenster), a CI/Actions platform owner (McManus) given the custom `actions/` directory, strong test discipline (Hockney) given the large Go test suite, docs/DevRel (Verbal) given the extensive Starlight site and skill library, and security (Kujan) given the project's focus on sandboxing untrusted agent input and safe-outputs. Keaton leads as architect given the scale (2850+ files) and cross-cutting compiler/CLI concerns.

**Cast on:** 2026-08-17
