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
		Since:      24 * time.Hour,
		DryRun:     true,
		FailedOnly: true,
	}

	r := NewRerunner(client, opts)
	// Mock time to match the cassette (2025-12-18 12:00:00 UTC)
	// The run was created at 10:00:00 UTC.
	// 12:00 - 24h = Previous day 12:00.
	// 10:00 is after Previous day 12:00? No, wait.
	// Since 24h means "runs created after (Now - 24h)".
	// Now = Dec 18 12:00.
	// Now - 24h = Dec 17 12:00.
	// Run = Dec 18 10:00.
	// Dec 18 10:00 > Dec 17 12:00. So it should be included.
	r.NowFunc = func() time.Time {
		return time.Date(2025, 12, 18, 12, 0, 0, 0, time.UTC)
	}

	err := r.Run()
	if err != nil {
		t.Fatalf("Rerunner.Run failed: %v", err)
	}
}
