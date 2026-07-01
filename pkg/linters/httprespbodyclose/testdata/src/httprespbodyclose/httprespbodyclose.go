package httprespbodyclose

import (
	"io"
	"net/http"
)

// BadManualClose calls resp.Body.Close() directly instead of deferring it.
func BadManualClose(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req) // want `HTTP response Body\.Close\(\) should be deferred immediately after receiving the response to prevent resource leaks`
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	return data, readErr
}

// GoodDeferClose uses defer resp.Body.Close() immediately after receiving — not flagged.
func GoodDeferClose(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GoodNoClose returns the response to the caller, which is responsible for closing — not flagged.
func GoodNoClose(client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}
