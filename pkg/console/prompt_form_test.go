//go:build !integration && !js && !wasm

package console

import (
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/require"
)

func TestPromptWrappersReturnNonNilForms(t *testing.T) {
	var inputValue string
	require.NotNil(t, NewInputForm(huh.NewInput().Value(&inputValue)))

	var selectValue string
	require.NotNil(t, NewSelectForm(huh.NewSelect[string]().
		Options(huh.NewOption("Option", "option")).
		Value(&selectValue)))

	var multiValue []string
	require.NotNil(t, NewMultiSelectForm(huh.NewMultiSelect[string]().
		Options(huh.NewOption("Option", "option")).
		Value(&multiValue)))

	var textValue string
	require.NotNil(t, NewTextForm(huh.NewText().Value(&textValue)))

	var confirmValue bool
	require.NotNil(t, NewConfirmForm(huh.NewConfirm().Value(&confirmValue)))
}
