package rawloginlib

import . "log"

// BadDotImportPrintln calls dot-imported log.Println and should be flagged.
func BadDotImportPrintln() {
	Println("boom") // want `log\.Println called in library package`
}
