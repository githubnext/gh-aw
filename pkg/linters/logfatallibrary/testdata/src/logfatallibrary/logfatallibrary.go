package logfatallibrary

import "log"

// bad: log.Fatal in a pkg/ package.
func stopWithFatal() {
	log.Fatal("something went wrong") // want `log.Fatal called in library package`
}

// bad: log.Fatalf in a pkg/ package.
func stopWithFatalf(msg string) {
	log.Fatalf("error: %s", msg) // want `log.Fatalf called in library package`
}

// bad: log.Fatalln in a pkg/ package.
func stopWithFatalln() {
	log.Fatalln("fatal error") // want `log.Fatalln called in library package`
}

// ok: log.Print is not fatal.
func logSomething() {
	log.Print("just a message")
}

// ok: nolint suppression on previous line.
func suppressedPreviousLine() {
	//nolint:logfatallibrary
	log.Fatal("suppressed")
}

// ok: nolint suppression on same line.
func suppressedSameLine() {
	log.Fatal("suppressed") //nolint:logfatallibrary
}
