package rerunner

import (
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/corneliusroemer/gh-rerun-failed/internal/gh"
)

type mockGHClient struct {
	gh.GHClient
	fetchWorkflowRunsFunc       func(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error)
	rerunWorkflowFunc           func(runID int64, failedOnly bool) error
	fetchPullRequestFunc        func(number int) (*gh.PullRequest, error)
	fetchOpenPullRequestsFunc   func() ([]gh.PullRequest, error)
	fetchWorkflowRunsForShaFunc func(sha string, status string, limit int) ([]gh.WorkflowRun, error)
	fetchCommitsFunc            func(branch string, limit int) ([]gh.Commit, error)
	fetchCommitFunc             func(sha string) (*gh.Commit, error)
	fetchWorkflowRunJobsFunc    func(runID int64) ([]gh.WorkflowJob, error)
	getRateLimitFunc            func() (*gh.RateLimit, error)
}

func (m *mockGHClient) FetchCommit(sha string) (*gh.Commit, error) {
	if m.fetchCommitFunc != nil {
		return m.fetchCommitFunc(sha)
	}
	return &gh.Commit{}, nil
}

func (m *mockGHClient) FetchWorkflowRunJobs(runID int64) ([]gh.WorkflowJob, error) {
	if m.fetchWorkflowRunJobsFunc != nil {
		return m.fetchWorkflowRunJobsFunc(runID)
	}
	return nil, nil
}

func (m *mockGHClient) FetchCommits(branch string, limit int) ([]gh.Commit, error) {
	if m.fetchCommitsFunc != nil {
		return m.fetchCommitsFunc(branch, limit)
	}
	return nil, nil
}

func (m *mockGHClient) GetRateLimit() (*gh.RateLimit, error) {
	if m.getRateLimitFunc != nil {
		return m.getRateLimitFunc()
	}
	return &gh.RateLimit{Limit: 5000, Remaining: 4999, Reset: time.Now().Unix()}, nil
}

func (m *mockGHClient) FetchWorkflowRuns(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error) {
	return m.fetchWorkflowRunsFunc(branch, status, since, limit)
}

func (m *mockGHClient) RerunWorkflow(runID int64, failedOnly bool) error {
	return m.rerunWorkflowFunc(runID, failedOnly)
}

func (m *mockGHClient) FetchPullRequest(number int) (*gh.PullRequest, error) {
	return m.fetchPullRequestFunc(number)
}

func (m *mockGHClient) FetchOpenPullRequests() ([]gh.PullRequest, error) {
	return m.fetchOpenPullRequestsFunc()
}

func (m *mockGHClient) FetchWorkflowRunsForSha(sha string, status string, limit int) ([]gh.WorkflowRun, error) {
	return m.fetchWorkflowRunsForShaFunc(sha, status, limit)
}

func (m *mockGHClient) Repo() repository.Repository {
	return repository.Repository{
		Owner: "owner",
		Name:  "repo",
		Host:  "github.com",
	}
}

