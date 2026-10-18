package parser

import (
	"errors"
	"fmt"
	"strings"
)

const BuiltinToolProfilePathPrefix = BuiltinPathPrefix + "tool-profiles/"

// importsWithToolProfile appends the embedded ecosystem defaults after explicit
// imports, preserving the normal first-wins import precedence.
func importsWithToolProfile(frontmatter map[string]any) ([]ImportSpec, error) {
	var specs []ImportSpec
	if imports, exists := frontmatter["imports"]; exists {
		var err error
		specs, err = parseImportSpecsFromField(imports)
		if err != nil {
			return nil, err
		}
	}
	tools, ok := frontmatter["tools"].(map[string]any)
	if !ok {
		return specs, nil
	}
	value, exists := tools["profile"]
	if !exists {
		return specs, nil
	}
	name, ok := value.(string)
	if !ok || name == "" || strings.ContainsAny(name, `/\.:`) {
		return nil, errors.New("tools.profile must name a supported ecosystem, for example 'go'")
	}
	profilePath := BuiltinToolProfilePathPrefix + name + ".md"
	if !BuiltinVirtualFileExists(profilePath) {
		return nil, fmt.Errorf("tools.profile: unknown ecosystem profile %q", name)
	}
	return append(specs, ImportSpec{Path: profilePath}), nil
}
