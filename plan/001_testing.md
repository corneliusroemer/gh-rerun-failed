# Testing Harness Research & Plan

## Problem Statement
The current testing strategy relies heavily on manual testing. This is inefficient and error-prone. The primary challenge in testing `gh-rerun-failed` is its dependency on the GitHub API.
- Real API calls are non-deterministic (state changes, runs expire).
- Rate limits can be hit.
- Setting up specific failure states (failed workflows) on demand is complex.

## Options Considered

### 1. Live End-to-End (E2E) Testing
This involves running tests against a real GitHub repository.
*   **Mechanism**: A dedicated testing repository is set up. A cron job or trigger ensures "fresh" failed runs exist. The test suite invokes the CLI against this repo.
*   **Pros**:
    *   Tests the full stack, including actual API behavior and network conditions.
    *   catches "drift" in API behavior (e.g. if GitHub changes their API).
*   **Cons**:
    *   **Flaky**: Network issues, rate limits, or concurrent test runs can break tests.
    *   **Complex Setup**: Requires maintaining a separate repo with automation to generate failures. Workflow runs expire after 30-90 days, so data needs constant refreshing.
    *   **Slow**: Real network calls are slow.
    *   **Security**: Requires injecting real credentials into CI/CD.

### 2. Manual HTTP Mocking (e.g., `h2non/gock` or `jarcoal/httpmock`)
This involves intercepting HTTP requests and returning predefined static responses. `cli/go-gh` uses a similar internal approach for some unit tests.
*   **Mechanism**: Define expected URLs and methods, and provide JSON blobs as responses.
*   **Pros**:
    *   Fast and deterministic.
    *   No external dependencies (network/credentials).
    *   Good for testing edge cases (e.g., specific error codes, malformed JSON).
*   **Cons**:
    *   **High Maintenance**: GitHub API responses are large and complex. Manually constructing these JSON mocks is tedious.
    *   **Fragile**: If the app changes a query parameter, the mock might stop matching.
    *   **Drift**: Mocks can easily diverge from reality. The test passes, but the app fails in production because the API shape changed.

### 3. Record and Replay (VCR) - **Recommended**
Using a library like [`dnaeon/go-vcr`](https://github.com/dnaeon/go-vcr) to record real interactions and replay them.
*   **Mechanism**:
    *   **Record Mode**: The test runs against the real GitHub API. All requests and responses are saved to a "cassette" (YAML file).
    *   **Replay Mode**: Subsequent runs use the cassette. The HTTP client intercepts requests; if they match a recorded interaction, the recorded response is returned.
*   **Pros**:
    *   **Realistic Data**: Tests run against real API responses, not hand-crafted mocks.
    *   **Deterministic**: Once recorded, the test is stable.
    *   **Easy to Author**: Just write the test, run it once with `record` flag, and commit the file.
    *   **Offline Support**: CI runs do not need credentials or internet access.
*   **Cons**:
    *   **Sensitive Data**: Must ensure tokens/secrets are scrubbed from cassettes (go-vcr supports this).
    *   **Cassette Maintenance**: If the logic changes (e.g., requesting a new field), the cassette must be re-recorded.

## Relevant Resources
*   [go-vcr](https://github.com/dnaeon/go-vcr): The standard VCR library for Go.
*   [cli/go-gh testing](https://github.com/cli/go-gh/blob/trunk/gh_test.go): The official GitHub CLI library uses a mix of internal mocking and helper processes.
*   [gock](https://github.com/h2non/gock): A popular HTTP mocking library.

## Recommendation
**Adopt `go-vcr`**. It strikes the best balance. It solves the "30-day run expiry" issue by freezing the API state in time. It avoids the toil of writing manual mocks.

### Implementation Plan
1.  **Refactor Client**: Modify `internal/gh/NewClient` to accept an optional `http.RoundTripper`. This allows injecting the VCR recorder.
2.  **Setup Test Helper**: Create a `setup` function for tests that:
    *   Checks if we are recording or replaying.
    *   Initializes the VCR recorder.
    *   Scrubs the `Authorization` header from recordings.
3.  **Create "Golden" Scenarios**:
    *   Run against a real repo (e.g., `corneliusroemer/gh-rerun-failed` or a personal fork) once to generate cassettes.
    *   Commit the cassettes.