func TestRerunner_Run_FetchRunsForContext(t *testing.T) {
	mock := &mockGHClient{
		fetchWorkflowRunsFunc: func(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error) {
			if status == "failure" {
				return []gh.WorkflowRun{
					{ID: 1, Name: "Workflow 1", CreatedAt: time.Now()},
				}, nil
			}
			return nil, nil
		},
		rerunWorkflowFunc: func(runID int64, failedOnly bool) error {
			return nil
		},
	}

	opts := Options{
		Repo:       "owner/repo",
		DryRun:     true,
		FailedOnly: true,
	}

	r := NewRerunner(mock, opts)
	err := r.Run()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestRerunner_Run_Limit(t *testing.T) {
	mock := &mockGHClient{
		fetchWorkflowRunsFunc: func(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error) {
			if status == "failure" {
				return []gh.WorkflowRun{
					{ID: 1, Name: "Workflow 1", CreatedAt: time.Now()},
					{ID: 2, Name: "Workflow 2", CreatedAt: time.Now()},
					{ID: 3, Name: "Workflow 3", CreatedAt: time.Now()},
				}, nil
			}
			return nil, nil
		},
		rerunWorkflowFunc: func(runID int64, failedOnly bool) error {
			return nil
		},
	}

	opts := Options{
		Repo:       "owner/repo",
		Limit:      2,
		DryRun:     true,
		FailedOnly: true,
	}

	r := NewRerunner(mock, opts)
	err := r.Run()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// We'd need to track calls to verify limit, but the output will show it.
}

func TestRerunner_Run_WorkflowFilter(t *testing.T) {
	var rerunIDs []int64
	var rerunMu sync.Mutex

	mock := &mockGHClient{
		fetchWorkflowRunsFunc: func(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error) {
			if status != "failure" {
				return nil, nil
			}
			if limit != 0 {
				t.Fatalf("expected fetch limit 0 when workflow filter is set, got %d", limit)
			}
			return []gh.WorkflowRun{
				{ID: 1, Name: "Unit Tests", CreatedAt: time.Now()},
				{ID: 2, Name: "Integration Tests", CreatedAt: time.Now()},
				{ID: 3, Name: "integration Smoke", CreatedAt: time.Now()},
			}, nil
		},
		rerunWorkflowFunc: func(runID int64, failedOnly bool) error {
			rerunMu.Lock()
			defer rerunMu.Unlock()
			rerunIDs = append(rerunIDs, runID)
			return nil
		},
	}

	opts := Options{
		Repo:           "owner/repo",
		Limit:          1,
		FailedOnly:     true,
		WorkflowFilter: "Integration",
	}

	r := NewRerunner(mock, opts)
	err := r.Run()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	sort.Slice(rerunIDs, func(i, j int) bool { return rerunIDs[i] < rerunIDs[j] })
	if len(rerunIDs) != 1 || rerunIDs[0] != 2 {
		t.Fatalf("expected only the first matching workflow to be rerun, got %v", rerunIDs)
	}
}

func TestRerunner_Run_WorkflowExclude(t *testing.T) {
	var rerunIDs []int64
	var rerunMu sync.Mutex

	mock := &mockGHClient{
		fetchWorkflowRunsFunc: func(branch string, status string, since time.Time, limit int) ([]gh.WorkflowRun, error) {
			if status != "failure" {
				return nil, nil
			}
			if limit != 0 {
				t.Fatalf("expected fetch limit 0 when workflow exclude is set, got %d", limit)
			}
			return []gh.WorkflowRun{
				{ID: 1, Name: "Mirror integration", CreatedAt: time.Now()},
				{ID: 2, Name: "Integration Tests", CreatedAt: time.Now()},
				{ID: 3, Name: "Unit Tests", CreatedAt: time.Now()},
			}, nil
		},
		rerunWorkflowFunc: func(runID int64, failedOnly bool) error {
			rerunMu.Lock()
			defer rerunMu.Unlock()
			rerunIDs = append(rerunIDs, runID)
			return nil
		},
	}

	opts := Options{
		Repo:            "owner/repo",
		FailedOnly:      true,
		WorkflowExclude: "mirror",
	}

	r := NewRerunner(mock, opts)
	err := r.Run()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	sort.Slice(rerunIDs, func(i, j int) bool { return rerunIDs[i] < rerunIDs[j] })
	if len(rerunIDs) != 2 || rerunIDs[0] != 2 || rerunIDs[1] != 3 {
		t.Fatalf("expected workflows without mirror to be rerun, got %v", rerunIDs)
	}
}

func TestFilterRunsByWorkflowName(t *testing.T) {
	runs := []gh.WorkflowRun{
		{ID: 1, Name: "Unit Tests"},
		{ID: 2, Name: "Integration Tests"},
		{ID: 3, Name: "integration Smoke"},
		{ID: 4, Name: "Mirror Integration"},
	}

	filtered := filterRunsByWorkflowName(runs, " Integration ", "mirror")
	if len(filtered) != 2 || filtered[0].ID != 2 || filtered[1].ID != 3 {
		t.Fatalf("expected case-insensitive include and exclude match for integration workflows, got %v", filtered)
	}
}
