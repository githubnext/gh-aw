package rawloginlib

import "log"

func bad() {
	log.Printf("hello %s", "world") // want `log\.Printf called in library package`
	log.Println("oops")             // want `log\.Println called in library package`
}

func good() {
	// Using pkg/logger is fine — this file only tests that raw log calls are flagged.
	_ = "no raw log call here"
}
