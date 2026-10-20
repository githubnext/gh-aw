package taint

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJavaScriptSummaryRuntimeGlobals(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		env          []string
		leak         bool
	}{
		{
			name: "process parameter does not shadow runtime credentials",
			source: `const {getOTLPSecretValues} = require("./actions/setup/js/otlp_header_helpers.cjs");
				function read(process) {console.error(getOTLPSecretValues())}
				read({env: {OTEL_EXPORTER_OTLP_HEADERS: "public"}});`,
			env:  []string{"OTEL_EXPORTER_OTLP_HEADERS"},
			leak: true,
		},
		{
			name: "local process does not shadow runtime credentials",
			source: `const {getOTLPRunnerMaskValues} = require("./actions/setup/js/otlp_header_helpers.cjs");
				function read() {
					const process = {env: {OTEL_EXPORTER_OTLP_HEADERS: "public"}};
					console.error(getOTLPRunnerMaskValues());
				}
				read();`,
			env:  []string{"OTEL_EXPORTER_OTLP_HEADERS"},
			leak: true,
		},
		{
			name: "process parameter does not shadow runtime server",
			source: `const {buildGitAuthEnv} = require("./actions/setup/js/git_auth_env.cjs");
				function read(process) {console.error(buildGitAuthEnv("public").GIT_CONFIG_KEY_0)}
				read({env: {GITHUB_SERVER_URL: "https://public.example"}});`,
			env:  []string{"GITHUB_SERVER_URL"},
			leak: true,
		},
		{
			name: "global parameter does not shadow runtime masking callback",
			source: `const {maskSecret} = require("./actions/setup/js/actions_secret_masking.cjs");
				global.core = {setSecret: token => console.error(token)};
				function mask(global) {maskSecret(process.env.TOKEN)}
				mask({core: {setSecret: () => {}}});`,
			env:  []string{"TOKEN"},
			leak: true,
		},
		{
			name: "global parameter does not shadow runtime warning callback",
			source: `const {getOTLPSecretValues} = require("./actions/setup/js/otlp_header_helpers.cjs");
				global.core = {warning: () => console.error(process.env.TOKEN)};
				function read(global) {getOTLPSecretValues()}
				read({core: {warning: () => {}}});`,
			env:  []string{"TOKEN"},
			leak: true,
		},
		{
			name: "local global object is not a runtime mutation",
			source: `const {maskSecret} = require("./actions/setup/js/actions_secret_masking.cjs");
				function mask(global) {
					global.core = {setSecret: token => console.error(token)};
					maskSecret(process.env.TOKEN);
				}
				mask({});`,
			env: []string{"TOKEN"},
		},
		{
			name: "local core binding is not a runtime callback",
			source: `const {maskSecret} = require("./actions/setup/js/actions_secret_masking.cjs");
				function mask(core) {maskSecret(process.env.TOKEN)}
				mask({setSecret: token => console.error(token)});`,
			env: []string{"TOKEN"},
		},
		{
			name: "module local global binding is not a runtime mutation",
			source: `const {maskSecret} = require("./actions/setup/js/actions_secret_masking.cjs");
				const global = {core: {setSecret: token => console.error(token)}};
				maskSecret(process.env.TOKEN);`,
			env: []string{"TOKEN"},
		},
		{
			name: "module local process binding does not shadow runtime credentials",
			source: `const {getOTLPSecretValues} = require("./actions/setup/js/otlp_header_helpers.cjs");
				const process = {env: {OTEL_EXPORTER_OTLP_HEADERS: "public"}};
				console.error(getOTLPSecretValues());`,
			env:  []string{"OTEL_EXPORTER_OTLP_HEADERS"},
			leak: true,
		},
		{
			name: "runtime environment mutation survives process shadowing",
			source: `const {getOTLPSecretValues} = require("./actions/setup/js/otlp_header_helpers.cjs");
				process.env.GH_AW_OTLP_ALL_HEADERS = process.env.TOKEN;
				function read(process) {console.error(getOTLPSecretValues())}
				read({env: {GH_AW_OTLP_ALL_HEADERS: "public"}});`,
			env:  []string{"TOKEN"},
			leak: true,
		},
		{
			name: "module runtime mutations survive lexical shadowing",
			source: `const {getOTLPSecretValues, maskSecret} = require("./actions/setup/js/actions_secret_masking.cjs");
				require("./configure.cjs");
				function read(process, global) {
					console.error(getOTLPSecretValues());
					maskSecret("public");
				}
				read({env: {GH_AW_OTLP_ALL_HEADERS: "public"}}, {core: {setSecret: () => {}}});`,
			env:  []string{"TOKEN"},
			leak: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			options := summaryTestOptions(t)
			options.Files.(fstest.MapFS)["configure.cjs"] = &fstest.MapFile{
				Data: []byte(`process.env.GH_AW_OTLP_ALL_HEADERS = process.env.TOKEN;
					global.core = {setSecret: () => console.error(process.env.TOKEN)};`),
			}
			env := make(map[string][]Source)
			for _, name := range tt.env {
				env[name] = []Source{{Reference: "secrets.token", Path: "env." + name}}
			}
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "entry.cjs", Source: tt.source, Env: env,
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			assert.Empty(t, result.Gaps)
			assert.Equal(t, tt.leak, len(result.Findings) > 0, "%+v", result)
		})
	}
}

func TestJavaScriptSummaryCyclicInput(t *testing.T) {
	options := summaryTestOptions(t)
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name: "entry.cjs",
		Source: `const error = {message: "public"}; error.self = error;
			require("./actions/setup/js/error_helpers.cjs").getErrorMessage(error);`,
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	assert.Empty(t, result.Gaps)
	assert.Empty(t, result.Findings)
}
