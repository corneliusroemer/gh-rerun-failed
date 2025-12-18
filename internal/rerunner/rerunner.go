package rerunner

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/corneliusroemer/gh-rerun-failed/internal/gh"
)

type Options struct {
	Repo             string
	Branch           string
	Limit            int
	Since            time.Duration
	PRNumber         int
	AllOpenPRs       bool
	DryRun           bool
	FailedOnly       bool
	IncludeDrafts    bool
	IncludeCancelled bool
	IncludeTimedOut  bool
}

type Rerunner struct {
	client gh.GHClient
	opts   Options
}

func NewRerunner(client gh.GHClient, opts Options) *Rerunner {
	return &Rerunner{
		client: client,
		opts:   opts,
	}
}

func (r *Rerunner) Run() error {
	repo := r.client.Repo()
	fmt.Printf("Targeting repository: %s/%s\n", repo.Owner, repo.Name)

	terminal := term.FromEnv()
	width, _, _ := terminal.Size()
	if width <= 0 {
		width = 120 // Default fallback
	}

	var startRate *gh.RateLimit
	if sr, err := r.client.GetRateLimit(); err == nil {
		startRate = sr
		resetTime := time.Unix(startRate.Reset, 0).Format("15:04:05")
		fmt.Printf("[Trace] Rate limit at start: %d/%d (resets at %s)\n",
			startRate.Remaining, startRate.Limit, resetTime)
	} else {
		fmt.Printf("[Warning] Could not fetch start rate limit: %v\n", err)
	}

	// Fetch commits to correlate SHA with distance from tip
	commitMap := make(map[string]int)
	commitMsgMap := make(map[string]string)
	var mu sync.Mutex

	if r.opts.Branch != "" || r.opts.PRNumber == 0 {
		commits, err := r.client.FetchCommits(r.opts.Branch, 50)
		if err == nil {
			for i, c := range commits {
				commitMap[c.SHA] = i
				msg := strings.Split(c.Message, "\n")[0]
				commitMsgMap[c.SHA] = msg
			}
		}
	}

	var runs []gh.WorkflowRun
	var err error

	if r.opts.PRNumber > 0 {
		runs, err = r.fetchRunsForPR(r.opts.PRNumber)
	} else if r.opts.AllOpenPRs {
		runs, err = r.fetchRunsForAllOpenPRs()
	} else {
		runs, err = r.fetchRunsForContextParallel()
	}

	if err != nil {
		return err
	}

	if len(runs) == 0 {
		fmt.Println("No failed workflow runs found matching the criteria.")
		return nil
	}

	totalFound := len(runs)
	// Limit if requested
	if r.opts.Limit > 0 && len(runs) > r.opts.Limit {
		runs = runs[:r.opts.Limit]
	}

	fmt.Printf("Found %d failed/cancelled workflow runs (processed %d). Starting reruns...\n", totalFound, len(runs))

	// Sort runs by CreatedAt descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	// Fetch failed jobs for each run to show matrix entries
	runFailedJobs := make(map[int64][]string)
	var jobMu sync.Mutex
	var jobWg sync.WaitGroup
	jobSem := make(chan struct{}, 10)

	for _, run := range runs {
		jobWg.Add(1)
		jobSem <- struct{}{}
		go func(run gh.WorkflowRun) {
			defer jobWg.Done()
			defer func() { <-jobSem }()
			jobs, err := r.client.FetchWorkflowRunJobs(run.ID)
			if err == nil {
				var failed []string
				for _, j := range jobs {
					if j.Conclusion == "failure" {
						failed = append(failed, j.Name)
					}
				}
				if len(failed) > 0 {
					jobMu.Lock()
					runFailedJobs[run.ID] = failed
					jobMu.Unlock()
				}
			}
		}(run)
	}
	jobWg.Wait()

	if r.opts.DryRun {
		wfW := 40
		attW := 3
		brW := 20
		shaW := 7
		dateW := 19

		maxUrlW := 0
		for _, run := range runs {
			if len(run.HTMLURL) > maxUrlW {
				maxUrlW = len(run.HTMLURL)
			}
		}
		if maxUrlW < 3 {
			maxUrlW = 3
		}

		// Header
		format := "%-40s | %-3s | %-20s | %-7s | %-19s | %-*s | %s\n"
		fmt.Printf("\n"+format,
			"Workflow (+Failed Jobs)", "Att", "Branch@Dist", "SHA", "Created At", maxUrlW, "URL", "Message")

		// Separator line
		overhead := 18 + wfW + attW + brW + shaW + dateW + maxUrlW
		msgW := width - overhead
		if msgW < 20 {
			msgW = 20
		}
		lineLen := overhead + msgW
		if lineLen > width {
			lineLen = width
		}
		fmt.Println(strings.Repeat("-", lineLen))

		rowFormat := "%-40s | %-3d | %-20s | %-7s | %-19s | %-*s | %s\n"
		var wg sync.WaitGroup
		sem := make(chan struct{}, 5)

		for _, run := range runs {
			wg.Add(1)
			sem <- struct{}{}
			go func(run gh.WorkflowRun) {
				defer wg.Done()
				defer func() { <-sem }()

				sha := run.HeadSha
				if len(sha) > 7 {
					sha = sha[:7]
				}

				distance := "HEAD^?"
				if d, ok := commitMap[run.HeadSha]; ok {
					if d == 0 {
						distance = "HEAD"
					} else {
						distance = fmt.Sprintf("HEAD^%d", d)
					}
				}

				mu.Lock()
				msg, ok := commitMsgMap[run.HeadSha]
				mu.Unlock()

				if !ok {
					// Fallback: fetch single commit info
					c, err := r.client.FetchCommit(run.HeadSha)
					if err == nil {
						msg = strings.Split(c.Message, "\n")[0]
						mu.Lock()
						commitMsgMap[run.HeadSha] = msg
						mu.Unlock()
					}
				}

				if msg == "" {
					msg = "unknown"
				}

				createdAt := run.CreatedAt.Format("2006-01-02 15:04:05")

				name := run.Name
				if failed, ok := runFailedJobs[run.ID]; ok {
					name = fmt.Sprintf("%s (%s)", name, strings.Join(failed, ", "))
				}

				fmt.Printf(rowFormat,
					truncate(name, wfW),
					run.RunAttempt,
					truncate(fmt.Sprintf("%s (%s)", run.HeadBranch, distance), brW),
					sha,
					createdAt,
					maxUrlW,
					run.HTMLURL,
					truncate(msg, msgW))
			}(run)
		}
		wg.Wait()
		fmt.Println("Dry-run complete. No reruns were triggered.")
		return nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limit concurrency to 5

	for _, run := range runs {
		wg.Add(1)
		sem <- struct{}{}

		go func(run gh.WorkflowRun) {
			defer wg.Done()
			defer func() { <-sem }()

			sha := run.HeadSha
			if len(sha) > 7 {
				sha = sha[:7]
			}

			err := r.client.RerunWorkflow(run.ID, r.opts.FailedOnly)
			if err != nil {
				fmt.Printf("✗ Failed to rerun %d (%s): %v\n", run.ID, run.Name, err)
			} else {
				fmt.Printf("✓ Triggered rerun for: %s (%s) | #%d (attempt %d) | %s\n",
					run.Name, run.HeadBranch, run.RunNumber, run.RunAttempt, sha)
			}
		}(run)
	}

	wg.Wait()

	endRate, err := r.client.GetRateLimit()
	if err == nil {
		spent := 0
		if startRate != nil {
			spent = startRate.Remaining - endRate.Remaining
		}
		fmt.Printf("[Trace] Rate limit at end: %d/%d (spent %d)\n",
			endRate.Remaining, endRate.Limit, spent)
	}

	if !r.opts.DryRun {
		fmt.Println("Done triggering reruns.")
	}
	return nil
}

func truncate(s string, l int) string {
	if len(s) > l {
		if l > 3 {
			return s[:l-3] + "..."
		}
		return s[:l]
	}
	return s
}

func (r *Rerunner) fetchRunsForPR(number int) ([]gh.WorkflowRun, error) {
	pr, err := r.client.FetchPullRequest(number)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR #%d: %w", number, err)
	}
	return r.fetchFailedRunsForSha(pr.HeadRefOid)
}

