package conformance

import (
	"fmt"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/github/gh-aw/actions/setup"
)

const identityPrefix = "${{ steps.artifact-prefix.outputs.identity_prefix }}"
const legacyArtifactUpload = "actions/upload-artifact@c6a366c94c3e0affe28c06c8df20a878f24da3cf"
const artifactMetadataRoot = "${{ runner.temp }}/gh-aw/artifact-producer"

var artifactSource = regexp.MustCompile(`\$\{\{ steps\.artifact-source-([A-Za-z_][A-Za-z0-9_-]*)\.outputs\.prefix \}\}`)

func sourcePrefix(producer string) string {
	return "${{ steps.artifact-source-" + producer + ".outputs.prefix }}"
}

func metadataDirectory(producer string) string {
	return path.Join(artifactMetadataRoot, producer)
}

func canonicalArtifactStep(s step, env map[string]any, run string) bool {
	return s.Uses == "" && s.Shell == "bash" && maps.Equal(s.Env, env) &&
		strings.TrimSpace(s.Run) == run
}

func uniqueStep(j job, id string) (step, int, bool) {
	var found step
	position, count := -1, 0
	for index, s := range j.Steps {
		if s.ID == id {
			found, position = s, index
			count++
		}
	}
	return found, position, count == 1
}

func trustedProducerDescriptor(doc document, producer string) (int, bool) {
	j, exists := doc.Jobs[producer]
	_, index, unique := uniqueStep(j, "artifact-prefix")
	writer, hasWriter := stepAt(j.Steps, index+1)
	upload, hasUpload := stepAt(j.Steps, index+2)
	if !exists || !unique || !hasWriter || !hasUpload || !trustedLocalPrefix(doc, j, index+1) {
		return 0, false
	}
	env := map[string]any{"INVOCATION_ID": identityPrefix, "PRODUCER_JOB": producer,
		"PRODUCER_ATTEMPT": "${{ github.run_attempt }}", "METADATA_DIRECTORY": metadataDirectory(producer)}
	if !canonicalArtifactStep(writer, env, embeddedArtifactScript(setup.ArtifactProducerScript)) ||
		writer.If != "" || writer.ContinueOnError != nil && writer.ContinueOnError != false {
		return 0, false
	}
	if !strings.HasPrefix(upload.Uses, "actions/upload-artifact@") || upload.If != "" ||
		upload.ContinueOnError != nil && upload.ContinueOnError != false ||
		upload.With["name"] != identityPrefix+"producer-"+producer ||
		upload.With["path"] != path.Join(metadataDirectory(producer), "producer.json") ||
		upload.With["if-no-files-found"] != "error" {
		return 0, false
	}
	if upload.Uses != legacyArtifactUpload && upload.With["overwrite"] != true {
		return 0, false
	}
	count := 0
	for _, candidate := range doc.Jobs {
		for _, s := range candidate.Steps {
			if strings.HasPrefix(s.Uses, "actions/upload-artifact@") && s.With["name"] == upload.With["name"] {
				count++
			}
		}
	}
	return index + 2, count == 1
}

func trustedSourceResolver(doc document, j job, before int, producer string) bool {
	needs, err := dependencies(j.Needs)
	if err != nil || !slices.Contains(needs, producer) || !trustedLocalPrefix(doc, j, before) {
		return false
	}
	metadata, downloadAt, uniqueDownload := uniqueStep(j, "artifact-metadata-"+producer)
	resolver, resolveAt, uniqueResolver := uniqueStep(j, "artifact-source-"+producer)
	_, prefixAt, _ := uniqueStep(j, "artifact-prefix")
	if !uniqueDownload || !uniqueResolver || downloadAt <= prefixAt ||
		resolveAt != downloadAt+1 || resolveAt >= before {
		return false
	}
	if !strings.HasPrefix(metadata.Uses, "actions/download-artifact@") || metadata.If != "always()" ||
		len(metadata.With) != 2 || metadata.With["name"] != identityPrefix+"producer-"+producer ||
		metadata.With["path"] != metadataDirectory(producer) {
		return false
	}
	env := map[string]any{"INPUTS_JSON": "${{ toJSON(inputs) }}", "INVOCATION_ID": identityPrefix,
		"PRODUCER_JOB": producer, "PRODUCER_RESULT": "${{ needs." + producer + ".result }}",
		"METADATA_DIRECTORY": metadataDirectory(producer)}
	expected := embeddedArtifactScript(setup.ArtifactProducerResolverScript) + "\n" + embeddedPrefixScript()
	if !canonicalArtifactStep(resolver, env, expected) ||
		resolver.If != "always() && steps.artifact-metadata-"+producer+".outcome == 'success'" {
		return false
	}
	for _, s := range []step{metadata, resolver} {
		if s.ContinueOnError != nil && s.ContinueOnError != false && s.ContinueOnError != true {
			return false
		}
	}
	return true
}

