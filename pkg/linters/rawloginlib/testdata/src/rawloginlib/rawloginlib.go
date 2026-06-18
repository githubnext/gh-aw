package rawloginlib

import "log"

type customLogger struct{}

func (*customLogger) Printf(string, ...any) {}

func bad() {
	log.Printf("hello %s", "world") // want `log\.Printf called in library package`
	log.Println("oops")             // want `log\.Println called in library package`
}

func good() {
	// Using pkg/logger is fine — this file only tests that raw log calls are flagged.
	_ = "no raw log call here"
}

func suppressed() {
	//nolint:rawloginlib
	log.Printf("suppressed previous line")
	log.Println("suppressed same line") //nolint:rawloginlib
}

func shadowedParam(log *customLogger) {
	log.Printf("shadowed parameter should not trigger")
}

func shadowedLocal() {
	log := &customLogger{}
	log.Printf("shadowed local should not trigger")
}
