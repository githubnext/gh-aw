package logfatallibrary

import applog "log"

func badAlias() {
	applog.Fatal("boom") // want `log\.Fatal called in library package`
}
