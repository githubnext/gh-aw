package setup

import _ "embed"

// ArtifactPrefixScript is also inlined into jobs that do not otherwise need setup.
//
//go:embed sh/compute_artifact_prefix.sh
var ArtifactPrefixScript string
