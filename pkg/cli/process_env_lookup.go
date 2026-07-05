package cli

import (
	"os"
	"sync"
)

type envLookupFunc func(string) (string, bool)

var (
	processEnvLookupMu sync.RWMutex
	processEnvLookup   envLookupFunc = os.LookupEnv
)

func lookupEnv(key string) string {
	processEnvLookupMu.RLock()
	defer processEnvLookupMu.RUnlock()
	// Intentionally ignore the existence flag to preserve os.Getenv semantics:
	// missing variables and explicitly empty variables are both treated as "".
	value, _ := processEnvLookup(key)
	return value
}
