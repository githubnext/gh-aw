package githubapi

import (
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/github/gh-aw/pkg/constants"
)

// ClientOptions returns go-gh REST client options with the repository default timeout applied.
func ClientOptions(host, authToken string) api.ClientOptions {
	return api.ClientOptions{
		Host:      host,
		AuthToken: authToken,
		Timeout:   constants.DefaultHTTPClientTimeout,
	}
}
