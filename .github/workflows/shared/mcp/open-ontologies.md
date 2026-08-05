---
mcp-servers:
  open-ontologies:
    container: "ghcr.io/fabio-rovai/open-ontologies"
    version: "latest"
    entrypointArgs:
      - "serve"
    allowed:
      # Validation, loading, and inspection (read-only over the ontology graph)
      - "onto_validate"
      - "onto_load"
      - "onto_stats"
      - "onto_query"
      - "onto_lint"
      # Reasoning and classification
      - "onto_reason"
      - "onto_classify_el"
      # Diff / lineage / governance inspection
      - "onto_cq_run"
      - "onto_verify_cq"
      - "onto_cq_verdicts_list"
      - "onto_owl_shacl_coevolve_check"
      - "onto_segment_retrieve"
      - "onto_eval_alignment"
      # Excluded mutating/write operations:
      # - "onto_save", "onto_version", "onto_enforce", "onto_action_apply",
      #   "onto_policy_register" and other state-changing tools are not
      #   allow-listed to keep this shared wrapper read-only/analysis-focused.
---

<!--
# Open Ontologies MCP Server
# Rust MCP server for AI-native ontology engineering: validate, classify,
# query, reason over, and govern RDF/OWL ontologies using an in-memory
# Oxigraph triple store.
#
# Upstream:
# - https://github.com/fabio-rovai/open-ontologies
# - Docker image: ghcr.io/fabio-rovai/open-ontologies
#
# Why this wrapper exists:
# - Exposes Open Ontologies' 70+ `onto_*` tools as a containerized MCP server.
# - Restricts access to read-only validation, reasoning, and query tools so
#   workflows can compute/inspect ontologies without mutating state or
#   requiring a persisted store.
#
# Usage:
#   imports:
#     - shared/mcp/open-ontologies.md
-->

Use the Open Ontologies MCP tools when a task involves building, validating, or
reasoning about ontologies (RDF/OWL/Turtle), taxonomies, or knowledge graphs.
Typical flow: `onto_validate` a Turtle/OWL document, `onto_load` it into the
in-memory store, `onto_reason` to materialize inferred triples via OWL-RL,
then `onto_stats`, `onto_lint`, or `onto_query` to inspect the result.
Use `onto_classify_el` for subsumption/classification tasks and
`onto_cq_run`/`onto_verify_cq` to check competency questions against the
ontology.
