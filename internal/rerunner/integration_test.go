package rerunner

import (
	"testing"
	"time"
)

func TestIntegration_DryRun(t *testing.T) {
	client, rec := newTestClient(t, "TestIntegration_DryRun")
	defer rec.Stop()

	opts := Options{
		Repo:       "corneliusroemer/gh-rerun-failed",
		Branch:     "",
		Limit:      5,
		Since:      2400 * time.Hour, // Long duration to ensure we capture old runs in the cassette
		DryRun:     true,
		FailedOnly: true,
	}

	r := NewRerunner(client, opts)
	err := r.Run()
	if err != nil {
		t.Fatalf("Rerunner.Run failed: %v", err)
	}
}
