# BackOffice E2E Testing

This directory contains end-to-end tests for the BackOffice library using Playwright Go. The tests are isolated in a separate Go module to prevent test dependencies from leaking to library users.

## Architecture

- **Separate Module**: This is an independent Go module (`backoffice-e2e-testing`)
- **Isolated Dependencies**: Playwright and other test dependencies don't affect the main library
- **Ground Truth Testing**: Comprehensive tests covering all BackOffice functionality
- **HTMX Aware**: Properly waits for asynchronous HTMX requests to complete

## Quick Start

### Prerequisites

- Go 1.24+ installed
- Internet connection (for downloading Playwright browsers on first run)

### Running Tests

```bash
# From the e2e_testing directory
cd e2e_testing
./run_e2e_tests.sh
```

The script will automatically:
1. Install Playwright dependencies if needed
2. Start the BackOffice demo application
3. Wait for it to be ready
4. Run comprehensive E2E tests
5. Clean up the demo application

### Configuration Options

```bash
# Run with visible browser (non-headless)
./run_e2e_tests.sh --headless false

# Use different port
./run_e2e_tests.sh --port 3000

# Slow down operations for debugging
./run_e2e_tests.sh --slow-mo 500ms

# Keep demo running after tests (for manual inspection)
./run_e2e_tests.sh --no-cleanup

# Longer timeout for slower systems
./run_e2e_tests.sh --timeout 60s

# Wait longer for demo startup
./run_e2e_tests.sh --wait 10
```

### Environment Variables

```bash
export BACKOFFICE_E2E_PORT=3000
export BACKOFFICE_E2E_HEADLESS=false
./run_e2e_tests.sh
```

## Test Coverage

The E2E test suite covers:

### Core Functionality
- **Homepage**: Navigation, resource listing, counts
- **User CRUD**: Create, read, update, delete operations
- **Product CRUD**: Full lifecycle testing
- **Category CRUD**: Basic operations
- **Navigation**: Sidebar, breadcrumbs, URL routing

### UI/UX Testing
- **Form Handling**: Field validation, submission
- **HTMX Interactions**: Asynchronous requests, proper waiting
- **Modal Dialogs**: Delete confirmations, form modals
- **Responsive Elements**: Tables, buttons, links

### Browser Compatibility
- **Chromium**: Primary testing browser
- **Playwright Integration**: Cross-browser ready (Firefox, Safari available)

## Test Structure

### Test Organization
```
TestE2EBackOffice/
├── HomePage          # Homepage functionality
├── UserCRUD/         
│   ├── Create        # User creation flow
│   ├── Read          # Detail view validation
│   ├── Update        # Edit functionality
│   └── Delete        # Deletion workflow
├── ProductCRUD/      # Product operations
├── CategoryCRUD/     # Category operations
└── Navigation/       # UI navigation testing
```

### HTMX Integration
The tests properly handle HTMX asynchronous requests:

```go
func waitForHTMXRequest(page playwright.Page, timeout time.Duration) error {
    // Wait for HTMX request class to be removed
    return page.WaitForFunction(
        "() => !document.body.classList.contains('htmx-request')", 
        playwright.PageWaitForFunctionOptions{
            Timeout: playwright.Float(float64(timeout.Milliseconds())),
        },
    )
}
```

### Test Data Management
- **Self-Contained**: Tests create and clean up their own data
- **Isolated**: Each test creates unique test records
- **Reliable**: No dependency on existing demo data

## Advanced Usage

### Running Individual Tests

```bash
# Run specific test function
go test -run TestE2EBackOffice/UserCRUD -v

# Run with custom parameters
go run e2e_test.go -port=8080 -headless=false -slow-mo=200ms
```

### Debugging Failed Tests

```bash
# Run with visible browser and slow motion
./run_e2e_tests.sh --headless false --slow-mo 1s --no-cleanup

# Check demo application logs
cat demo_output.log
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Run E2E Tests
  run: |
    cd e2e_testing
    ./run_e2e_tests.sh --headless true --timeout 60s
  env:
    BACKOFFICE_E2E_PORT: 8080
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   ./run_e2e_tests.sh --port 8081
   ```

2. **Playwright Not Installed**
   ```bash
   go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium
   ```

3. **Demo App Fails to Start**
   - Check `demo_output.log` for errors
   - Ensure Go and dependencies are properly installed
   - Verify demo directory exists at `../examples/gorm-example/`

4. **Tests Timeout**
   - Increase timeout: `--timeout 60s`
   - Increase startup wait: `--wait 10`
   - Check system performance

5. **HTMX Wait Failures**
   - Usually indicates application performance issues
   - Try increasing `--slow-mo` duration
   - Check browser developer tools for JavaScript errors

### Development Tips

- Use `--headless false` to watch tests run in real time
- Use `--slow-mo` to slow down operations for debugging
- Use `--no-cleanup` to inspect the demo application after tests
- Check `demo_output.log` for backend issues

## Module Independence

This module is completely independent from the main BackOffice library:

```go
// Main library go.mod - NO playwright dependency
module backoffice
require (
    github.com/a-h/templ v0.3.924
    gorm.io/driver/sqlite v1.6.0
    gorm.io/gorm v1.30.1
)

// E2E testing go.mod - playwright isolated here
module backoffice-e2e-testing
require (
    github.com/playwright-community/playwright-go v0.5200.0
)
```

This ensures that:
- Library users never see test dependencies
- Main library `go.mod` stays clean
- Testing capabilities are fully isolated
- Build times for library users are optimized

## Contributing

When adding new features to BackOffice:

1. Add corresponding E2E tests to cover the new functionality
2. Update this README if new test patterns are introduced  
3. Ensure tests are self-contained and clean up after themselves
4. Test both success and failure scenarios where applicable

The E2E test suite serves as the ground truth for BackOffice functionality and should comprehensively cover all user-facing features.