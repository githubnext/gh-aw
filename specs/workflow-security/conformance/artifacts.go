package conformance

import (
	"fmt"
	"slices"
	"strings"

	"github.com/github/gh-aw/actions/setup"
)

const invocationPrefix = "${{ needs.activation.outputs.artifact_prefix }}"
const localInvocationPrefix = "${{ steps.artifact-prefix.outputs.prefix }}"
const externalPrefixScript = `bash "${RUNNER_TEMP}/gh-aw/actions/compute_artifact_prefix.sh"`

func embeddedPrefixScript() string {
	return embeddedArtifactScript(setup.ArtifactPrefixScript)
}

func embeddedArtifactScript(script string) string {
	var lines []string
	for line := range strings.SplitSeq(script, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func trustedPrefixStep(s step) bool {
	if s.If != "" || s.ContinueOnError != nil && s.ContinueOnError != false || s.Uses != "" {
		return false
	}
	if len(s.Env) != 1 || s.Env["INPUTS_JSON"] != "${{ toJSON(inputs) }}" {
		return false
	}
	run := strings.TrimSpace(s.Run)
	if run == externalPrefixScript {
		return s.Shell == "" || s.Shell == "bash"
	}
	return s.Shell == "bash" && run == embeddedPrefixScript()
}

func trustedLocalPrefix(doc document, j job, before int) bool {
	for _, env := range []map[string]any{doc.Env, j.Env} {
		for _, key := range []string{"GITHUB_RUN_ATTEMPT", "GITHUB_OUTPUT", "BASH_ENV", "ENV"} {
			if _, overridden := env[key]; overridden {
				return false
			}
		}
	}
	count := 0
	for index, s := range j.Steps {
		if s.ID == "artifact-prefix" {
			count++
			if index >= before || !trustedPrefixStep(s) {
				return false
			}
		}
	}
	return count == 1
}

func trustedInvocationPrefix(doc document) bool {
	activation := doc.Jobs["activation"]
	return activation.Outputs["artifact_prefix"] == localInvocationPrefix &&
		trustedLocalPrefix(doc, activation, len(activation.Steps))
}

func checkArtifactPrefixes(doc document) []string {
	var violations []string
	for name, j := range doc.Jobs {
		for index, s := range j.Steps {
			if !isArtifactAction(s) {
				continue
			}
			selector := fmt.Sprint(s.With["name"]) + fmt.Sprint(s.With["pattern"])
			trusted := true
			if strings.Contains(selector, localInvocationPrefix) {
				trusted = trustedLocalPrefix(doc, j, index)
			}
			if strings.Contains(selector, invocationPrefix) {
				needs, err := dependencies(j.Needs)
				trusted = trusted && err == nil && slices.Contains(needs, "activation") &&
					trustedInvocationPrefix(doc)
			}
			if strings.HasPrefix(s.Uses, "actions/download-artifact@") && len(agentDownloadNames(s)) > 0 {
				needs, err := dependencies(j.Needs)
				trusted = trusted && err == nil && slices.Contains(needs, "agent")
			}
			if !trusted {
				violations = append(violations, "ArtifactProvenance: "+name+" artifact prefix lacks trusted preceding producer")
			}
		}
	}
	return violations
}

func isArtifactAction(s step) bool {
	return strings.HasPrefix(s.Uses, "actions/upload-artifact@") ||
		strings.HasPrefix(s.Uses, "actions/download-artifact@")
}

func artifactName(name string) string {
	for _, prefix := range []string{invocationPrefix, localInvocationPrefix, sourcePrefix("agent")} {
		if artifact, hasPrefix := strings.CutPrefix(name, prefix); hasPrefix {
			return "invocation:" + artifact
		}
	}
	return name
}

func agentDownloadNames(s step) []string {
	for _, prefix := range []string{"", invocationPrefix, localInvocationPrefix, sourcePrefix("agent")} {
		name := prefix + "agent"
		if s.With["name"] == name {
			return []string{artifactName(name)}
		}
		if s.With["pattern"] == "{"+name+","+prefix+"agent-output-fallback}" {
			return []string{artifactName(name), artifactName(prefix + "agent-output-fallback")}
		}
	}
	return nil
}

func artifactAbsent(doc document, name string) bool {
	for _, j := range doc.Jobs {
		for _, s := range j.Steps {
			if strings.HasPrefix(s.Uses, "actions/upload-artifact@") &&
				artifactName(fmt.Sprint(s.With["name"])) == name {
				return false
			}
		}
	}
	return true
}

func checkArtifacts(doc document) []string {
	uploaded := map[string]struct{}{}
	for _, s := range doc.Jobs["agent"].Steps {
		if strings.HasPrefix(s.Uses, "actions/upload-artifact@") {
			uploaded[artifactName(fmt.Sprint(s.With["name"]))] = struct{}{}
		}
	}
	violations := checkArtifactPrefixes(doc)
	violations = append(violations, checkArtifactSources(doc)...)
	found := false
	for _, s := range doc.Jobs["safe_outputs"].Steps {
		if !strings.HasPrefix(s.Uses, "actions/download-artifact@") {
			continue
		}
		if _, ok := s.With["run-id"]; ok {
			violations = append(violations, "ArtifactProvenance: cross-run download")
		}
		if _, ok := s.With["repository"]; ok {
			violations = append(violations, "ArtifactProvenance: cross-repository download")
		}
		names := agentDownloadNames(s)
		primaryUploaded := false
		for index, name := range names {
			_, present := uploaded[name]
			if index == 0 {
				primaryUploaded = present
			}
			if !present {
				if index == 1 && primaryUploaded && artifactAbsent(doc, name) {
					continue
				}
				violations = append(violations, "ArtifactProvenance: missing agent upload for "+name)
			}
		}
		found = found || len(names) > 0
	}
	if !found {
		violations = append(violations, "ArtifactProvenance: no matching agent artifact handoff")
	}
	return violations
}
