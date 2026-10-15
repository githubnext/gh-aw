package conformance

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/actions/setup"
	"gopkg.in/yaml.v3"
)

func descriptorFixture(t *testing.T) document {
	t.Helper()
	doc := localArtifactFixture(t, true)
	agent := doc.Jobs["agent"]
	writer := step{Shell: "bash", Run: embeddedArtifactScript(setup.ArtifactProducerScript),
		Env: map[string]any{"INVOCATION_ID": identityPrefix, "PRODUCER_JOB": "agent",
			"PRODUCER_ATTEMPT": "${{ github.run_attempt }}", "METADATA_DIRECTORY": metadataDirectory("agent")}}
	upload := step{Uses: "actions/upload-artifact@0000000000000000000000000000000000000000",
		With: map[string]any{"name": identityPrefix + "producer-agent",
			"path": metadataDirectory("agent") + "/producer.json", "if-no-files-found": "error", "overwrite": true}}
	agent.Steps = append(agent.Steps[:1], append([]step{writer, upload}, agent.Steps[1:]...)...)
	doc.Jobs["agent"] = agent
	consumer := doc.Jobs["safe_outputs"]
	metadata := step{ID: "artifact-metadata-agent", If: "always()",
		Uses: "actions/download-artifact@0000000000000000000000000000000000000000",
		With: map[string]any{"name": identityPrefix + "producer-agent", "path": metadataDirectory("agent")}}
	resolver := step{ID: "artifact-source-agent", Shell: "bash",
		If:  "always() && steps.artifact-metadata-agent.outcome == 'success'",
		Run: embeddedArtifactScript(setup.ArtifactProducerResolverScript) + "\n" + embeddedPrefixScript(),
		Env: map[string]any{"INPUTS_JSON": "${{ toJSON(inputs) }}", "INVOCATION_ID": identityPrefix,
			"PRODUCER_JOB": "agent", "PRODUCER_RESULT": "${{ needs.agent.result }}",
			"METADATA_DIRECTORY": metadataDirectory("agent")}}
	consumer.Steps[1].With["pattern"] = "{" + sourcePrefix("agent") + "agent," + sourcePrefix("agent") + "agent-output-fallback}"
	consumer.Steps[1].If = "steps.artifact-source-agent.outcome == 'success'"
	consumer.Steps = append(consumer.Steps[:1], append([]step{metadata, resolver}, consumer.Steps[1:]...)...)
	doc.Jobs["safe_outputs"] = consumer
	return doc
}

func TestProducerAttemptArtifactProvenance(t *testing.T) {
	for _, mode := range []string{"required", "optional", "v3", "named-download", "additional-guard"} {
		t.Run(mode, func(t *testing.T) {
			doc := descriptorFixture(t)
			consumer, agent := doc.Jobs["safe_outputs"], doc.Jobs["agent"]
			switch mode {
			case "optional":
				consumer.Steps[1].ContinueOnError, consumer.Steps[2].ContinueOnError = true, true
			case "v3":
				agent.Steps[2].Uses = legacyArtifactUpload
				delete(agent.Steps[2].With, "overwrite")
			case "named-download":
				delete(consumer.Steps[3].With, "pattern")
				consumer.Steps[3].With["name"] = sourcePrefix("agent") + "agent"
			case "additional-guard":
				consumer.Steps[3].If = "always() && steps.artifact-source-agent.outcome == 'success' && inputs.enabled"
			}
			doc.Jobs["agent"], doc.Jobs["safe_outputs"] = agent, consumer
			if got := verify(doc, "daily"); len(got) != 0 {
				t.Fatalf("canonical producer-attempt handoff rejected: %v", got)
			}
		})
	}
}

func TestProducerDescriptorMutations(t *testing.T) {
	for _, mutation := range []string{"body", "attempt", "identity", "producer", "path", "missing-writer",
		"missing-upload", "late-upload", "duplicate-upload", "conditional", "continue", "overwrite", "missing-prefix"} {
		t.Run(mutation, func(t *testing.T) {
			doc := descriptorFixture(t)
			agent := doc.Jobs["agent"]
			mutateDescriptor(&agent, mutation)
			doc.Jobs["agent"] = agent
			assertArtifactRejection(t, doc)
		})
	}
}

