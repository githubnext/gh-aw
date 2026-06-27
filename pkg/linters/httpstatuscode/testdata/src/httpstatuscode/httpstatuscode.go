package httpstatuscode

import "net/http"

func compareStatus(status int) {
	if status == 200 { // want `use http\.StatusOK instead of magic HTTP status code 200`
	}
	if status == 206 { // want `use http\.StatusPartialContent instead of magic HTTP status code 206`
	}
	if status == 304 { // want `use http\.StatusNotModified instead of magic HTTP status code 304`
	}
	if status == 403 { // want `use http\.StatusForbidden instead of magic HTTP status code 403`
	}
	if status == 404 { // want `use http\.StatusNotFound instead of magic HTTP status code 404`
	}
	if status == 407 { // want `use http\.StatusProxyAuthRequired instead of magic HTTP status code 407`
	}
	if status == 599 { // want `use http\.Status\* constant instead of magic HTTP status code 599`
	}
}

func compareStatusCode(statusCode int) {
	if statusCode != 200 { // want `use http\.StatusOK instead of magic HTTP status code 200`
	}
	if statusCode == 206 { // want `use http\.StatusPartialContent instead of magic HTTP status code 206`
	}
	if statusCode == 304 { // want `use http\.StatusNotModified instead of magic HTTP status code 304`
	}
	if statusCode == 403 { // want `use http\.StatusForbidden instead of magic HTTP status code 403`
	}
	if statusCode == 407 { // want `use http\.StatusProxyAuthRequired instead of magic HTTP status code 407`
	}
}

func compareResponse(resp *http.Response) {
	if resp.StatusCode == 200 { // want `use http\.StatusOK instead of magic HTTP status code 200`
	}
	if resp.StatusCode == 404 { // want `use http\.StatusNotFound instead of magic HTTP status code 404`
	}
}

func compareNamedConstants(status int, resp *http.Response) {
	if status == http.StatusOK {
	}
	if resp.StatusCode == http.StatusNotFound {
	}
}

func compareNonHTTP(status int, buildNumber int) {
	if status == 0 {
	}
	if buildNumber == 200 {
	}
}

func compareReversed(status int) {
	if 200 == status { // want `use http\.StatusOK instead of magic HTTP status code 200`
	}
}

func compareNoLint(status int) {
	if status == 200 { //nolint:httpstatuscode
	}
}
