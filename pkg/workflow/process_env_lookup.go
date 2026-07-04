package workflow

import (
	"os"
	"sync"
)

type envLookupFunc func(string) (string, bool)

var (
	processEnvLookupMu sync.RWMutex
	processEnvLookup   envLookupFunc = os.LookupEnv
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
	// Intentionally ignore the existence flag to preserve os.Getenv semantics:
	// missing variables and explicitly empty variables are both treated as "".
	value, _ := processEnvLookup(key)
	return value
}
