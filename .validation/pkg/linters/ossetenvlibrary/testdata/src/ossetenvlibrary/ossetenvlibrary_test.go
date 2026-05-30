package ossetenvlibrary

import "os"

func helperInTestFile() {
	os.Setenv("KEY", "val")
	os.Unsetenv("KEY")
}
