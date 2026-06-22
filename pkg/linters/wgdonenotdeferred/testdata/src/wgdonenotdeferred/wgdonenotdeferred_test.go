package wgdonenotdeferred

import (
	"sync"
	"testing"
)

func TestSkippedTestFile(t *testing.T) {
	t.Helper()

	var wg sync.WaitGroup
	go func() {
		wg.Done()
	}()
}
