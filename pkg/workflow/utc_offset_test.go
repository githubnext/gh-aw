//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeUTCOffset_InvalidValueIncludesExample(t *testing.T) {
	_, err := NormalizeUTCOffset("bad-offset")
	require.Error(t, err)
	require.ErrorContains(t, err, "must be a numeric UTC offset")
	require.ErrorContains(t, err, "Example:")
}

func TestParseUTCOffsetLocation_MalformedValueIncludesExample(t *testing.T) {
	_, err := ParseUTCOffsetLocation("bad-offset")
	require.Error(t, err)
	require.ErrorContains(t, err, "must be a numeric UTC offset")
	require.ErrorContains(t, err, "Example:")
}
