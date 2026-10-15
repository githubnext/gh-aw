package conformance

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow"
	"gopkg.in/yaml.v3"
)

func TestInvocationArtifactProvenance(t *testing.T) {
	for _, mutation := range []string{"", "untrusted-prefix", "different-prefix", "cross-run", "skipped-prefix"} {
		t.Run(mutation, func(t *testing.T) {
			doc := fixtureDocument(t)
			activation := doc.Jobs["activation"]
			activation.Outputs = map[string]string{"artifact_prefix": "${{ steps.artifact-prefix.outputs.prefix }}"}
			activation.Steps = []step{prefixFixture(false)}
			if mutation == "untrusted-prefix" {
				activation.Outputs["artifact_prefix"] = "${{ inputs.artifact_prefix }}"
			}
			if mutation == "skipped-prefix" {
				activation.Steps[0].If = "false"
			}
			doc.Jobs["activation"] = activation
			agent := doc.Jobs["agent"]
			agent.Steps[1].With["name"] = invocationPrefix + "agent"
			agent.Steps = append(agent.Steps, fallbackUpload(invocationPrefix))
			doc.Jobs["agent"] = agent
			processor := doc.Jobs["safe_outputs"]
			processor.Needs = yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "activation"},
				{Kind: yaml.ScalarNode, Value: "agent"},
				{Kind: yaml.ScalarNode, Value: "detection"},
			}}
			delete(processor.Steps[0].With, "name")
			processor.Steps[0].With["pattern"] = "{" + invocationPrefix + "agent," + invocationPrefix + "agent-output-fallback}"
			if mutation == "different-prefix" {
				processor.Steps[0].With["pattern"] = "{other-agent,other-agent-output-fallback}"
			}
			if mutation == "cross-run" {
				processor.Steps[0].With["run-id"] = "other-run"
			}
			doc.Jobs["safe_outputs"] = processor
			got := strings.Join(verify(doc, "daily"), "\n")
			if mutation == "" && got != "" {
				t.Fatalf("trusted symbolic handoff rejected: %s", got)
			}
			if mutation != "" && !strings.Contains(got, "ArtifactProvenance") {
				t.Fatalf("provenance mutation %q accepted", mutation)
			}
		})
	}
}

func prefixFixture(embedded bool) step {
	s := step{ID: "artifact-prefix", Shell: "bash", Run: externalPrefixScript,
		Env: map[string]any{"INPUTS_JSON": "${{ toJSON(inputs) }}"}}
	if embedded {
		s.Run = embeddedPrefixScript()
	}
	return s
}

func fallbackUpload(prefix string) step {
	return step{Uses: "actions/upload-artifact@0000000000000000000000000000000000000000",
		With: map[string]any{"name": prefix + "agent-output-fallback"}}
}

func localArtifactFixture(t *testing.T, embedded bool) document {
	t.Helper()
	doc := fixtureDocument(t)
	agent := doc.Jobs["agent"]
	agent.Steps[1].With["name"] = localInvocationPrefix + "agent"
	agent.Steps = append([]step{prefixFixture(embedded)}, agent.Steps...)
	agent.Steps = append(agent.Steps, fallbackUpload(localInvocationPrefix))
	doc.Jobs["agent"] = agent
	consumer := doc.Jobs["safe_outputs"]
	delete(consumer.Steps[0].With, "name")
	consumer.Steps[0].With["pattern"] = "{" + localInvocationPrefix + "agent," + localInvocationPrefix + "agent-output-fallback}"
	consumer.Steps = append([]step{prefixFixture(embedded)}, consumer.Steps...)
	doc.Jobs["safe_outputs"] = consumer
	return doc
}

func TestLocalArtifactPrefixComputations(t *testing.T) {
	for _, embedded := range []bool{false, true} {
		doc := localArtifactFixture(t, embedded)
		if got := verify(doc, "daily"); len(got) != 0 {
			t.Fatalf("canonical local handoff (embedded=%v) rejected: %v", embedded, got)
		}
	}
	for _, jobName := range []string{"agent", "safe_outputs"} {
		for _, mutation := range []string{"hash", "attempt", "identity-salt", "inputs", "output", "extra-command",
			"missing", "duplicate", "late", "conditional", "continue", "shell", "binding", "override", "uses"} {
			t.Run(jobName+"/"+mutation, func(t *testing.T) {
				doc := localArtifactFixture(t, true)
				j := doc.Jobs[jobName]
				mutatePrefix(&j, mutation)
				doc.Jobs[jobName] = j
				if got := strings.Join(verify(doc, "daily"), "\n"); !strings.Contains(got, "ArtifactProvenance") {
					t.Fatalf("tampered %s prefix accepted: %s", jobName, got)
				}
			})
		}
	}
}

