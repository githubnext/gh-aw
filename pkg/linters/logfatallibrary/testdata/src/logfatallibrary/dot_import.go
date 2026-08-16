package logfatallibrary

import . "log"

// BadDotImportFatal calls dot-imported log.Fatal and should be flagged.
func BadDotImportFatal() {
	Fatal("boom") // want `log\.Fatal called in library package`
}