func mutateDescriptor(j *job, mutation string) {
	switch mutation {
	case "body":
		j.Steps[1].Run = strings.Replace(j.Steps[1].Run, "identity: $identity", "identity: \"foreign\"", 1)
	case "attempt":
		j.Steps[1].Env["PRODUCER_ATTEMPT"] = "${{ needs.agent.outputs.attempt }}"
	case "identity":
		j.Steps[1].Env["INVOCATION_ID"] = "${{ inputs.identity }}"
	case "producer":
		j.Steps[1].Env["PRODUCER_JOB"] = "other"
	case "path":
		j.Steps[2].With["path"] = "other/producer.json"
	case "missing-writer":
		j.Steps = append(j.Steps[:1], j.Steps[2:]...)
	case "missing-upload":
		j.Steps = append(j.Steps[:2], j.Steps[3:]...)
	case "late-upload":
		j.Steps[2], j.Steps[4] = j.Steps[4], j.Steps[2]
	case "duplicate-upload":
		j.Steps = append(j.Steps, j.Steps[2])
	case "conditional":
		j.Steps[1].If = "false"
	case "continue":
		j.Steps[1].ContinueOnError = true
	case "overwrite":
		delete(j.Steps[2].With, "overwrite")
	case "missing-prefix":
		j.Steps = j.Steps[1:]
	}
}

func assertArtifactRejection(t *testing.T, doc document) {
	t.Helper()
	if got := strings.Join(verify(doc, "daily"), "\n"); !strings.Contains(got, "ArtifactProvenance") {
		t.Fatalf("tampered descriptor protocol accepted: %s", got)
	}
}

func TestProducerResolverMutations(t *testing.T) {
	for _, mutation := range []string{"future-attempt", "retained-failure", "foreign-producer", "recompute",
		"source-prefix", "inputs", "identity", "result", "missing", "duplicate", "late", "metadata-missing",
		"metadata-duplicate", "metadata-order", "metadata-origin", "metadata-path", "metadata-guard",
		"resolver-guard", "unguarded-payload", "guard-bypass", "missing-needs", "wrong-source"} {
		t.Run(mutation, func(t *testing.T) {
			doc := descriptorFixture(t)
			consumer := doc.Jobs["safe_outputs"]
			mutateResolver(&consumer, mutation)
			doc.Jobs["safe_outputs"] = consumer
			assertArtifactRejection(t, doc)
		})
	}
}

func mutateResolver(j *job, mutation string) {
	s := &j.Steps[2]
	switch mutation {
	case "future-attempt":
		s.Run = strings.Replace(s.Run, "and .attempt <= $current", "", 1)
	case "retained-failure":
		s.Run = strings.Replace(s.Run, `[[ "$PRODUCER_RESULT" != "success" ]]`, "false", 1)
	case "foreign-producer":
		s.Run = strings.Replace(s.Run, ".producer == $producer", "true", 1)
	case "recompute":
		s.Run = strings.Replace(s.Run, `export GITHUB_RUN_ATTEMPT="$ATTEMPT"`, "export GITHUB_RUN_ATTEMPT=1", 1)
	case "source-prefix":
		s.Run = strings.Replace(s.Run, "sha256sum", "sha1sum", 1)
	case "inputs", "identity", "result":
		key := map[string]string{"inputs": "INPUTS_JSON", "identity": "INVOCATION_ID", "result": "PRODUCER_RESULT"}[mutation]
		s.Env[key] = "${{ inputs.attacker }}"
	case "missing":
		j.Steps = append(j.Steps[:2], j.Steps[3:]...)
	case "duplicate":
		j.Steps = append(j.Steps, *s)
	case "late":
		j.Steps[2], j.Steps[3] = j.Steps[3], j.Steps[2]
	case "metadata-missing":
		j.Steps = append(j.Steps[:1], j.Steps[2:]...)
	case "metadata-duplicate":
		j.Steps = append(j.Steps, j.Steps[1])
	case "metadata-order":
		j.Steps[1], j.Steps[2] = j.Steps[2], j.Steps[1]
	case "metadata-origin":
		j.Steps[1].With["run-id"] = "other"
	case "metadata-path":
		j.Steps[1].With["path"] = "other"
	case "metadata-guard":
		j.Steps[1].If = "false"
	case "resolver-guard":
		s.If = "always()"
	case "unguarded-payload":
		j.Steps[3].If = ""
	case "guard-bypass":
		j.Steps[3].If += " || always()"
	case "missing-needs":
		j.Needs = yaml.Node{Kind: yaml.ScalarNode, Value: "detection"}
	case "wrong-source":
		j.Steps[3].With["pattern"] = "{" + sourcePrefix("other") + "agent," + sourcePrefix("other") + "agent-output-fallback}"
	}
}

