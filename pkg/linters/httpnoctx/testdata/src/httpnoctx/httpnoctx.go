package httpnoctx

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// BadClientGet calls (*http.Client).Get without context.
func BadClientGet(client *http.Client, rawURL string) (*http.Response, error) {
	return client.Get(rawURL) // want `\(\*http\.Client\)\.Get does not accept a context`
}

// BadClientHead calls (*http.Client).Head without context.
func BadClientHead(client *http.Client, rawURL string) (*http.Response, error) {
	return client.Head(rawURL) // want `\(\*http\.Client\)\.Head does not accept a context`
}

// BadClientPost calls (*http.Client).Post without context.
func BadClientPost(client *http.Client, rawURL string, body io.Reader) (*http.Response, error) {
	return client.Post(rawURL, "application/json", body) // want `\(\*http\.Client\)\.Post does not accept a context`
}

// BadClientPostForm calls (*http.Client).PostForm without context.
func BadClientPostForm(client *http.Client, rawURL string, data url.Values) (*http.Response, error) {
	return client.PostForm(rawURL, data) // want `\(\*http\.Client\)\.PostForm does not accept a context`
}

// BadPkgGet calls the package-level http.Get without context.
func BadPkgGet(rawURL string) (*http.Response, error) {
	return http.Get(rawURL) // want `http\.Get does not accept a context`
}

// BadPkgHead calls the package-level http.Head without context.
func BadPkgHead(rawURL string) (*http.Response, error) {
	return http.Head(rawURL) // want `http\.Head does not accept a context`
}

// BadPkgPost calls the package-level http.Post without context.
func BadPkgPost(rawURL string, body io.Reader) (*http.Response, error) {
	return http.Post(rawURL, "application/json", body) // want `http\.Post does not accept a context`
}

// BadPkgPostForm calls the package-level http.PostForm without context.
func BadPkgPostForm(rawURL string, data url.Values) (*http.Response, error) {
	return http.PostForm(rawURL, data) // want `http\.PostForm does not accept a context`
}

// GoodClientDo uses http.NewRequestWithContext + client.Do — not flagged.
func GoodClientDo(ctx context.Context, client *http.Client, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// GoodDefaultClientDo uses http.NewRequestWithContext + http.DefaultClient.Do — not flagged.
func GoodDefaultClientDo(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader("body"))
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
