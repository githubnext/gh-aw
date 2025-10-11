---
"gh-aw": patch
---

Add lockfile statistics analysis workflow for nightly audits

Adds a new agentic workflow that performs comprehensive statistical and structural analysis of all `.lock.yml` files in the repository, publishing insights to the "audits" discussion category. The workflow runs nightly at 3am UTC and provides valuable visibility into workflow usage patterns, trigger types, safe outputs, file sizes, and structural characteristics.
