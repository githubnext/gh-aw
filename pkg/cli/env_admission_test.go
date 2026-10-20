//go:build !integration

package cli

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionManagedDefaults(t *testing.T) {
	file, err := defaultsParseFile("defaults.yml", []byte("default_admission_url: https://admission.example.org/api/admission\ndefault_admission_audience: audience\ndefault_admission_on_outage: block\n"))
	require.NoError(t, err)
	require.NoError(t, defaultsValidateFile(&file))
	for _, field := range []string{"url", "audience", "on_outage"} {
		index := slices.IndexFunc(defaultsBindings, func(binding defaultsBinding) bool { return binding.fieldName == "default_admission_"+field })
		require.NotEqual(t, -1, index)
		assert.NotNil(t, *defaultsBindings[index].get(&file))
	}
	for _, data := range []string{"{}", "default_admission_url: null\n", "default_admission_url: ''\ndefault_admission_audience: ''\ndefault_admission_on_outage: ''\n"} {
		file, err := defaultsParseFile("defaults.yml", []byte(data))
		require.NoError(t, err)
		require.NoError(t, defaultsValidateFile(&file))
		changes := defaultsBuildUpdateChanges(&file)
		index := slices.IndexFunc(changes, func(change defaultsUpdateChange) bool { return change.envName == "GH_AW_DEFAULT_ADMISSION_URL" })
		require.NotEqual(t, -1, index)
		assert.Equal(t, file.DefaultAdmissionURL == nil, changes[index].delete)
		assert.Empty(t, changes[index].value)
	}
}

func TestAdmissionDefaultsValidation(t *testing.T) {
	for _, endpoint := range []string{"http://admission.example.org/api/admission", "https://user@admission.example.org/api/admission", "https://admission.example.org/other", "https://admission.example.org/api/admission?", "https://admission.example.org/api/admission#", "${{ vars.ENDPOINT }}", " https://admission.example.org/api/admission", "https://admission.example.org/api/%61dmission"} {
		require.ErrorContains(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionURL: &endpoint}), "default_admission_url")
	}
	for _, endpoint := range []string{"https://admission.example.org:99999/api/admission", "https://[invalid]/api/admission", "https://[fe80::1%25eth0]/api/admission", "https://2001:db8::1/api/admission"} {
		require.ErrorContains(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionURL: &endpoint}), "default_admission_url")
	}
	for _, endpoint := range []string{"https://admission.example.org:8443/api/admission", "https://[2001:db8::1]/api/admission"} {
		require.NoError(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionURL: &endpoint}))
	}
	for _, audience := range []string{"bad\nvalue", "${{ vars.AUDIENCE }}"} {
		require.ErrorContains(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionAudience: &audience}), "default_admission_audience")
	}
	for _, policy := range []string{"", "ignore", "block"} {
		require.NoError(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionOnOutage: &policy}))
	}
	policy := "allow"
	require.ErrorContains(t, defaultsValidateFile(&defaultsFile{DefaultAdmissionOnOutage: &policy}), "default_admission_on_outage")
}
