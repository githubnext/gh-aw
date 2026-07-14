package rawloginlib

import applog "log"

func badAlias() {
	applog.Printf("boom") // want `log\.Printf called in library package`
}
