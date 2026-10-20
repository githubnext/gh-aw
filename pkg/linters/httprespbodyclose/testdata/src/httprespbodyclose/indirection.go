package httprespbodyclose

import "net/http"

func closeBody(resp *http.Response)   { _ = resp.Body.Close() }
func useResponse(resp *http.Response) {}

// Helpers do not provide interprocedural cleanup evidence in either direction.
func helperCleanup(client *http.Client, req *http.Request) {
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	closeBody(resp)
}

func deferredHelperCleanup(client *http.Client, req *http.Request) {
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer closeBody(resp)
}

func helperUseWithLocalDefer(client *http.Client, req *http.Request) {
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	useResponse(resp)
}

func helperDoesNotHideManualClose(client *http.Client, req *http.Request) {
	resp, err := client.Do(req) // want `HTTP response Body\.Close\(\) should be deferred`
	if err != nil {
		return
	}
	defer useResponse(resp)
	_ = resp.Body.Close()
}