func (r *Rerunner) fetchRunsForAllOpenPRs() ([]gh.WorkflowRun, error) {
	prs, err := r.client.FetchOpenPullRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch open PRs: %w", err)
	}

	var allRuns []gh.WorkflowRun
	var sinceTime time.Time
	if r.opts.Since > 0 {
		sinceTime = time.Now().Add(-r.opts.Since)
	}

	consecutiveEmpty := 0
	for _, pr := range prs {
		if pr.IsDraft && !r.opts.IncludeDrafts {
			continue
		}
		runs, err := r.fetchFailedRunsForSha(pr.HeadRefOid)
		if err != nil {
			fmt.Printf("Warning: failed to fetch runs for PR #%d: %v\n", pr.Number, err)
			continue
		}

		if len(runs) > 0 {
			allRuns = append(allRuns, runs...)
			consecutiveEmpty = 0
		} else if !sinceTime.IsZero() {
			consecutiveEmpty++
			// If we found NO recent failed runs for 5 PRs in a row,
			// and since PRs are ordered by newest first, it's very likely
			// that older PRs also won't have recent failed runs.
			if consecutiveEmpty >= 5 {
				fmt.Printf("[Trace] Stopping PR fetch after %d consecutive PRs with no recent failed runs.\n", consecutiveEmpty)
				break
			}
		}
	}
	return allRuns, nil
}