func TestBoundArtifactProducerUniqueness(t *testing.T) {
	for _, mode := range []string{"payload", "fallback", "metadata", "legacy-payload"} {
		doc := descriptorFixture(t)
		producer := doc.Jobs["agent"]
		index := 4
		switch mode {
		case "metadata":
			index = 2
		case "fallback":
			index = 5
		}
		foreign := job{Steps: []step{prefixFixture(true), producer.Steps[index]}}
		if mode == "legacy-payload" {
			doc.Jobs["activation"] = job{Outputs: map[string]string{"artifact_prefix": localInvocationPrefix},
				Steps: []step{prefixFixture(true)}}
			foreign.Needs = yaml.Node{Kind: yaml.ScalarNode, Value: "activation"}
			foreign.Steps[1].With = map[string]any{"name": invocationPrefix + "agent", "path": "request.json"}
		}
		doc.Jobs["foreign"] = foreign
		assertArtifactRejection(t, doc)
	}
}

func TestBoundArtifactForeignOrigin(t *testing.T) {
	for _, key := range []string{"run-id", "repository"} {
		doc := descriptorFixture(t)
		extra := doc.Jobs["safe_outputs"]
		extra.Steps = append([]step(nil), extra.Steps...)
		extra.Steps[3].With = map[string]any{"name": sourcePrefix("agent") + "agent", key: "foreign"}
		doc.Jobs["additional_consumer"] = extra
		assertArtifactRejection(t, doc)
	}
}

func TestOptionalArtifactChannels(t *testing.T) {
	for _, bound := range []bool{false, true} {
		doc := localArtifactFixture(t, true)
		if bound {
			doc = descriptorFixture(t)
		}
		agent := doc.Jobs["agent"]
		agent.Steps = agent.Steps[:len(agent.Steps)-1]
		doc.Jobs["agent"] = agent
		consumer := doc.Jobs["safe_outputs"]
		consumer.Steps = append(consumer.Steps, step{
			Uses:            "actions/download-artifact@0000000000000000000000000000000000000000",
			If:              "false",
			ContinueOnError: true,
			With:            map[string]any{"name": localInvocationPrefix + "safe-outputs-items"},
		})
		doc.Jobs["safe_outputs"] = consumer
		if got := checkArtifacts(doc); len(got) != 0 {
			t.Fatalf("safe absent optional channel (bound=%v) rejected: %v", bound, got)
		}
	}
}

func TestMissingBoundFallbackDoesNotTrustForeignProducer(t *testing.T) {
	doc := descriptorFixture(t)
	agent := doc.Jobs["agent"]
	fallback := agent.Steps[len(agent.Steps)-1]
	agent.Steps = agent.Steps[:len(agent.Steps)-1]
	doc.Jobs["agent"] = agent
	doc.Jobs["foreign"] = job{Steps: []step{prefixFixture(true), fallback}}
	assertArtifactRejection(t, doc)
}

func TestCompiledActivationSourceProof(t *testing.T) {
	compiled := compiledReusableArtifactFixture(t, false)
	for _, mutation := range []string{"resolver", "metadata", "dependency", "guard", "foreign-run"} {
		t.Run(mutation, func(t *testing.T) {
			var doc document
			if err := yaml.Unmarshal([]byte(compiled), &doc); err != nil {
				t.Fatal(err)
			}
			agent := doc.Jobs["agent"]
			mutateActivationSource(t, &agent, mutation)
			doc.Jobs["agent"] = agent
			if got := strings.Join(checkArtifacts(doc), "\n"); !strings.Contains(got, "ArtifactProvenance") {
				t.Fatalf("tampered activation-to-agent binding accepted: %s", got)
			}
		})
	}
}

func mutateActivationSource(t *testing.T, agent *job, mutation string) {
	t.Helper()
	resolver, resolverAt, resolverFound := uniqueStep(*agent, "artifact-source-activation")
	_, metadataAt, metadataFound := uniqueStep(*agent, "artifact-metadata-activation")
	if !resolverFound || !metadataFound {
		t.Fatal("fresh compiled agent must resolve activation metadata")
	}
	switch mutation {
	case "resolver":
		agent.Steps[resolverAt].Run = strings.Replace(resolver.Run, ".producer == $producer", "true", 1)
	case "metadata":
		agent.Steps[metadataAt].With["name"] = identityPrefix + "producer-other"
	case "dependency":
		agent.Needs = yaml.Node{Kind: yaml.ScalarNode, Value: "other"}
	case "guard", "foreign-run":
		found := false
		for index, s := range agent.Steps {
			if s.With["name"] != sourcePrefix("activation")+"activation" {
				continue
			}
			found = true
			if mutation == "guard" {
				agent.Steps[index].If = "always()"
			} else {
				agent.Steps[index].With["run-id"] = "foreign"
			}
		}
		if !found {
			t.Fatal("fresh compiled agent must consume activation artifact")
		}
	}
}
