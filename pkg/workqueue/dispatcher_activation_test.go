package workqueue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

type dispatcherActivationTransport func(*http.Request) (*http.Response, error)

func (transport dispatcherActivationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestPrepareDispatcherActivation(t *testing.T) {
	sha := strings.Repeat("f", 40)
	tag := "refs/tags/gh-aw/work-queue/" + sha
	for _, test := range []struct {
		name       string
		initial    int
		create     int
		message    string
		readback   int
		ref, kind  string
		revision   string
		wantError  bool
		wantReads  int
		wantWrites int
	}{
		{name: "existing", initial: 200, ref: tag, kind: "commit", revision: sha, wantReads: 1},
		{name: "created", initial: 404, create: 201, readback: 200, ref: tag, kind: "commit", revision: sha, wantReads: 2, wantWrites: 1},
		{name: "concurrent creator", initial: 404, create: 422, message: "Reference already exists", readback: 200, ref: tag, kind: "commit", revision: sha, wantReads: 2, wantWrites: 1},
		{name: "collision wrong target", initial: 404, create: 422, message: "Reference already exists", readback: 200, ref: tag, kind: "commit", revision: strings.Repeat("a", 40), wantError: true, wantReads: 2, wantWrites: 1},
		{name: "wrong ref", initial: 200, ref: tag + "-other", kind: "commit", revision: sha, wantError: true, wantReads: 1},
		{name: "wrong revision", initial: 200, ref: tag, kind: "commit", revision: strings.Repeat("a", 40), wantError: true, wantReads: 1},
		{name: "annotated tag", initial: 200, ref: tag, kind: "tag", revision: sha, wantError: true, wantReads: 1},
		{name: "missing type", initial: 200, ref: tag, revision: sha, wantError: true, wantReads: 1},
		{name: "read denied", initial: 403, wantError: true, wantReads: 1},
		{name: "read throttled", initial: 429, wantError: true, wantReads: 1},
		{name: "create denied", initial: 404, create: 403, wantError: true, wantReads: 1, wantWrites: 1},
		{name: "other validation failure", initial: 404, create: 422, message: "Validation Failed", wantError: true, wantReads: 1, wantWrites: 1},
		{name: "ambiguous create response", initial: 404, create: 500, wantError: true, wantReads: 1, wantWrites: 1},
		{name: "ambiguous create transport", initial: 404, create: -1, wantError: true, wantReads: 1, wantWrites: 1},
		{name: "missing readback", initial: 404, create: 201, readback: 404, wantError: true, wantReads: 2, wantWrites: 1},
		{name: "readback denied", initial: 404, create: 201, readback: 403, wantError: true, wantReads: 2, wantWrites: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			reads, writes := 0, 0
			transport := dispatcherActivationTransport(func(request *http.Request) (*http.Response, error) {
				if request.Header.Get("Authorization") != "token test-token" {
					t.Fatal("activation request was not authenticated")
				}
				status := test.initial
				var value any
				switch {
				case request.Method == http.MethodGet && request.URL.Path == "/repos/owner/repo/git/ref/"+strings.TrimPrefix(tag, "refs/"):
					reads++
					if reads > 1 {
						status = test.readback
					}
					value = map[string]any{"ref": test.ref, "object": map[string]string{"type": test.kind, "sha": test.revision}}
				case request.Method == http.MethodPost && request.URL.Path == "/repos/owner/repo/git/refs":
					writes++
					var payload map[string]string
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if len(payload) != 2 || payload["ref"] != tag || payload["sha"] != sha {
						t.Fatalf("unexpected snapshot target: %v", payload)
					}
					if test.create == -1 {
						return nil, errors.New("ambiguous transport failure")
					}
					status = test.create
					value = map[string]string{"message": test.message}
				default:
					t.Fatalf("unexpected request or ref overwrite: %s %s", request.Method, request.URL.Path)
				}
				if status >= 400 && request.Method == http.MethodGet {
					value = map[string]string{"message": http.StatusText(status)}
				}
				body, _ := json.Marshal(value)
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
			})
			client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", Transport: transport})
			if err != nil {
				t.Fatal(err)
			}
			branch := Branch{Remote: "owner/repo", Name: DefaultBranch, client: client}
			ref, err := branch.PrepareDispatcherActivation(context.Background(), DispatcherRoute{Repository: "owner/repo", Revision: sha})
			if (err != nil) != test.wantError || (!test.wantError && ref != tag) || (test.wantError && ref != "") {
				t.Fatalf("unexpected activation snapshot: ref=%q err=%v", ref, err)
			}
			if reads != test.wantReads || writes != test.wantWrites {
				t.Fatalf("unexpected request count or retry: reads=%d writes=%d", reads, writes)
			}
		})
	}
}

func TestPrepareDispatcherActivationRejectsInvalidRoute(t *testing.T) {
	client, err := api.NewRESTClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token",
		Transport: dispatcherActivationTransport(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid route reached the API")
			return nil, nil
		})})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []DispatcherRoute{
		{Repository: "other/repo", Revision: strings.Repeat("f", 40)},
		{Repository: "owner/repo", Revision: "main"},
		{Repository: "owner/repo", Revision: strings.Repeat("0", 40)},
	} {
		if ref, err := (Branch{Remote: "owner/repo", Name: DefaultBranch, client: client}).PrepareDispatcherActivation(context.Background(), route); err == nil || ref != "" {
			t.Fatalf("invalid route accepted: %v %q %v", route, ref, err)
		}
	}
}
