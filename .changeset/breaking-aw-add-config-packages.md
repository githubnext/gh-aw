---
"gh-aw": major
---

`gh aw add` now errors for packages that declare `aw.yml` config steps instead of installing with a TODO message.

**⚠️ Breaking Change**: Running `gh aw add <package>` on a package that declares interactive config steps in `aw.yml` now returns an error instead of silently installing with a placeholder TODO.

**Migration guide:**
- Replace `gh aw add <package>` with `gh aw add-wizard <package>` for any package that declares interactive config steps in `aw.yml`.
- If running in non-interactive automation, check whether the package supports a non-interactive installation path.
