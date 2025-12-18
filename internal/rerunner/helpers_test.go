package rerunner

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/corneliusroemer/gh-rerun-failed/internal/gh"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func newTestClient(t *testing.T, cassetteName string) (gh.GHClient, *recorder.Recorder) {
	t.Helper()

	// Set dummy token for go-gh
	os.Setenv("GH_TOKEN", "dummy")

	// Ensure testdata directory exists
	cassetteDir := filepath.Join("testdata", "fixtures")
	if _, err := os.Stat(cassetteDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cassetteDir, 0755); err != nil {
			t.Fatalf("Failed to create cassette directory: %v", err)
		}
	}

	cassettePath := filepath.Join(cassetteDir, cassetteName)

	// Add hooks to remove sensitive information (for recording)
	hook := func(i *cassette.Interaction) error {
		delete(i.Request.Headers, "Authorization")
		return nil
	}

	// Custom matcher to be lenient
	matcher := func(r *http.Request, i cassette.Request) bool {
		if r.Method != i.Method {
			return false
		}
		if r.URL.String() != i.URL {
			return false
		}
		return true
	}

	r, err := recorder.New(
		cassettePath,
		recorder.WithHook(hook, recorder.BeforeSaveHook),
		recorder.WithMatcher(matcher),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	// Inject the recorder as the transport
	client, err := gh.NewClient("corneliusroemer/gh-rerun-failed", r)
	if err != nil {
		t.Fatalf("Failed to create GH client: %v", err)
	}

	return client, r
}