func (r *Rerunner) fetchRunsForContextParallel() ([]gh.WorkflowRun, error) {
	var sinceTime time.Time
	if r.opts.Since > 0 {
		sinceTime = time.Now().Add(-r.opts.Since)
	}

	statuses := []string{"failure"}
	if r.opts.IncludeCancelled {
		statuses = append(statuses, "cancelled")
	}
	if r.opts.IncludeTimedOut {
		statuses = append(statuses, "timed_out")
	}

	var allRuns []gh.WorkflowRun
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(statuses))

	for _, status := range statuses {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			runs, err := r.client.FetchWorkflowRuns(r.opts.Branch, s, sinceTime, r.opts.Limit)
			if err != nil {
				errChan <- err
				return
			}
			mu.Lock()
			allRuns = append(allRuns, runs...)
			mu.Unlock()
		}(status)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return allRuns, nil
}

func (r *Rerunner) fetchFailedRunsForSha(sha string) ([]gh.WorkflowRun, error) {
	var allRuns []gh.WorkflowRun
	var mu sync.Mutex
	var wg sync.WaitGroup

	statuses := []string{"failure"}
	if r.opts.IncludeCancelled {
		statuses = append(statuses, "cancelled")
	}
	if r.opts.IncludeTimedOut {
		statuses = append(statuses, "timed_out")
	}

	for _, status := range statuses {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			runs, err := r.client.FetchWorkflowRunsForSha(sha, s, r.opts.Limit)
			if err != nil {
				fmt.Printf("Warning: failed to fetch %s runs for sha %s: %v\n", s, sha, err)
				return
			}
			mu.Lock()
			allRuns = append(allRuns, runs...)
			mu.Unlock()
		}(status)
	}
	wg.Wait()

	// Filter by since if needed
	if r.opts.Since > 0 {
		sinceTime := time.Now().Add(-r.opts.Since)
		var filtered []gh.WorkflowRun
		for _, run := range allRuns {
			if run.CreatedAt.After(sinceTime) {
				filtered = append(filtered, run)
			}
		}
		return filtered, nil
	}

	return allRuns, nil
}
