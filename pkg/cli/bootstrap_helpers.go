package cli

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"

	"charm.land/huh/v2"
	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/tty"
)

// parseBootstrapNames splits newline-delimited gh-api output into a sorted,
// deduplicated slice of non-empty names.
func parseBootstrapNames(output []byte) []string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	sort.Strings(result)
	return result
}

// resolveBootstrapTextValue resolves a text configuration value from an environment
// variable, a terminal prompt, or a provided default. It validates the result
// against an optional allowlist (enum).
func resolveBootstrapTextValue(envName, title, description, defaultValue string, allowed []string, optional bool) (string, bool, error) {
	if envValue := strings.TrimSpace(os.Getenv(envName)); envValue != "" {
		if err := validateBootstrapEnumValue(envValue, allowed, optional); err != nil {
			return "", false, err
		}
		return envValue, true, nil
	}
	if !tty.IsStderrTerminal() {
		if defaultValue != "" {
			if err := validateBootstrapEnumValue(defaultValue, allowed, optional); err != nil {
				return "", false, err
			}
			return defaultValue, true, nil
		}
		if optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s is required; set environment variable %s or rerun interactively. Example: export %s='example-value'", title, envName, envName)
	}

	var value string
	input := huh.NewInput().Title(title).Description(description).Value(&value)
	if defaultValue != "" {
		input = input.Placeholder(defaultValue)
	}
	input = input.Validate(func(v string) error {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" && defaultValue != "" {
			trimmed = defaultValue
		}
		if trimmed == "" && optional {
			return nil
		}
		if trimmed == "" {
			return errors.New("value cannot be empty. Example: enter a non-empty value such as example-value")
		}
		return validateBootstrapEnumValue(trimmed, allowed, optional)
	})
	if err := console.NewInputForm(input).Run(); err != nil {
		return "", false, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	if value == "" && optional {
		return "", false, nil
	}
	return value, true, nil
}

// resolveBootstrapSecretValue resolves a secret value from an environment
// variable or an interactive terminal prompt.
func resolveBootstrapSecretValue(envName, title, description string, optional bool) (string, bool, error) {
	if envValue := strings.TrimRight(os.Getenv(envName), "\r\n"); envValue != "" {
		return envValue, true, nil
	}
	if !tty.IsStderrTerminal() {
		if optional {
			return "", false, nil
		}
		return "", false, fmt.Errorf("%s is required; set environment variable %s or rerun interactively. Example: export %s='example-secret'", title, envName, envName)
	}
	value, err := console.PromptSecretInput(title, description)
	if err != nil {
		return "", false, err
	}
	value = strings.TrimRight(value, "\r\n")
	if value == "" && optional {
		return "", false, nil
	}
	return value, true, nil
}

// validateBootstrapEnumValue returns an error when value is not in the allowed
// list. An empty allowed list means all values are permitted.
func validateBootstrapEnumValue(value string, allowed []string, optional bool) error {
	if value == "" && optional {
		return nil
	}
	if len(allowed) == 0 {
		return nil
	}
	if slices.Contains(allowed, value) {
		return nil
	}
	return fmt.Errorf("value must be one of: %s. Example: %s", strings.Join(allowed, ", "), allowed[0])
}

// parseBootstrapBool parses a human-friendly boolean string such as "yes",
// "true", "1", "no", "false", or "0".
func parseBootstrapBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errors.New("expected one of: 1, true, yes, on, 0, false, no, off. Example: GH_AW_BOOTSTRAP_NO_OPEN_BROWSER=true")
	}
}

func bootstrapRandomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "\"", "&quot;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}

func openBootstrapBrowser(url string) bool {
	commands := [][]string{{"gh", "browse", url}}
	switch runtime.GOOS {
	case "darwin":
		commands = append([][]string{{"open", url}}, commands...)
	case "windows":
		commands = append([][]string{{"cmd", "/c", "start", "", url}}, commands...)
	default:
		commands = append([][]string{{"xdg-open", url}}, commands...)
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if err := cmd.Start(); err == nil {
			return true
		}
	}
	return false
}

func netListener() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// bootstrapRepositoryVariableEnvName returns the environment variable name
// that can be used to pre-supply a repository variable value non-interactively.
func bootstrapRepositoryVariableEnvName(name string) string {
	return bootstrapInputEnvName("VAR", name)
}

// bootstrapRepositorySecretEnvName returns the environment variable name
// that can be used to pre-supply a repository secret value non-interactively.
func bootstrapRepositorySecretEnvName(name string) string {
	return bootstrapInputEnvName("SECRET", name)
}

// bootstrapInputEnvName builds the canonical GH_AW_BOOTSTRAP_<kind>_<NAME>
// environment variable name for the given kind (VAR or SECRET) and raw name.
func bootstrapInputEnvName(kind, name string) string {
	suffix := strings.ToUpper(strings.TrimSpace(name))
	if suffix == "" {
		suffix = "VALUE"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range suffix {
		switch {
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	suffix = strings.Trim(builder.String(), "_")
	if suffix == "" {
		suffix = "VALUE"
	}
	return "GH_AW_BOOTSTRAP_" + kind + "_" + suffix
}

// deriveBootstrapAppName derives a valid GitHub App name from a repository slug
// or an explicit override. It strips invalid characters and truncates to 34
// characters while preserving a meaningful suffix.
func deriveBootstrapAppName(repo, explicitName string) string {
	candidate := strings.TrimSpace(explicitName)
	if candidate == "" {
		candidate = repo
	}
	candidate = strings.ReplaceAll(candidate, "/", "-")
	clean := strings.Builder{}
	previousDash := false
	for _, ch := range candidate {
		allowed := ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9'
		if allowed {
			clean.WriteRune(ch)
			previousDash = false
			continue
		}
		if !previousDash {
			clean.WriteRune('-')
			previousDash = true
		}
	}
	result := strings.Trim(clean.String(), "-")
	if len(result) <= 34 {
		return result
	}
	suffix := strings.TrimLeft(result[len(result)-15:], "-")
	prefixLength := max(1, 34-len(suffix)-1)
	prefix := strings.TrimRight(result[:prefixLength], "-")
	return strings.Trim(prefix+"-"+suffix, "-")
}
