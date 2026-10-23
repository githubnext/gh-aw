---
description: Refresh the hash-locked dependencies used by the repository's shared Python chart and NLP imports.
---

# Refresh shared Python dependency locks

The repository-owned `shared/python-dataviz.md`, `shared/trending-charts-simple.md`,
and `shared/python-nlp.md` imports install complete dependency closures from
`.github/workflows/requirements-python-{scientific,nlp}.txt`. Every package has
an exact version and SHA-256 distribution hashes. Installation requires hashes
and prebuilt wheels; missing hashes or incompatible wheels fail the setup.

These locks target the imports' uv-managed CPython 3.12 on Linux x86_64
(manylinux 2.28). Keep the lockfiles in the repository checkout, including any
sparse checkout configuration. Copying these imports to another repository also
requires copying their lockfiles.

Edit the corresponding `.in` file, then regenerate both locks from the repository
root using uv:

```bash
uv pip compile --python-version 3.12 --python-platform x86_64-manylinux_2_28 --only-binary=:all: --generate-hashes --no-annotate .github/workflows/requirements-python-scientific.in -o .github/workflows/requirements-python-scientific.txt
uv pip compile --python-version 3.12 --python-platform x86_64-manylinux_2_28 --only-binary=:all: --generate-hashes --no-annotate -c .github/workflows/requirements-python-scientific.txt .github/workflows/requirements-python-nlp.in -o .github/workflows/requirements-python-nlp.txt
```

Keep only SHA-256 hashes if a registry mirror also supplies weaker digests.
Review version and hash changes, then verify the combined wheel closure with
`pip download --require-hashes --only-binary=:all: --python-version 3.12
--platform manylinux_2_28_x86_64 --platform manylinux2014_x86_64
-r .github/workflows/requirements-python-scientific.txt
-r .github/workflows/requirements-python-nlp.txt --dest <temporary-directory>`.
Run `make recompile` and `go test ./pkg/workflow -run '^TestSharedPython'`.

The NLP lock is constrained by the scientific lock so their common dependencies
remain identical. Installs are additive rather than `uv pip sync`: both imports
share a virtual environment, and syncing either lock would remove packages
installed by the other import. Virtual environments are created without seeding
unlocked pip or setuptools packages.

This guarantee covers package installation for these three imports, not the
managed interpreter download or NLP corpus downloads. Other workflow-owned
Python installs, compiler-generated npm engine/SDK/LSP installs, and arbitrary
engine version expressions require separate hardening. `npm ci` does not support
global installs, so replacing global npm commands requires an explicit
cross-platform CLI layout and PATH compatibility design.
