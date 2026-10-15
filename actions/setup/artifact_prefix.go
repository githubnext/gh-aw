package setup

import _ "embed"

// ArtifactPrefixScript is also inlined into jobs that do not otherwise need setup.
//
//go:embed sh/compute_artifact_prefix.sh
var ArtifactPrefixScript string

//go:embed sh/write_artifact_producer.sh
var ArtifactProducerScript string

//go:embed sh/resolve_artifact_producer.sh
var ArtifactProducerResolverScript string
