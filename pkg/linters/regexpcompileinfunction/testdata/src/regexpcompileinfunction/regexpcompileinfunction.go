package regexpcompileinfunction

import (
	"regexp"
)

// Package-level regexp compilation is allowed and recommended.
var packageLevelRegexp = regexp.MustCompile(`^[a-z]+$`)

// This is also valid at package level (though MustCompile is more common).
var (
	anotherPackageLevelRegexp, _ = regexp.Compile(`\d+`)
)

const constPattern = `^const$`

// flagged: regexp.MustCompile inside function body
func ProcessString(s string) bool {
	re := regexp.MustCompile(`^[a-z]+$`) // want `regexp compilation inside function should be moved to package-level variable`
	return re.MatchString(s)
}

// flagged: regexp.Compile inside function body
func ValidateInput(input string) (bool, error) {
	re, err := regexp.Compile(`\d+`) // want `regexp compilation inside function should be moved to package-level variable`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// flagged: regexp.MustCompile with constant identifier inside function body
func ValidateConst(input string) bool {
	re := regexp.MustCompile(constPattern) // want `regexp compilation inside function should be moved to package-level variable`
	return re.MatchString(input)
}

// flagged: regexp.MustCompile inside loop (even worse!)
func ProcessMultiple(items []string) []bool {
	results := make([]bool, len(items))
	for i, item := range items {
		re := regexp.MustCompile(`[0-9]`) // want `regexp compilation inside function should be moved to package-level variable`
		results[i] = re.MatchString(item)
	}
	return results
}

// flagged: regexp.MustCompile in function literal
func GetValidator() func(string) bool {
	return func(s string) bool {
		re := regexp.MustCompile(`^valid`) // want `regexp compilation inside function should be moved to package-level variable`
		return re.MatchString(s)
	}
}

// not flagged: using package-level regexp
func CheckString(s string) bool {
	return packageLevelRegexp.MatchString(s)
}

// not flagged: method using package-level regexp
type Validator struct{}

func (v *Validator) Validate(s string) bool {
	return packageLevelRegexp.MatchString(s)
}

// not flagged: dynamic pattern cannot be moved to package-level variable directly
func ValidateDynamic(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

func suppressedPreviousLine() bool {
	//nolint:regexpcompileinfunction
	re := regexp.MustCompile(`^suppressed$`)
	return re.MatchString("suppressed")
}

func suppressedSameLine() bool {
	re := regexp.MustCompile(`^suppressed$`) //nolint:regexpcompileinfunction
	return re.MatchString("suppressed")
}
