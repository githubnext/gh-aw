package regexpdynamicpattern

import (
	"fmt"
	"regexp"
)

// not flagged: literal pattern at package level.
var packageLevelRegexp = regexp.MustCompile(`^[a-z]+$`)

const constPattern = `^const$`
const constSuffix = `$`

// not flagged: literal pattern.
func ValidateLiteral(input string) bool {
	re := regexp.MustCompile(`^[a-z]+$`)
	return re.MatchString(input)
}

// not flagged: const identifier pattern.
func ValidateConst(input string) bool {
	re := regexp.MustCompile(constPattern)
	return re.MatchString(input)
}

// not flagged: concatenation of constant-only expressions.
func ValidateConstConcat(input string) bool {
	re := regexp.MustCompile(`^const` + constSuffix)
	return re.MatchString(input)
}

// flagged: pattern built with fmt.Sprintf.
func ValidateSprintf(prefix, input string) (bool, error) {
	re, err := regexp.Compile(fmt.Sprintf("^%s$", prefix)) // want `regexp pattern is not a compile-time constant; dynamic patterns can panic at runtime or enable ReDoS if influenced by untrusted input`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// flagged: string concatenation with a variable.
func ValidateConcatVariable(suffix, input string) bool {
	re := regexp.MustCompile(`^prefix` + suffix) // want `regexp pattern is not a compile-time constant; dynamic patterns can panic at runtime or enable ReDoS if influenced by untrusted input`
	return re.MatchString(input)
}

// flagged: pattern passed through from a function parameter.
func ValidateDynamic(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern) // want `regexp pattern is not a compile-time constant; dynamic patterns can panic at runtime or enable ReDoS if influenced by untrusted input`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

func suppressedPreviousLine(pattern, input string) (bool, error) {
	//nolint:regexpdynamicpattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

func suppressedSameLine(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern) //nolint:regexpdynamicpattern
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}
