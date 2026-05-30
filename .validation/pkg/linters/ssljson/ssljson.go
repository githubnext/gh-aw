// Package ssljson implements a Go analysis linter that validates
// .github/skills/*/ssl.json files against the SSL specification rules.
//
// The analyzer fires only when run against its own anchor package
// (github.com/github/gh-aw/pkg/linters/ssljson), ensuring it executes exactly
// once per golint-custom invocation without producing duplicate diagnostics
// across every package in ./cmd/... and ./pkg/...
package ssljson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
)

const anchorPkg = "github.com/github/gh-aw/pkg/linters/ssljson"

// Analyzer is the ssl-json analysis pass.
var Analyzer = &analysis.Analyzer{
	Name: "ssljson",
	Doc:  "validates .github/skills/*/ssl.json files against the SSL specification enum and graph rules",
	URL:  "https://github.com/github/gh-aw/tree/main/pkg/linters/ssljson",
	Run:  run,
}

// --- types ------------------------------------------------------------------

// SSLDoc is the top-level structure of an ssl.json file.
type SSLDoc struct {
	Scheduling SSLScheduling `json:"scheduling"`
	Scenes     []SSLScene    `json:"scenes"`
	LogicSteps []SSLStep     `json:"logic_steps"`
}

// SSLScheduling holds the scheduling layer of an SSL document.
type SSLScheduling struct {
	ID         string `json:"id"`
	EntryScene string `json:"entry_scene"`
}

// SSLScene is a macro-level execution stage.
type SSLScene struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	EntryLogicStep string         `json:"entry_logic_step"`
	NextSceneRules []SSLSceneRule `json:"next_scene_rules"`
}

// SSLSceneRule is a conditional transition rule for a scene.
type SSLSceneRule struct {
	Condition string `json:"condition"`
	Target    string `json:"target"`
}

// SSLStep is an atomic logic step within a scene.
type SSLStep struct {
	ID            string `json:"id"`
	SceneID       string `json:"scene_id"`
	ActionType    string `json:"action_type"`
	ResourceScope string `json:"resource_scope"`
	Next          string `json:"next"`
}

// --- restricted enumerations ------------------------------------------------

var allowedSceneTypes = map[string]bool{
	"PREPARE": true, "ACQUIRE": true, "REASON": true, "ACT": true,
	"VERIFY": true, "RECOVER": true, "FINALIZE": true,
}

var allowedActionTypes = map[string]bool{
	"READ": true, "SELECT": true, "COMPARE": true, "VALIDATE": true,
	"INFER": true, "WRITE": true, "UPDATE_STATE": true, "CALL_TOOL": true,
	"REQUEST": true, "TRANSFER": true, "NOTIFY": true, "TERMINATE": true,
}

var allowedResourceScopes = map[string]bool{
	"MEMORY": true, "LOCAL_FS": true, "CODEBASE": true, "PROCESS": true,
	"USER_DATA": true, "CREDENTIALS": true, "NETWORK": true, "OTHER": true,
}

var sceneTerminals = map[string]bool{"END_SUCCESS": true, "END_FAIL": true}
var stepTerminals = map[string]bool{"YIELD_SUCCESS": true, "YIELD_FAIL": true}

// --- validation logic -------------------------------------------------------

