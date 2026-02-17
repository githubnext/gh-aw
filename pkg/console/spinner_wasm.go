//go:build js || wasm

package console

import (
	"fmt"
	"os"
)

type SpinnerWrapper struct {
	enabled bool
}

func NewSpinner(message string) *SpinnerWrapper {
	return &SpinnerWrapper{enabled: false}
}

func (s *SpinnerWrapper) Start() {}
func (s *SpinnerWrapper) Stop()  {}
func (s *SpinnerWrapper) StopWithMessage(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
}
func (s *SpinnerWrapper) UpdateMessage(message string) {}
func (s *SpinnerWrapper) IsEnabled() bool              { return false }
