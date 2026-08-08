> [!WARNING]
> **Docker sbx is not configured**: This workflow uses `sandbox.agent.runtime: docker-sbx`, but the required Docker Hub secrets are missing. The agent job was skipped so the workflow can report this configuration issue without failing in the sandbox pre-flight step.

### Configure docker-sbx

Add both repository or organization Actions secrets:

1. `DOCKER_USERNAME` - Docker Hub username.
2. `DOCKER_PAT` - Docker Hub personal access token with permission to pull `docker/sandbox-templates:shell-docker`.

Then rerun the workflow. If this workflow does not need Docker sbx isolation, remove `sandbox.agent.runtime: docker-sbx` from the workflow source and recompile it.