// ValidateDoc validates an SSLDoc against all SSL Pass-4 rules.
// It returns a slice of diagnostic messages; an empty slice means the document
// is valid.
func ValidateDoc(doc SSLDoc) []string {
	var msgs []string

	sceneIDs := make(map[string]bool, len(doc.Scenes))
	for _, s := range doc.Scenes {
		sceneIDs[s.ID] = true
	}
	stepIDs := make(map[string]bool, len(doc.LogicSteps))
	for _, ls := range doc.LogicSteps {
		stepIDs[ls.ID] = true
	}

	// Rule 1: entry_scene must reference an existing scene.
	if doc.Scheduling.EntryScene != "" && !sceneIDs[doc.Scheduling.EntryScene] {
		msgs = append(msgs, fmt.Sprintf("entry_scene %q not found in scenes", doc.Scheduling.EntryScene))
	}

	for _, scene := range doc.Scenes {
		// Rule 2: scene type must be an allowed value.
		if !allowedSceneTypes[scene.Type] {
			msgs = append(msgs, fmt.Sprintf("scene %q has invalid type %q", scene.ID, scene.Type))
		}
		// Rule 3: entry_logic_step must reference an existing logic step.
		if scene.EntryLogicStep != "" && !stepIDs[scene.EntryLogicStep] {
			msgs = append(msgs, fmt.Sprintf("scene %q entry_logic_step %q not found in logic_steps", scene.ID, scene.EntryLogicStep))
		}
		// Rule 4: scene transition targets must resolve to a scene ID or terminal.
		for _, rule := range scene.NextSceneRules {
			if !sceneIDs[rule.Target] && !sceneTerminals[rule.Target] {
				msgs = append(msgs, fmt.Sprintf(
					"scene %q transition target %q is not a scene ID or END_SUCCESS/END_FAIL",
					scene.ID, rule.Target,
				))
			}
		}
	}

	for _, step := range doc.LogicSteps {
		// Rule 5: action_type must be an allowed value.
		if !allowedActionTypes[step.ActionType] {
			msgs = append(msgs, fmt.Sprintf("logic step %q has invalid action_type %q", step.ID, step.ActionType))
		}
		// Rule 6: resource_scope must be an allowed value.
		if !allowedResourceScopes[step.ResourceScope] {
			msgs = append(msgs, fmt.Sprintf("logic step %q has invalid resource_scope %q", step.ID, step.ResourceScope))
		}
		// Rule 7: logic-step next must be a step ID or a terminal target.
		if !stepIDs[step.Next] && !stepTerminals[step.Next] {
			msgs = append(msgs, fmt.Sprintf(
				"logic step %q next %q is not a step ID or YIELD_SUCCESS/YIELD_FAIL",
				step.ID, step.Next,
			))
		}
	}

	return msgs
}

// LoadDoc reads and parses an ssl.json file from disk.
func LoadDoc(path string) (SSLDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SSLDoc{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var doc SSLDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return SSLDoc{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return doc, nil
}

// FindSSLFiles returns all ssl.json file paths found under root/.github/skills/.
func FindSSLFiles(root string) ([]string, error) {
	skillsDir := filepath.Join(root, ".github", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading skills directory: %w", err)
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(skillsDir, e.Name(), "ssl.json")
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// findRepoRoot walks up from a file path to find the directory containing .git.
func findRepoRoot(from string) (string, error) {
	dir := filepath.Dir(from)
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repository root from %s", from)
		}
		dir = parent
	}
}

// --- analyzer entry point ---------------------------------------------------

func run(pass *analysis.Pass) (any, error) {
	// Only run when analyzing the anchor package, so that diagnostics are
	// emitted exactly once per golint-custom invocation.
	if pass.Pkg.Path() != anchorPkg {
		return nil, nil
	}
	if len(pass.Files) == 0 {
		return nil, nil
	}

	// Use the package declaration of the first source file as the diagnostic
	// position anchor. go/analysis requires a valid token.Pos for Reportf, but
	// ssl.json violations are not associated with any Go AST node; attaching
	// them to the package declaration is the conventional workaround for
	// file-system linters.
	anchorPos := pass.Files[0].Package
	filename := pass.Fset.Position(anchorPos).Filename

	repoRoot, err := findRepoRoot(filename)
	if err != nil {
		return nil, err
	}

	sslFiles, err := FindSSLFiles(repoRoot)
	if err != nil {
		return nil, err
	}

	for _, f := range sslFiles {
		doc, err := LoadDoc(f)
		if err != nil {
			pass.Reportf(anchorPos, "ssljson: %v", err)
			continue
		}
		rel, _ := filepath.Rel(repoRoot, filepath.Dir(f))
		for _, msg := range ValidateDoc(doc) {
			pass.Reportf(anchorPos, "ssljson %s: %s", rel, msg)
		}
	}

	return nil, nil
}
