package workflow

import (
	"os"
	"sync"
)

type processEnvLookupFunc func(string) (string, bool)

var (
	processEnvLookupMu sync.RWMutex
	processEnvLookup   processEnvLookupFunc = os.LookupEnv
)

// SetProcessEnvLookup configures how workflow helpers resolve environment values.
// Passing nil restores the default process environment lookup.
func SetProcessEnvLookup(lookup func(string) (string, bool)) {
	processEnvLookupMu.Lock()
	defer processEnvLookupMu.Unlock()
	if lookup == nil {
		processEnvLookup = os.LookupEnv
		return
	}
	processEnvLookup = lookup
}

func lookupProcessEnv(key string) string {
	processEnvLookupMu.RLock()
	defer processEnvLookupMu.RUnlock()
	lookup := processEnvLookup
	if lookup == nil {
		return ""
	}
	value, _ := lookup(key)
	return value
}
