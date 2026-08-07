package regexpdynamicpattern

import (
	"fmt"
	"regexp"
)

// not flagged: literal pattern at package level.
var PackageLevelRegexp = regexp.MustCompile(`^[a-z]+$`)

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

// not flagged: POSIX literal pattern.
func ValidatePOSIXLiteral(input string) (bool, error) {
	re, err := regexp.CompilePOSIX(`^[a-z]+$`)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// not flagged: POSIX const identifier pattern.
func ValidatePOSIXConst(input string) bool {
	re := regexp.MustCompilePOSIX(constPattern)
	return re.MatchString(input)
}

// flagged: pattern built with fmt.Sprintf.
func ValidateSprintf(prefix, input string) (bool, error) {
	re, err := regexp.Compile(fmt.Sprintf("^%s$", prefix)) // want `regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants or let untrusted input control pattern complexity/size`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// flagged: string concatenation with a variable.
func ValidateConcatVariable(suffix, input string) bool {
	re := regexp.MustCompile(`^prefix` + suffix) // want `regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants or let untrusted input control pattern complexity/size`
	return re.MatchString(input)
}

// flagged: pattern passed through from a function parameter.
func ValidateDynamic(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern) // want `regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants or let untrusted input control pattern complexity/size`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// flagged: POSIX pattern passed through from a function parameter.
func ValidatePOSIXDynamic(pattern, input string) (bool, error) {
	re, err := regexp.CompilePOSIX(pattern) // want `regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants or let untrusted input control pattern complexity/size`
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

// flagged: POSIX MustCompile pattern passed through from a function parameter.
func ValidateMustPOSIXDynamic(pattern, input string) bool {
	re := regexp.MustCompilePOSIX(pattern) // want `regexp pattern is not a compile-time constant; malformed dynamic patterns can panic in MustCompile variants or let untrusted input control pattern complexity/size`
	return re.MatchString(input)
}

func SuppressedPreviousLine(pattern, input string) (bool, error) {
	//nolint:regexpdynamicpattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}

func SuppressedSameLine(pattern, input string) (bool, error) {
	re, err := regexp.Compile(pattern) //nolint:regexpdynamicpattern
	if err != nil {
		return false, err
	}
	return re.MatchString(input), nil
}
