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

	var confirmValue bool
	require.NotNil(t, NewConfirmForm(huh.NewConfirm().Value(&confirmValue)))
}
