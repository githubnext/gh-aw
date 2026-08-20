//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalIndentJSONOrWrap(t *testing.T) {
	t.Parallel()
	t.Run("marshals value as indented JSON", func(t *testing.T) {
		t.Parallel()
		data, err := marshalIndentJSONOrWrap(map[string]string{"a": "b"}, "test value")
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":"b"}`, string(data))
		assert.Contains(t, string(data), "\n  \"a\"", "output should be indented with two spaces")
	})

	t.Run("wraps error with context", func(t *testing.T) {
		t.Parallel()
		_, err := marshalIndentJSONOrWrap(make(chan int), "test value")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal test value to JSON")
	})
}
