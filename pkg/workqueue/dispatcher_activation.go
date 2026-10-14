package workqueue

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
)

// PrepareDispatcherActivation binds a previously verified dispatcher route to a
// named snapshot ref. It never moves or deletes tags, including on collisions.
// Repository writers must not mutate the snapshot tag while it is in use.
func (b Branch) PrepareDispatcherActivation(ctx context.Context, route DispatcherRoute) (string, error) {
	b, err := b.withClient()
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(route.Repository, b.Remote) || !revisionPattern.MatchString(route.Revision) ||
		strings.Trim(route.Revision, "0") == "" {
		return "", queueError("dispatcher_invalid", "activation requires the verified repository and immutable revision")
	}
	b.Remote = route.Repository
	ref := "refs/tags/gh-aw/work-queue/" + route.Revision
	endpoint := "git/ref/" + strings.TrimPrefix(ref, "refs/")
	var snapshot struct {
		Ref    string `json:"ref"`
		Object struct {
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"object"`
	}
	err = b.request(ctx, http.MethodGet, endpoint, nil, &snapshot)
	if hasStatus(err, http.StatusNotFound) {
		err = b.request(ctx, http.MethodPost, "git/refs", map[string]string{
			"ref": ref, "sha": route.Revision,
		}, nil)
		if err != nil {
			var failure *api.HTTPError
			// Only a definitive concurrent creation permits readback, not a
			// permission denial, other validation failure, or ambiguous write.
			if !errors.As(err, &failure) || failure.StatusCode != http.StatusUnprocessableEntity ||
				failure.Message != "Reference already exists" {
				return "", err
			}
		}
		err = b.request(ctx, http.MethodGet, endpoint, nil, &snapshot)
	}
	if err != nil {
		return "", err
	}
	if snapshot.Ref != ref || snapshot.Object.Type != "commit" || snapshot.Object.SHA != route.Revision {
		return "", queueError("dispatcher_invalid", "activation snapshot tag does not match the verified commit; refusing to dispatch")
	}
	return ref, nil
}
