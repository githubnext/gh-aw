package taint

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeJavaScriptMaskCommands(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	secret := []Source{{Reference: "secrets.token", Path: "env.TOKEN"}}
	const escape = `function escape(value) {
		return String(value).replace(/%/g,"%25").replace(/\r/g,"%0D").replace(/\n/g,"%0A");
	}`
	tests := []struct {
		name     string
		source   string
		disclose bool
	}{
		{"stdout registration", "process.stdout.write(`::add-mask::${escape(process.env.TOKEN)}\\n`);", false},
		{"console registration", `console.log("::add-mask::" + escape(process.env.TOKEN));`, false},
		{"registration alias", "const message=`::add-mask::${escape(process.env.TOKEN)}\\r\\n`; process.stdout.write(message);", false},
		{"registration is not sanitizer", "process.stdout.write(`::add-mask::${escape(process.env.TOKEN)}\\n`); console.log(process.env.TOKEN);", true},
		{"escaped value remains tainted", `const escaped=escape(process.env.TOKEN); console.log("::add-mask::"+escaped); console.log(escaped);`, true},
		{"stderr is not registration", `console.error("::add-mask::"+escape(process.env.TOKEN));`, true},
		{"missing prefix", `console.log("prefix ::add-mask::"+escape(process.env.TOKEN));`, true},
		{"possible newline injection", `console.log("::add-mask::"+process.env.TOKEN);`, true},
		{"non-global replacement", `console.log("::add-mask::"+process.env.TOKEN.replace(/\r/,"%0D").replace(/\n/,"%0A"));`, true},
		{"partial replacement", `console.log("::add-mask::"+process.env.TOKEN.replace(/\n/g,"%0A"));`, true},
		{"replaceAll", `console.log("::add-mask::"+process.env.TOKEN.replaceAll("\r","%0D").replaceAll("\n","%0A"));`, false},
		{"combined regex", `console.log("::add-mask::"+process.env.TOKEN.replace(/[\r\n]/g,""));`, false},
		{"sticky regex may stop early", `console.log("::add-mask::"+process.env.TOKEN.replace(/[\r\n]/gy,""));`, true},
		{"replacement reinserts input", `console.log("::add-mask::"+process.env.TOKEN.replace(/\r/g,"$&").replace(/\n/g,"$&"));`, true},
		{"reintroduced newline", `console.log("::add-mask::"+escape(process.env.TOKEN).replaceAll("%","\n"));`, true},
		{"content after terminator", `console.log("::add-mask::"+escape(process.env.TOKEN)+"\n"+process.env.TOKEN);`, true},
		{"conditional unsafe payload", `const value=flag?escape(process.env.TOKEN):process.env.TOKEN; console.log("::add-mask::"+value);`, true},
		{"conditional noncommand", `const value=flag?"::add-mask::"+escape(process.env.TOKEN):escape(process.env.TOKEN); console.log(value);`, true},
		{"serialized payload", `console.log("::add-mask::"+JSON.stringify(process.env.TOKEN));`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "mask.cjs", Source: escape + tt.source, Env: map[string][]Source{"TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			assert.Empty(t, result.Gaps)
			assert.Equal(t, tt.disclose, len(result.Findings) != 0, "%+v", result)
			for _, finding := range result.Findings {
				assert.Equal(t, Log, finding.Site.Kind)
				assert.Equal(t, secret, finding.Sources)
			}
		})
	}
	result, err := AnalyzeJavaScript(JavaScriptInput{
		Name: "capture.cjs", Source: escape + `console.log("::add-mask::"+escape(process.env.TOKEN));`,
		Env: map[string][]Source{"TOKEN": secret}, CaptureStdout: true,
	}, SecretLeakRule(), *options)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
	assert.Empty(t, result.Gaps)
	assert.Equal(t, secret, result.Stdout, "captured stdout is data, not a runner command")
}

