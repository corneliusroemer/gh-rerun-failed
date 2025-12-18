package gh

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
)

type Client struct {
	restClient    *api.RESTClient
	graphqlClient *api.GraphQLClient
	repo          repository.Repository
}

func NewClient(repoOverride string, transport http.RoundTripper) (GHClient, error) {
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

	restOpts := api.ClientOptions{
		Transport: transport,
	}
	restClient, err := api.NewRESTClient(restOpts)
	if err != nil {
		return nil, fmt.Errorf("could not create GitHub REST API client: %w", err)
	}

	gqlOpts := api.ClientOptions{
		Transport: transport,
	}
	graphqlClient, err := api.NewGraphQLClient(gqlOpts)
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
	// First fetch page 1 to get TotalCount
	firstPage, totalCount, err := c.fetchWorkflowRunsPage(branch, status, 1, 100)
	if err != nil {
		return nil, err
	}

	allRuns := firstPage
	if !since.IsZero() {
		// Filter first page
		var filtered []WorkflowRun
		for _, run := range allRuns {
			if run.CreatedAt.After(since) {
				filtered = append(filtered, run)
			}
		}
		allRuns = filtered
		// If we already hit runs older than since, or we have enough runs for the limit, return early
		if (len(firstPage) > 0 && firstPage[len(firstPage)-1].CreatedAt.Before(since)) || (limit > 0 && len(allRuns) >= limit) {
			if limit > 0 && len(allRuns) > limit {
				allRuns = allRuns[:limit]
			}
			return allRuns, nil
		}
	}

	remainingPages := (totalCount + 99) / 100
	if remainingPages > 10 {
		remainingPages = 10 // Cap at 10 pages
	}

	if remainingPages <= 1 {
		if limit > 0 && len(allRuns) > limit {
			allRuns = allRuns[:limit]
		}
		return allRuns, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, remainingPages-1)

	for p := 2; p <= remainingPages; p++ {
		wg.Add(1)
		go func(page int) {
			defer wg.Done()
			runs, _, err := c.fetchWorkflowRunsPage(branch, status, page, 100)
			if err != nil {
				errChan <- err
				return
			}

			var filtered []WorkflowRun
			for _, run := range runs {
				if !since.IsZero() && run.CreatedAt.Before(since) {
					continue
				}
				filtered = append(filtered, run)
			}

			mu.Lock()
			allRuns = append(allRuns, filtered...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	close(errChan)
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	if limit > 0 && len(allRuns) > limit {
		sort.Slice(allRuns, func(i, j int) bool {
			return allRuns[i].CreatedAt.After(allRuns[j].CreatedAt)
		})
		allRuns = allRuns[:limit]
	}

	return allRuns, nil
}

func (c *Client) fetchWorkflowRunsPage(branch string, status string, page int, perPage int) ([]WorkflowRun, int, error) {
	path := fmt.Sprintf("repos/%s/%s/actions/runs?status=%s&per_page=%d&page=%d", c.repo.Owner, c.repo.Name, status, perPage, page)
	if branch != "" {
		path += fmt.Sprintf("&branch=%s", branch)
	}

	fmt.Printf("Fetching page %d for runs with status %s...\n", page, status)
	var response WorkflowRunsResponse
	err := c.restClient.Get(path, &response)
	if err != nil {
		return nil, 0, err
	}
	return response.WorkflowRuns, response.TotalCount, nil
}
func (c *Client) FetchWorkflowRunsForSha(sha string, status string, limit int) ([]WorkflowRun, error) {
	var allRuns []WorkflowRun
	page := 1
	perPage := 100

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

func (c *Client) FetchCommit(sha string) (*Commit, error) {
	path := fmt.Sprintf("repos/%s/%s/commits/%s", c.repo.Owner, c.repo.Name, sha)

	var response struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	}

	err := c.restClient.Get(path, &response)
	if err != nil {
		return nil, err
	}

	return &Commit{
		SHA:     response.SHA,
		Message: response.Commit.Message,
	}, nil
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
