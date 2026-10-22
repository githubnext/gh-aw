package workqueue

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/cli/go-gh/v2/pkg/api"
)

func definitiveIssueLabelRejection(err error) bool {
	if hasStatus(err, http.StatusBadRequest) || hasStatus(err, http.StatusUnauthorized) ||
		hasStatus(err, http.StatusForbidden) || hasStatus(err, http.StatusUnprocessableEntity) {
		return true
	}
	var response *api.GraphQLError
	if !errors.As(err, &response) || len(response.Errors) == 0 {
		return false
	}
	for _, item := range response.Errors {
		if !slices.Equal(item.Path, []any{"createLabel"}) {
			return false
		}
		switch item.Type {
		case "UNPROCESSABLE", "ALREADY_EXISTS", "FORBIDDEN", "NOT_FOUND", "UNAUTHORIZED":
		default:
			return false
		}
	}
	return true
}

func readIssueSyncLabel(ctx context.Context, graph issueGraphQL, repositoryID, name string) (*issueSyncLabel, error) {
	var read struct {
		Node struct {
			ID    string
			Label *issueSyncLabel
		}
	}
	if err := graph(ctx, `query($repository:ID!,$name:String!){node(id:$repository){... on Repository{id label(name:$name){id name}}}}`,
		map[string]any{"repository": repositoryID, "name": name}, &read); err != nil {
		return nil, queueError("projection_label_pending", "label rejection could not be reconciled with an exact native read")
	}
	if read.Node.ID != repositoryID || read.Node.Label == nil || read.Node.Label.ID == "" || read.Node.Label.Name != name {
		return nil, queueError("projection_label_pending", "label rejection has no exact existing native label")
	}
	return read.Node.Label, nil
}
