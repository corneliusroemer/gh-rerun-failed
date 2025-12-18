package gh

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type Client struct {
	restClient    *api.RESTClient
	graphqlClient *api.GraphQLClient
	repo          repository.Repository
}

func NewClient(repoOverride string) (GHClient, error) {
	var repo repository.Repository
	var err error

	if repoOverride != "" {
		repo, err = repository.Parse(repoOverride)
	} else {
		repo, err = repository.Current()
	}

	if err != nil {
		return nil, fmt.Errorf("could not determine repository: %w", err)
	}

	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, fmt.Errorf("could not create GitHub REST API client: %w", err)
	}

	graphqlClient, err := api.DefaultGraphQLClient()
	if err != nil {
		return nil, fmt.Errorf("could not create GitHub GraphQL API client: %w", err)
	}

	return &Client{
		restClient:    restClient,
		graphqlClient: graphqlClient,
		repo:          repo,
	}, nil
}

func (c *Client) FetchWorkflowRuns(branch string, status string, since time.Time, limit int) ([]WorkflowRun, error) {
	var allRuns []WorkflowRun
	page := 1
	perPage := 100
	if limit > 0 && limit < 100 {
		perPage = limit
	}

	for {
		path := fmt.Sprintf("repos/%s/%s/actions/runs?status=%s&per_page=%d&page=%d", c.repo.Owner, c.repo.Name, status, perPage, page)
		if branch != "" {
			path += fmt.Sprintf("&branch=%s", branch)
		}

		fmt.Printf("Fetching page %d for runs with status %s...\n", page, status)
		var response WorkflowRunsResponse
		err := c.restClient.Get(path, &response)
		if err != nil {
			return nil, err
		}

		if len(response.WorkflowRuns) == 0 {
			break
		}

		for _, run := range response.WorkflowRuns {
			if !since.IsZero() && run.CreatedAt.Before(since) {
				continue
			}
			allRuns = append(allRuns, run)
			if limit > 0 && len(allRuns) >= limit {
				return allRuns, nil
			}
		}

		if len(allRuns) >= response.TotalCount || len(response.WorkflowRuns) < perPage {
			break
		}

		if page > 10 {
			break
		}
		page++
	}

	return allRuns, nil
}

func (c *Client) FetchWorkflowRunsForSha(sha string, status string, limit int) ([]WorkflowRun, error) {
	var allRuns []WorkflowRun
	page := 1
	perPage := 100
	if limit > 0 && limit < 100 {
		perPage = limit
	}

	for {
		path := fmt.Sprintf("repos/%s/%s/actions/runs?head_sha=%s&per_page=%d&page=%d", c.repo.Owner, c.repo.Name, sha, perPage, page)
		if status != "" {
			path += fmt.Sprintf("&status=%s", status)
		}

		fmt.Printf("Fetching page %d for runs with SHA %s and status %s...\n", page, sha, status)
		var response WorkflowRunsResponse
		err := c.restClient.Get(path, &response)
		if err != nil {
			return nil, err
		}

		if len(response.WorkflowRuns) == 0 {
			break
		}

		for _, run := range response.WorkflowRuns {
			allRuns = append(allRuns, run)
			if limit > 0 && len(allRuns) >= limit {
				return allRuns, nil
			}
		}

		if len(allRuns) >= response.TotalCount || len(response.WorkflowRuns) < perPage {
			break
		}
		if page > 5 {
			break
		}
		page++
	}

	return allRuns, nil
}

func (c *Client) FetchPullRequest(number int) (*PullRequest, error) {
	query := `
		query GetPR($owner: String!, $name: String!, $number: Int!) {
			repository(owner: $owner, name: $name) {
				pullRequest(number: $number) {
					number
					headRefOid
					isDraft
					title
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner":  c.repo.Owner,
		"name":   c.repo.Name,
		"number": number,
	}

	var response struct {
		Repository struct {
			PullRequest PullRequest `json:"pullRequest"`
		} `json:"repository"`
	}

	err := c.graphqlClient.Do(query, variables, &response)
	if err != nil {
		return nil, err
	}

	return &response.Repository.PullRequest, nil
}

func (c *Client) FetchOpenPullRequests() ([]PullRequest, error) {
	query := `
		query ListPRs($owner: String!, $name: String!) {
			repository(owner: $owner, name: $name) {
				pullRequests(first: 100, states: OPEN, orderBy: {field: CREATED_AT, direction: DESC}) {
					nodes {
						number
						headRefOid
						isDraft
						title
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"owner": c.repo.Owner,
		"name":  c.repo.Name,
	}

	var response struct {
		Repository struct {
			PullRequests struct {
				Nodes []PullRequest `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	}

	err := c.graphqlClient.Do(query, variables, &response)
	if err != nil {
		return nil, err
	}

	return response.Repository.PullRequests.Nodes, nil
}

func (c *Client) FetchCommits(branch string, limit int) ([]Commit, error) {
	path := fmt.Sprintf("repos/%s/%s/commits?per_page=%d", c.repo.Owner, c.repo.Name, limit)
	if branch != "" {
		path += fmt.Sprintf("&sha=%s", branch)
	}

	var response []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}

	err := c.restClient.Get(path, &response)
	if err != nil {
		return nil, err
	}

	commits := make([]Commit, len(response))
	for i, r := range response {
		commits[i] = Commit{
			SHA:     r.SHA,
			Message: r.Commit.Message,
		}
	}

	return commits, nil
}

func (c *Client) FetchWorkflowRunJobs(runID int64) ([]WorkflowJob, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/jobs", c.repo.Owner, c.repo.Name, runID)

	var response struct {
		Jobs []WorkflowJob `json:"jobs"`
	}
	err := c.restClient.Get(path, &response)
	if err != nil {
		return nil, err
	}
	return response.Jobs, nil
}

func (c *Client) RerunWorkflow(runID int64, failedOnly bool) error {
	endpoint := "rerun"
	if failedOnly {
		endpoint = "rerun-failed-jobs"
	}
	path := fmt.Sprintf("repos/%s/%s/actions/runs/%d/%s", c.repo.Owner, c.repo.Name, runID, endpoint)

	return c.restClient.Post(path, nil, nil)
}

func (c *Client) GetRateLimit() (*RateLimit, error) {
	var response struct {
		Resources struct {
			Core RateLimit `json:"core"`
		} `json:"resources"`
	}
	err := c.restClient.Get("rate_limit", &response)
	if err != nil {
		return nil, err
	}
	return &response.Resources.Core, nil
}

func (c *Client) Repo() repository.Repository {
	return c.repo
}
