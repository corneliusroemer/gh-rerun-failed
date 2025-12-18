package gh

import (
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type WorkflowRun struct {
	ID         int64     `json:"id"`
	RunNumber  int       `json:"run_number"`
	RunAttempt int       `json:"run_attempt"`
	Name       string    `json:"name"`
	HeadBranch string    `json:"head_branch"`
	HeadSha    string    `json:"head_sha"`
	Conclusion string    `json:"conclusion"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	HTMLURL    string    `json:"html_url"`
}

type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

type PullRequest struct {
	Number     int    `json:"number"`
	HeadRefOid string `json:"headRefOid"`
	IsDraft    bool   `json:"isDraft"`
	Title      string `json:"title"`
}

type RateLimit struct {
	Limit     int   `json:"limit"`
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"`
}

type Commit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
}

type WorkflowJob struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
}

type GHClient interface {
	FetchWorkflowRuns(branch string, status string, since time.Time, limit int) ([]WorkflowRun, error)
	FetchWorkflowRunsForSha(sha string, status string, limit int) ([]WorkflowRun, error)
	FetchPullRequest(number int) (*PullRequest, error)
	FetchOpenPullRequests() ([]PullRequest, error)
	FetchCommits(branch string, limit int) ([]Commit, error)
	FetchWorkflowRunJobs(runID int64) ([]WorkflowJob, error)
	RerunWorkflow(runID int64, failedOnly bool) error
	GetRateLimit() (*RateLimit, error)
	Repo() repository.Repository
}
