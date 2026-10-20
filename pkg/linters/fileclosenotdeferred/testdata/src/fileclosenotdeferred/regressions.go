package fileclosenotdeferred

import "os"

func parenthesizedAcquisition() {
	f, err := ((os.Open)("test.txt")) // want `file Close\(\) should be deferred`
	if err != nil {
		return
	}
	_ = ((f).Close)()
}

func parenthesizedDeferredCleanup() {
	f, err := (os.Open("test.txt"))
	if err != nil {
		return
	}
	defer ((f).Close)()
}

func closeFile(f *os.File) { _ = f.Close() }

// Helper cleanup is intentionally outside the direct-call analysis.
func helperCleanup() {
	f, err := os.Open("test.txt")
	if err != nil {
		return
	}
	closeFile(f)
}

func useFile(f *os.File) {}

func helperUseWithLocalDefer() {
	f, err := os.Open("test.txt")
	if err != nil {
		return
	}
	defer f.Close()
	useFile(f)
}
