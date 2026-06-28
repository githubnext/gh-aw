package main

import "os"

func main() {
	// os.Getenv and os.LookupEnv are allowed in main packages.
	_ = os.Getenv("KEY")
	_, _ = os.LookupEnv("KEY")
}
