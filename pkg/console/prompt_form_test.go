//go:build !integration && !js && !wasm

package console

import (
	"errors"
	"fmt"
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

func TestIsCancelled(t *testing.T) {
	t.Run("returns true for huh.ErrUserAborted", func(t *testing.T) {
		require.True(t, IsCancelled(huh.ErrUserAborted))
	})

	t.Run("returns true for wrapped huh.ErrUserAborted", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", huh.ErrUserAborted)
		require.True(t, IsCancelled(wrapped))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		require.False(t, IsCancelled(errors.New("some other error")))
	})

	t.Run("returns false for nil", func(t *testing.T) {
		require.False(t, IsCancelled(nil))
	})
}
