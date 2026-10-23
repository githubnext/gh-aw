# ADR-71298: Hash-Lock Shared Python Chart and NLP Dependencies

**Date**: 2026-10-23
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The repository-owned shared workflow imports `shared/python-dataviz.md`, `shared/trending-charts-simple.md`, and `shared/python-nlp.md` install Python scientific (NumPy, Pandas, Matplotlib, Seaborn, SciPy) and NLP (NLTK, scikit-learn, TextBlob, WordCloud) libraries into a single shared virtual environment at `/tmp/gh-aw/python/venv`. These imports pinned only the nine direct package versions, so transitive dependencies were resolved at runtime and downloaded artifacts were not verified, making installation non-reproducible and exposed to supply-chain substitution (issue #68043). The venv was also created with `--seed`, pulling in unlocked `pip`/`setuptools`. Because both import families share one environment and can run in either order, any locking approach must not let one import remove the other's packages. The fix is scoped to these imports; it explicitly does not cover compiler-generated npm installs, other workflow-owned Python installs, managed interpreter downloads, or NLP corpus downloads.

### Decision

We will commit complete, hash-pinned dependency lockfiles (`.github/workflows/requirements-python-scientific.txt` and `requirements-python-nlp.txt`, generated from `.in` files with `uv pip compile --generate-hashes`) and install them with `uv pip install --require-hashes --only-binary=:all:`. The NLP lock is compiled with the scientific lock as a constraint so shared dependencies have identical versions and hashes, and installs remain additive (`uv pip install`, not `uv pip sync`) so the two imports can coexist in the shared venv. Virtual environments will be created without `--seed`. The locks target uv-managed CPython 3.12 on Linux x86_64 (manylinux 2.28), and maintenance is documented in `.github/aw/python-dependency-locks.md`, with regression tests in `pkg/workflow/shared_python_workflow_test.go`.

### Alternatives Considered

#### Alternative 1: Keep direct version pins only

Continue installing `pkg==x.y.z` for the nine direct dependencies. This is simplest and keeps imports self-contained, but transitive dependencies still float and no artifact hash is verified, which is the exact gap issue #68043 identified. Rejected because it does not make installs deterministic.

#### Alternative 2: Use `uv pip sync` (exact environment sync) per lockfile

Syncing makes the environment exactly match one lockfile, which is the strictest reproducibility guarantee. Rejected because the scientific and NLP imports share one virtual environment; syncing either lock would uninstall the packages installed by the other import.

#### Alternative 3: A single combined lockfile for all shared Python imports

One lockfile covering both scientific and NLP packages would avoid cross-lock constraint management. Not chosen [TODO: verify rationale] — evidence suggests it would force chart-only workflows to install the NLP closure, increasing install time and attack surface for workflows that do not need it.

### Consequences

#### Positive
- Installs of the shared chart/NLP imports are deterministic, including transitive dependencies, and reject artifacts whose SHA-256 hash does not match.
- Wheel-only installation removes build-time code execution from source distributions during setup.
- Constraining the NLP lock by the scientific lock keeps common dependencies identical, so import order does not matter.
- Removing `--seed` eliminates unlocked `pip`/`setuptools` from the environment.

#### Negative
- The lockfiles are platform-specific (CPython 3.12, manylinux 2.28 x86_64); other platforms or Python versions will fail installation until locks are regenerated.
- Workflows using these imports now depend on the lockfiles being present in the repository checkout (including sparse checkouts), and copying the imports to another repository requires copying the lockfiles too.
- Version bumps require a manual two-step regeneration procedure and review of large lockfile diffs (~1,400 lines added).
- Additive installs mean the environment is not guaranteed to contain *only* locked packages if other steps install extra packages.

#### Neutral
- The nine direct dependency versions are unchanged; 21 generated workflow lock files were regenerated to pick up the new install commands.
- `.gitignore` gains exceptions for the two new requirements files, following the existing `requirements-agentrx.txt` pattern.
- This is a partial remediation of #68043; npm engine/SDK/LSP installs, other Python installs, interpreter downloads, and NLTK corpus downloads remain unlocked and need separate decisions.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