func TestNativeJavaScriptPublicMetadata(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	secret := []Source{{Reference: "secrets.token", Path: "env.TOKEN"}}
	tests := []struct {
		name     string
		source   string
		disclose bool
		gap      bool
	}{
		{"string length", `console.log(process.env.TOKEN.length);`, false, false},
		{"computed string length", `console.log(process.env.TOKEN["length"]);`, false, false},
		{"buffer length", `console.log(require("fs").readFileSync(0).length);`, false, false},
		{"array length", `console.log([process.env.TOKEN].length);`, false, false},
		{"split length", `console.log(process.env.TOKEN.split("\n").length);`, false, false},
		{"string index", `console.log(process.env.TOKEN.indexOf("\n"));`, false, false},
		{"array index", `console.log([process.env.TOKEN].indexOf(process.env.TOKEN));`, false, false},
		{"line and column", `const text=process.env.TOKEN; const before=text.slice(0,20); const line=before.split("\n").length; const column=20-(before.lastIndexOf("\n")+1)+1; console.log({line,column});`, false, false},
		{"mixed public and private context", `function context(text){return {line:text.split("\n").length,lineText:text.slice(0,20)}} console.log(context(process.env.TOKEN).line);`, false, false},
		{"private context member", `function context(text){return {line:text.split("\n").length,lineText:text.slice(0,20)}} console.log(context(process.env.TOKEN).lineText);`, true, false},
		{"object length", `console.log({length:process.env.TOKEN}.length);`, true, false},
		{"opaque JSON length", `console.log(JSON.parse(process.env.TOKEN).length);`, true, false},
		{"opaque JSON line", `console.log(JSON.parse(process.env.TOKEN).line);`, true, false},
		{"numeric secret", `console.log(Number(process.env.TOKEN));`, true, false},
		{"numeric secret character", `console.log(process.env.TOKEN.charCodeAt(0));`, true, false},
		{"numeric secret in object", `console.log({line:Number(process.env.TOKEN)}.line);`, true, false},
		{"numeric secret as string", `console.log(String(Number(process.env.TOKEN)));`, true, false},
		{"numeric secret assigned to array length", `const data=[]; data.length=Number(process.env.TOKEN); console.log(data.length);`, true, false},
		{"mixed length types", `const data=flag?String(process.env.TOKEN):{length:process.env.TOKEN}; console.log(data.length);`, true, false},
		{"reassigned length type", `let data=String(process.env.TOKEN); data={length:process.env.TOKEN}; console.log(data.length);`, true, false},
		{"custom index method", `const data={indexOf(){return process.env.TOKEN}}; console.log(data.indexOf("x"));`, true, false},
		{"unknown index method", `console.log(JSON.parse(process.env.TOKEN).indexOf("x"));`, false, true},
		{"unknown numeric transform", `console.log(unknown(process.env.TOKEN).length);`, false, true},
		{"string coercion is not invocation", `function f(){console.log(process.env.TOKEN)} const text=String(f); text();`, false, true},
		{"dynamic environment index", `console.log(process.env[key]);`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "metadata.cjs", Source: tt.source, Stdin: secret, Env: map[string][]Source{"TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			assert.Equal(t, tt.disclose, len(result.Findings) != 0, "%+v", result)
			assert.Equal(t, tt.gap, len(result.Gaps) != 0, "%+v", result.Gaps)
		})
	}
}

func TestNativeJavaScriptNetworkWitness(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	secret := []Source{{Reference: "github.token", Path: "env.GITHUB_TOKEN"}}
	for _, tt := range []struct {
		name, source string
		finding, gap bool
	}{
		{"authentication only", `fetch("https://api.github.com/repos/owner/repo/pulls/1",{headers:{Authorization:"Bearer "+process.env.GITHUB_TOKEN}});`, false, false},
		{"unresolved PR number", `const number=Number(process.env[unknown]); fetch("https://api.github.com/repos/owner/repo/pulls/"+number,{headers:{Authorization:"Bearer "+process.env.GITHUB_TOKEN}});`, false, true},
		{"explicit token in URL", `fetch("https://api.github.com/repos/owner/repo/pulls/"+process.env.GITHUB_TOKEN,{headers:{Authorization:"Bearer "+process.env.GITHUB_TOKEN}});`, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "witness.cjs", Source: tt.source, Env: map[string][]Source{"GITHUB_TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			assert.Equal(t, tt.finding, len(result.Findings) != 0)
			assert.Equal(t, tt.gap, len(result.Gaps) != 0)
			for _, finding := range result.Findings {
				assert.Equal(t, Network, finding.Site.Kind)
				assert.Equal(t, secret, finding.Sources)
			}

		})
	}
}

func TestNativeJavaScriptObjectRestPrecision(t *testing.T) {
	options := nativeJavaScriptOptions(t)
	secret := []Source{{Reference: "github.token", Path: "env.GH_TOKEN"}}
	for _, tt := range []struct {
		name, source string
		disclose     bool
	}{
		{"public timeout with secret env", `const options={env:{GH_TOKEN:process.env.GH_TOKEN},timeout:60000}; const {suppressLogs=false,...spawnOptions}=options; console.error(spawnOptions.timeout);`, false},
		{"secret timeout remains tainted", `const options={env:{GH_TOKEN:process.env.GH_TOKEN},timeout:process.env.GH_TOKEN}; const {suppressLogs=false,...spawnOptions}=options; console.error(spawnOptions.timeout);`, true},
		{"rest credential remains tainted", `const options={env:{GH_TOKEN:process.env.GH_TOKEN},timeout:60000}; const {timeout,...rest}=options; console.error(rest.env.GH_TOKEN);`, true},
		{"excluded credential is not rest", `const options={token:process.env.GH_TOKEN,timeout:60000}; const {token,...rest}=options; console.error(rest);`, false},
		{"opaque rest retains unknown source", `const options=JSON.parse(process.env.GH_TOKEN); const {token,...rest}=options; console.error(rest.timeout);`, true},
		{"computed exclusion", `const options={token:process.env.GH_TOKEN,timeout:60000}; const key="token"; const {[key]:token,...rest}=options; console.error(rest);`, false},
		{"rest is shallow copy", `const options={env:{}}; const {...rest}=options; rest.env.token=process.env.GH_TOKEN; console.error(options.env.token);`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeJavaScript(JavaScriptInput{
				Name: "rest.cjs", Source: tt.source, Env: map[string][]Source{"GH_TOKEN": secret},
			}, SecretLeakRule(), *options)
			require.NoError(t, err)
			assert.Empty(t, result.Gaps)
			assert.Equal(t, tt.disclose, len(result.Findings) != 0, "%+v", result)
		})
	}
}
