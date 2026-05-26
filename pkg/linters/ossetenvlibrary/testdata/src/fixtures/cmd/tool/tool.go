package main

import "os"

func main() {
	os.Setenv("KEY", "val")
	os.Unsetenv("KEY")
}
