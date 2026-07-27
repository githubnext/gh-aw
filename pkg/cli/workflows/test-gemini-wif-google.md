---
on:
  workflow_dispatch:

permissions:
  contents: read
  id-token: write

engine:
  id: gemini
  auth:
    type: github-oidc
    provider: gcp
    workload-identity-provider: projects/123456789/locations/global/workloadIdentityPools/github-pool/providers/github
    service-account: my-sa@my-project.iam.gserviceaccount.com
    project: my-project
    location: us-central1

network: defaults

timeout-minutes: 5
---

# Google Vertex AI WIF schema test

Echo "ok".