func sourceUploadExists(doc document, producer, pattern string, publishedAt int) bool {
	found := false
	for jobName, j := range doc.Jobs {
		for index, s := range j.Steps {
			name, ok := s.With["name"].(string)
			if !ok || !strings.HasPrefix(s.Uses, "actions/upload-artifact@") ||
				!strings.HasPrefix(artifactName(name), "invocation:") {
				continue
			}
			matched, err := path.Match(pattern, strings.TrimPrefix(artifactName(name), "invocation:"))
			if err != nil || matched && (jobName != producer || index <= publishedAt ||
				!strings.HasPrefix(name, localInvocationPrefix)) {
				return false
			}
			found = found || matched
		}
	}
	return found
}

func trustedSourceDownload(doc document, j job, index int, s step, selector string) bool {
	if !strings.HasPrefix(s.Uses, "actions/download-artifact@") {
		return false
	}
	for _, key := range []string{"run-id", "repository"} {
		if _, foreign := s.With[key]; foreign {
			return false
		}
	}
	for selection := range strings.SplitSeq(strings.TrimSuffix(strings.TrimPrefix(selector, "{"), "}"), ",") {
		expression := artifactSource.FindString(selection)
		artifact, hasPrefix := strings.CutPrefix(selection, expression)
		if expression == "" || !hasPrefix {
			return false
		}
		producer := strings.TrimSuffix(strings.TrimPrefix(expression, "${{ steps.artifact-source-"), ".outputs.prefix }}")
		publishedAt, trusted := trustedProducerDescriptor(doc, producer)
		if !trusted || !trustedSourceResolver(doc, j, index, producer) ||
			requirePredicate(s.If, "steps.artifact-source-"+producer+".outcome == 'success'") != nil {
			return false
		}
		if !sourceUploadExists(doc, producer, artifact, publishedAt) &&
			!optionalSourceFallback(doc, producer, artifact, selector, publishedAt) {
			return false
		}
	}
	return true
}

func optionalSourceFallback(doc document, producer, artifact, selector string, publishedAt int) bool {
	prefix := sourcePrefix("agent")
	return producer == "agent" && artifact == "agent-output-fallback" &&
		selector == "{"+prefix+"agent,"+prefix+"agent-output-fallback}" &&
		artifactAbsent(doc, "invocation:agent-output-fallback") &&
		sourceUploadExists(doc, producer, "agent", publishedAt)
}

func checkArtifactSources(doc document) []string {
	var violations []string
	for name, j := range doc.Jobs {
		for index, s := range j.Steps {
			if !isArtifactAction(s) {
				continue
			}
			for _, key := range []string{"name", "pattern"} {
				selector, ok := s.With[key].(string)
				if !ok {
					continue
				}
				if artifactSource.MatchString(selector) && !trustedSourceDownload(doc, j, index, s, selector) {
					violations = append(violations, fmt.Sprintf("ArtifactProvenance: %s lacks trusted producer-attempt handoff", name))
				}
				if strings.Contains(selector, identityPrefix) && !trustedLocalPrefix(doc, j, index) {
					violations = append(violations, "ArtifactProvenance: "+name+" lacks trusted stable invocation identity")
				}
			}
		}
	}
	return violations
}