func mutatePrefix(j *job, mutation string) {
	s := &j.Steps[0]
	switch mutation {
	case "hash":
		s.Run = strings.Replace(s.Run, "sha256sum", "sha1sum", 1)
	case "attempt":
		s.Run = strings.Replace(s.Run, "${GITHUB_RUN_ATTEMPT:-1}", "1", 1)
	case "identity-salt":
		s.Run = strings.Replace(s.Run, "::attempt=0", "::attempt=1", 1)
	case "inputs":
		s.Run = strings.Replace(s.Run, "${INPUTS_JSON:-{}}", "{}", 1)
	case "output":
		s.Run = strings.Replace(s.Run, `echo "prefix=${PREFIX}-"`, `echo "prefix=attacker-"`, 1)
	case "extra-command":
		s.Run += "\necho prefix=attacker- >> \"$GITHUB_OUTPUT\""
	case "missing":
		j.Steps = j.Steps[1:]
	case "duplicate":
		j.Steps = append(j.Steps, *s)
	case "late":
		j.Steps = append(j.Steps[1:], *s)
	case "conditional":
		s.If = "false"
	case "continue":
		s.ContinueOnError = true
	case "shell":
		s.Shell = "python"
	case "binding":
		s.Env["INPUTS_JSON"] = "${{ inputs.prefix }}"
	case "override":
		s.Env["GITHUB_RUN_ATTEMPT"] = "1"
	case "uses":
		s.Uses = "example/untrusted@0000000000000000000000000000000000000000"
	}
}

func TestArtifactHandoffBindings(t *testing.T) {
	for _, mutation := range []string{"cross-run", "cross-repository", "different-prefix", "fallback-only",
		"missing-upload", "workflow-env", "job-env", "legacy-dependency", "producer-dependency"} {
		t.Run(mutation, func(t *testing.T) {
			doc := localArtifactFixture(t, true)
			consumer := doc.Jobs["safe_outputs"]
			agent := doc.Jobs["agent"]
			switch mutation {
			case "cross-run":
				consumer.Steps[1].With["run-id"] = "7"
			case "cross-repository":
				consumer.Steps[1].With["repository"] = "other/repo"
			case "different-prefix":
				consumer.Steps[1].With["pattern"] = "{other-agent,other-agent-output-fallback}"
			case "fallback-only":
				consumer.Steps[1].With["pattern"] = localInvocationPrefix + "agent-output-fallback"
			case "missing-upload":
				agent.Steps = agent.Steps[:2]
			case "workflow-env":
				doc.Env = map[string]any{"GITHUB_RUN_ATTEMPT": "1"}
			case "job-env":
				consumer.Env = map[string]any{"BASH_ENV": "untrusted.sh"}
			case "legacy-dependency":
				consumer.Steps[1].With["pattern"] = "{" + invocationPrefix + "agent," + invocationPrefix + "agent-output-fallback}"
			case "producer-dependency":
				consumer.Needs = yaml.Node{Kind: yaml.ScalarNode, Value: "detection"}
			}
			doc.Jobs["agent"], doc.Jobs["safe_outputs"] = agent, consumer
			if got := strings.Join(verify(doc, "daily"), "\n"); !strings.Contains(got, "ArtifactProvenance") {
				t.Fatalf("handoff mutation accepted: %s", got)
			}
		})
	}
}

func TestCompiledReusableArtifactProvenance(t *testing.T) {
	for _, mode := range []string{"dotcom", "ghes-v3"} {
		t.Run(mode, func(t *testing.T) {
			compiled := compiledReusableArtifactFixture(t, mode == "ghes-v3")
			if !strings.Contains(compiled, sourcePrefix("agent")) || !strings.Contains(compiled, "producer-agent") {
				t.Fatal("fixture must exercise producer-attempt metadata and bound downloads")
			}
			var doc document
			if err := yaml.Unmarshal([]byte(compiled), &doc); err != nil {
				t.Fatal(err)
			}
			if got := checkArtifacts(doc); len(got) != 0 {
				t.Fatalf("compiled reusable workflow provenance rejected: %v", got)
			}
		})
	}
}

func compiledReusableArtifactFixture(t *testing.T, ghes bool) string {
	t.Helper()
	const source = "reusable-provenance.md"
	const markdown = `---
on: workflow_call
engine: copilot
checkout: false
permissions:
  contents: read
  copilot-requests: none
safe-outputs:
  create-issue:
---
Report findings.
`
	compiler := workflow.NewCompiler(workflow.WithNoEmit(true), workflow.WithSkipValidation(true))
	compiler.SetGHESCompat(ghes)
	parsed, err := compiler.ParseWorkflowString(markdown, source)
	if err != nil {
		t.Fatal(err)
	}
	parsed.SafeOutputs = &workflow.SafeOutputsConfig{CreateIssues: &workflow.CreateIssuesConfig{}}
	compiled, err := compiler.CompileToYAML(parsed, source)
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}
