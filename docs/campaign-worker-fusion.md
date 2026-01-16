# Campaign Worker Workflow Fusion

Campaign worker workflow fusion adapts existing workflows for campaign use by adding `workflow_dispatch` triggers and storing them in `.github/workflows/campaigns/<campaign-id>/` folders. This enables campaign orchestrators to dispatch workers on-demand using the `dispatch_workflow` safe output, while maintaining clear lineage through metadata (`campaign-worker: true`, `campaign-id`, `source-workflow`). The separate folder structure supports future pattern analysis to identify which workflow patterns work best for different campaign types.

See [Campaign Examples](./docs/src/content/docs/examples/campaigns.md) for usage examples.
