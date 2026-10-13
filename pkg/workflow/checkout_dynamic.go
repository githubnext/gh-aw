package workflow

import (
	"fmt"
	"strconv"
	"strings"
)

func generateDynamicCheckoutSteps(entry *resolvedCheckout, index int, keepCredentials bool, pushToken string, getActionPin func(string) string) []string {
	env := buildDynamicCheckoutEnvironment(entry, index, keepCredentials, pushToken, getActionPin)
	prepare := buildDynamicCheckoutScriptStep("Validate runtime checkout repositories", "prepare", "await prepare({ core, github, getOctokit });", index, env, getActionPin)
	checkout := fmt.Sprintf("      - name: Checkout runtime repositories (%d)\n        uses: ./.gh-aw/checkouts/%d\n        env:\n%s", index, index, env)
	finish := buildDynamicCheckoutScriptStep("Finalize runtime checkouts", "finish", "await finish();", index, env, getActionPin)
	steps := []string{prepare, checkout}
	if entry.cleanCreds && !keepCredentials {
		steps = append(steps, generateCheckoutCredentialsCleanupStep())
	}
	return append(steps, finish)
}

func dynamicCheckoutToken(entry *resolvedCheckout, index int, keepCredentials bool, pushToken string) string {
	token := resolveCheckoutTokenExpression(entry, index, false)
	if keepCredentials && entry.safeOutputApp != nil {
		token = fmt.Sprintf("${{ steps.checkout-safe-output-app-token-%d.outputs.token }}", index)
		if entry.safeOutputApp.shouldIgnoreMissingKey() {
			fallback := resolveCheckoutTokenExpression(entry, index, true)
			if entry.githubApp == nil && entry.token == "" && pushToken != "" {
				fallback = pushToken
			}
			token = combineTokenExpressions(token, fallback)
		}
	}
	if token == "" {
		token = pushToken
	}
	if token == "" {
		token = resolveGitHubToken("")
	}
	return token
}

func buildDynamicCheckoutEnvironment(entry *resolvedCheckout, index int, keepCredentials bool, pushToken string, getActionPin func(string) string) string {
	depth := 1
	if entry.fetchDepth != nil {
		depth = *entry.fetchDepth
	}
	var env strings.Builder
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_CHECKOUT_INDEX", strconv.Itoa(index))
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_REPOSITORIES", entry.key.repository)
	// Expressions remain data in environment variables, never executable script text.
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_ALLOW_REPOS", strings.Join(entry.allowRepos, "\n"))
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_PREFIX", entry.key.path)
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_REF", entry.ref)
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_FETCH_DEPTH", strconv.Itoa(depth))
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_SPARSE", strings.Join(entry.sparsePatterns, "\n"))
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_SUBMODULES", entry.submodules)
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_LFS", strconv.FormatBool(entry.lfs))
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_FETCH_COUNT", strconv.Itoa(len(entry.fetchRefs)))
	for i, ref := range entry.fetchRefs {
		writeYAMLEnv(&env, "          ", fmt.Sprintf("GH_AW_DYNAMIC_FETCH_%d", i), ref)
	}
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_PERSIST", strconv.FormatBool(keepCredentials || entry.cleanCreds))
	checkoutAction, _, _ := strings.Cut(getActionPin("actions/checkout"), " #")
	writeYAMLEnv(&env, "          ", "GH_AW_DYNAMIC_CHECKOUT_ACTION", checkoutAction)
	env.WriteString("          GH_AW_DYNAMIC_TOKEN: " + dynamicCheckoutToken(entry, index, keepCredentials, pushToken) + "\n")
	return env.String()
}

func buildDynamicCheckoutScriptStep(name, function, invocation string, index int, env string, getActionPin func(string) string) string {
	return fmt.Sprintf("      - name: %s (%d)\n        uses: %s\n        env:\n%s        with:\n          script: |\n            const { %s } = require('${{ runner.temp }}/gh-aw/actions/checkout_dynamic.cjs');\n            %s\n",
		name, index, getActionPin("actions/github-script"), env, function, invocation)
}
