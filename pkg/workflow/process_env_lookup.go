package workflow

import "os"

func lookupProcessEnv(key string) string {
	// Intentionally ignore the existence flag to preserve os.Getenv semantics:
	// missing variables and explicitly empty variables are both treated as "".
	value, _ := os.LookupEnv(key)
	return value
}
