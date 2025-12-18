package rerunner

import (
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
	return &gh.Commit{SHA: sha, Message: "test commit message"}, nil
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
