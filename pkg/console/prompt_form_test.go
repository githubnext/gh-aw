//go:build !integration && !js && !wasm

package console

import (
	"testing"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/require"
)

func TestPromptWrappersReturnForms(t *testing.T) {
	var inputValue string
	require.NotNil(t, PromptInput(huh.NewInput().Value(&inputValue)))

	var selectValue string
	require.NotNil(t, PromptSelect(huh.NewSelect[string]().
		Options(huh.NewOption("Option", "option")).
		Value(&selectValue)))

	var multiValue []string
	require.NotNil(t, PromptMultiSelect(huh.NewMultiSelect[string]().
		Options(huh.NewOption("Option", "option")).
		Value(&multiValue)))

	var textValue string
	require.NotNil(t, PromptText(huh.NewText().Value(&textValue)))

	var confirmValue bool
	require.NotNil(t, PromptConfirm(huh.NewConfirm().Value(&confirmValue)))
}
