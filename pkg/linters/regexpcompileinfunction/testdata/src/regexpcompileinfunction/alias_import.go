package regexpcompileinfunction

import re "regexp"

// flagged: aliased import of regexp — alias must not cause a false negative.
func ProcessWithAlias(s string) bool {
	r := re.MustCompile(`^[a-z]+$`) // want `regexp compilation inside function should be moved to package-level variable`
	return r.MatchString(s)
}

func ValidateWithAlias(input string) (bool, error) {
	r, err := re.Compile(`\d+`) // want `regexp compilation inside function should be moved to package-level variable`
	if err != nil {
		return false, err
	}
	return r.MatchString(input), nil
}

// not flagged: aliased import at package level is fine.
var packageLevelAliasRegexp = re.MustCompile(`^[a-z]+$`)
