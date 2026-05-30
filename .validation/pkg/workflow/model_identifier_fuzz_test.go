//go:build !integration

package workflow

import (
	"math"
	"strconv"
	"testing"
)

// FuzzParseModelIdentifier fuzz tests the MAF identifier parser (Section 4.1 of the spec).
//
// The fuzzer validates that:
//  1. The parser never panics on any input.
//  2. A successfully parsed identifier has a non-empty Base.
//  3. p.Raw always equals the original input on success.
//  4. Known-parameter validation and UnrecognizedParams never panic on the parsed output.
//
// To run the fuzzer:
//
//	go test -v -fuzz=FuzzParseModelIdentifier -fuzztime=30s ./pkg/workflow
func FuzzParseModelIdentifier(f *testing.F) {
	// Valid bare names.
	f.Add("sonnet")
	f.Add("agent")
	f.Add("gpt-5")
	f.Add("my_model")
	f.Add("model.v2")

	// Valid provider-scoped names.
	f.Add("copilot/gpt-5")
	f.Add("openai/o3")
	f.Add("anthropic/claude-opus-4.5")
	f.Add("google/gemini-pro")

	// Valid glob patterns.
	f.Add("copilot/*sonnet*")
	f.Add("copilot/*")
	f.Add("openai/gpt-*")

	// Valid identifiers with parameters.
	f.Add("opus?effort=high")
	f.Add("gpt-5?temperature=0.7")
	f.Add("openai/o3?effort=low&temperature=0.2")
	f.Add("sonnet?effort=medium")
	f.Add("sonnet?temperature=2.0")
	f.Add("sonnet?temperature=0.0")

	// Edge cases that the parser must handle without panicking.
	f.Add("")
	f.Add("a")
	f.Add("a/b")
	f.Add("a?b=c")

	// Inputs that should be rejected.
	f.Add(".hidden")
	f.Add("-model")
	f.Add("my model")
	f.Add("copilot/")
	f.Add("copilot-/model")
	f.Add("?effort=high")
	f.Add("opus?effort=")
	f.Add("opus?=value")
	f.Add("opus?effort")
	f.Add("my@model")
	f.Add("a:b")
	f.Add("a\x00b")
	f.Add("a\nb")

	f.Fuzz(func(t *testing.T, input string) {
		// Must never panic.
		p, err := ParseModelIdentifier(input)
		if err != nil {
			// Rejected input — nothing further to check.
			return
		}

		// A successful parse must preserve the original input.
		if p.Raw != input {
			t.Errorf("ParseModelIdentifier(%q): Raw=%q, want %q", input, p.Raw, input)
		}
		// Base must be non-empty on success.
		if p.Base == "" {
			t.Errorf("ParseModelIdentifier(%q): Base must not be empty on success", input)
		}

		// These helpers must not panic on the parsed output.
		_ = ValidateKnownParams(p.Params)
		_ = UnrecognizedParams(p.Params)
	})
}

// FuzzValidateTemperatureParam fuzz tests that ValidateTemperatureParam:
//  1. Never panics on any string input.
//  2. Rejects NaN and infinite values.
//  3. Accepts only finite values in [0.0, 2.0].
//
// To run:
//
//	go test -v -fuzz=FuzzValidateTemperatureParam -fuzztime=30s ./pkg/workflow
func FuzzValidateTemperatureParam(f *testing.F) {
	// Valid values.
	f.Add("0.0")
	f.Add("0.7")
	f.Add("1.0")
	f.Add("2.0")
	f.Add("0")
	f.Add("2")
	f.Add("1.5")
	// Boundary violations.
	f.Add("-0.1")
	f.Add("2.1")
	f.Add("3.0")
	// Special float strings accepted by strconv.ParseFloat but not by the spec.
	f.Add("NaN")
	f.Add("nan")
	f.Add("+Inf")
	f.Add("-Inf")
	f.Add("Inf")
	f.Add("inf")
	// Non-numeric strings.
	f.Add("")
	f.Add("abc")
	f.Add("1.0a")
	f.Add("1,0")

	f.Fuzz(func(t *testing.T, input string) {
		// Must never panic.
		err := ValidateTemperatureParam(input)

		if err == nil {
			// If accepted, the value must be a finite float in [0.0, 2.0].
			f64, parseErr := strconv.ParseFloat(input, 64)
			if parseErr != nil {
				t.Errorf("ValidateTemperatureParam(%q): accepted value that cannot be re-parsed as float64", input)
				return
			}
			if math.IsNaN(f64) || math.IsInf(f64, 0) {
				t.Errorf("ValidateTemperatureParam(%q): accepted non-finite value", input)
			}
			if f64 < 0.0 || f64 > 2.0 {
				t.Errorf("ValidateTemperatureParam(%q): accepted out-of-range value %v", input, f64)
			}
		}
	})
}
