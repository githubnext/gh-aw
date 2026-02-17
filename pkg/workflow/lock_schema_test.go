//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestPrependLockSchemaVersionHeader(t *testing.T) {
	content := "name: test\non: push\n"
	withHeader := prependLockSchemaVersionHeader(content)

	expectedPrefix := "# gh-aw-lock-schema-version: 1\n"
	if !strings.HasPrefix(withHeader, expectedPrefix) {
		t.Fatalf("expected header prefix %q, got %q", expectedPrefix, withHeader)
	}

	// Header should not be duplicated when already present.
	withHeaderAgain := prependLockSchemaVersionHeader(withHeader)
	if strings.Count(withHeaderAgain, lockSchemaVersionCommentPrefix) != 1 {
		t.Fatalf("expected exactly one schema header, got: %q", withHeaderAgain)
	}
}

func TestValidateLockSchemaCompatibility(t *testing.T) {
	t.Run("accepts legacy unversioned lock", func(t *testing.T) {
		content := []byte("name: legacy\non: push\n")
		if err := ValidateLockSchemaCompatibility("legacy.lock.yml", content); err != nil {
			t.Fatalf("expected legacy lock to be readable, got error: %v", err)
		}
	})

	t.Run("accepts current versioned lock", func(t *testing.T) {
		content := []byte("# gh-aw-lock-schema-version: 1\nname: test\non: push\n")
		if err := ValidateLockSchemaCompatibility("current.lock.yml", content); err != nil {
			t.Fatalf("expected current lock to be readable, got error: %v", err)
		}
	})

	t.Run("rejects incompatible future version", func(t *testing.T) {
		content := []byte("# gh-aw-lock-schema-version: 999\nname: future\non: push\n")
		err := ValidateLockSchemaCompatibility("future.lock.yml", content)
		if err == nil {
			t.Fatal("expected incompatible schema error, got nil")
		}
		if !strings.Contains(err.Error(), "incompatible lock schema version 999") {
			t.Fatalf("expected incompatible version error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "gh aw compile future") {
			t.Fatalf("expected migration guidance in error, got: %v", err)
		}
	})

	t.Run("rejects malformed version header", func(t *testing.T) {
		content := []byte("# gh-aw-lock-schema-version: banana\nname: bad\non: push\n")
		err := ValidateLockSchemaCompatibility("bad.lock.yml", content)
		if err == nil {
			t.Fatal("expected malformed schema error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to read lock schema version") {
			t.Fatalf("expected parse error details, got: %v", err)
		}
	})
}
