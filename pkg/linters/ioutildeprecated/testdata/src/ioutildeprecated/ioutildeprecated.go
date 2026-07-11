package ioutildeprecated

import (
	"io/ioutil"
	"os"
)

func BadReadAll() {
	f, _ := os.Open("file.txt")
	defer f.Close()
	_, _ = ioutil.ReadAll(f) // want `ioutil\.ReadAll is deprecated`
}

func BadReadFile() {
	_, _ = ioutil.ReadFile("file.txt") // want `ioutil\.ReadFile is deprecated`
}

func BadWriteFile() {
	_ = ioutil.WriteFile("file.txt", []byte("hello"), 0644) // want `ioutil\.WriteFile is deprecated`
}

func BadTempFile() {
	_, _ = ioutil.TempFile("", "prefix") // want `ioutil\.TempFile is deprecated`
}

func BadTempDir() {
	_, _ = ioutil.TempDir("", "prefix") // want `ioutil\.TempDir is deprecated`
}

func BadReadDir() {
	_, _ = ioutil.ReadDir(".") // want `ioutil\.ReadDir is deprecated`
}

func GoodReadAll() {
	f, _ := os.Open("file.txt")
	defer f.Close()
	// Using io.ReadAll is fine
	_ = f
}
