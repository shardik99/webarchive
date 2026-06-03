# Webarchive Test Plan

This document outlines the testing strategy, currently enabled test suites, and continuous integration (CI) workflows for the `webarchive` project.

## Current Testing Strategy

The project utilizes the standard Go testing library (`testing`) and the `testify` toolkit (specifically `assert` and `require`) for writing assertions. Tests are separated into fast unit tests and slower integration tests that require external resources or disk access.

### 1. Unit Tests
*   **Config Testing (`config/config_test.go`)**: Verifies that the configuration system correctly parses environment variables, respects prefixes (e.g., `WEBARCHIVE_DB_PATH`), and falls back to default values when variables are not set.

### 2. Integration / External Resource Tests
These tests interact with the file system or make actual HTTP requests. They can be skipped during fast test runs by passing the `-short` flag to `go test`.
*   **Database Testing (`adapters/repository/badger/page_test.go`)**: Tests the `BadgerDB` persistence layer by creating a temporary directory, initializing the database, and validating that a `Page` entity can be saved and retrieved accurately. *Skipped when `testing.Short()` is active.*
*   **Processor Metadata Testing (`adapters/processors/processors_test.go`)**: Makes a live network call to fetch a real web page and asserts that the `<title>` and metadata are extracted correctly.
*   **PDF Generation (`adapters/processors/pdf_test.go`)**: Located behind a special build tag (`//go:build integration`), this test fetches a live URL and utilizes the `wkhtmltopdf` binary to render a PDF document, validating the file size and metadata of the generated `File` struct.

## Continuous Integration (CI)

The project leverages **GitHub Actions** for continuous integration. The workflow is defined in `.github/workflows/test.yaml`.

*   **Triggers**: The pipeline runs automatically on all Pull Requests and direct pushes to the `master` branch.
*   **Environment**: Uses `ubuntu-latest` with Go version 1.23.x.
*   **Caching**: Caches `~/.cache/go-build` and `~/go/pkg/mod` to speed up successive builds.
*   **Execution**:
    1.  **Unit & Integration Tests**: Runs `go test ./...` across the entire project. Note that since the `integration` build tag is not explicitly passed in the CI command, the `pdf_test.go` may not be executed in the default CI run, although the database tests and metadata tests will run.
    2.  **Linting**: Runs `golangci-lint` to enforce code quality, styling, and catch common static analysis errors.

## Proposed Tests (Identified Testing Gaps)

To improve test coverage and reliability, the following testing gaps have been identified and are proposed for future implementation:

1.  **API Handler Tests**: *Gap: No testing of the REST endpoints.*
    *   **Proposal**: Add `httptest` suites for the `ports/rest` endpoints to ensure requests are parsed correctly, HTTP status codes are accurate, and edge cases (like invalid URLs) are handled gracefully.
2.  **Worker Logic Tests**: *Gap: No testing of the core asynchronous background processor.*
    *   **Proposal**: Test the asynchronous `Worker` to ensure it properly marks pages as `Processing`, `Done`, or `Failed`, especially when dealing with channel closures or processor errors.
3.  **Mocking External Dependencies**: *Gap: Tests are flaky because they rely on external websites (e.g., github.com).*
    *   **Proposal**: Currently, the processor tests rely on live web pages. These should be refactored to use an embedded `httptest.Server` so the tests are deterministic and do not fail due to external network issues.
4.  **CI Integration Tag**: *Gap: PDF generation tests are skipped in GitHub Actions.*
    *   **Proposal**: Update the GitHub actions workflow to run a matrix with and without the `-tags integration` flag to ensure the PDF generation path is tested in the pipeline.
