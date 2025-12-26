# API Integration Tests

This document describes the comprehensive integration tests implemented for the Keybase API client.

## Overview

The integration tests use mock HTTP servers to simulate the Keybase API and verify that the client correctly handles various scenarios including successful requests, error conditions, retry logic, and performance characteristics.

## Test Coverage

### Single and Multiple User Lookups

#### TestIntegrationSingleUserLookup
- **Purpose**: Verifies basic single-user lookup functionality
- **Validates**:
  - Correct HTTP method (GET)
  - Proper query parameters (usernames, fields)
  - Response parsing for single user
  - Public key and Key ID extraction
  - Realistic API response format

#### TestIntegrationMultipleUserLookup
- **Purpose**: Tests batch lookup of multiple users in one API call
- **Validates**:
  - Comma-separated username format in request
  - Response parsing for multiple users
  - All requested users are returned
  - Efficient batch processing

#### TestIntegrationLargeUserBatch
- **Purpose**: Tests handling of large batches (50 users)
- **Validates**:
  - Scalability of batch operations
  - Response parsing for large payloads
  - Memory efficiency

### Error Handling

#### TestIntegrationUserNotFound
- **Purpose**: Tests handling of non-existent users
- **Validates**:
  - Empty response array handling
  - Appropriate error message
  - No panic on missing users

#### TestIntegrationServerError500
- **Purpose**: Tests 500 Internal Server Error handling
- **Validates**:
  - APIError type returned
  - Correct status code captured
  - Error marked as temporary
  - Retry behavior (3 total attempts with 2 retries)

#### TestIntegration400NoRetry
- **Purpose**: Verifies that 4xx errors (except 429) are not retried
- **Validates**:
  - Single request made (no retries)
  - Error not marked as temporary
  - Proper APIError handling

#### TestIntegrationPartialSuccess
- **Purpose**: Tests when some requested users are found but not all
- **Validates**:
  - Partial result detection
  - Appropriate error for missing users
  - Correct error message format

#### TestIntegrationMalformedJSON
- **Purpose**: Tests handling of invalid JSON responses
- **Validates**:
  - JSON parsing error handling
  - No panic on malformed data
  - Descriptive error message

#### TestIntegrationAPIErrorStatus
- **Purpose**: Tests API-level error codes (status.code != 0)
- **Validates**:
  - Status field parsing
  - Error message extraction
  - Proper error propagation

### Rate Limiting and Retries

#### TestIntegrationRateLimiting429
- **Purpose**: Tests rate limiting (429 Too Many Requests) handling
- **Validates**:
  - Retry on 429 status
  - Eventual success after retries
  - Correct number of attempts
  - 429 treated as temporary error

#### TestIntegrationExponentialBackoff
- **Purpose**: Verifies exponential backoff retry strategy
- **Validates**:
  - First retry delay: ~50ms (1x base delay)
  - Second retry delay: ~100ms (2x base delay)
  - Third retry delay: ~200ms (4x base delay)
  - Timing tolerances: ±30ms for test jitter
  - Total of 4 requests (1 initial + 3 retries)

### Timeouts and Cancellation

#### TestIntegrationNetworkTimeout
- **Purpose**: Tests client timeout behavior
- **Validates**:
  - Request timeout enforced
  - Timeout error returned
  - Error marked as temporary
  - APIError type used

#### TestIntegrationContextCancellation
- **Purpose**: Tests context cancellation during requests
- **Validates**:
  - Context deadline respected
  - Proper context error returned
  - Early termination on cancellation
  - No resource leaks

### Client Behavior

#### TestIntegrationUserAgentHeader
- **Purpose**: Verifies proper User-Agent header
- **Validates**:
  - Header value: "pulumi-keybase-encryption/1.0"
  - Header sent on all requests
  - Correct identification to API

## Test Architecture

### Mock Server Pattern

All integration tests use `httptest.NewServer()` to create mock HTTP servers that simulate the Keybase API. This provides:

- **Isolation**: Tests don't depend on external services
- **Control**: Full control over responses and timing
- **Speed**: Fast test execution without network I/O
- **Reliability**: No flakiness from network issues

### Request Validation

Each test validates:
1. **HTTP Method**: Ensures GET is used
2. **Query Parameters**: Verifies usernames and fields parameters
3. **Headers**: Checks User-Agent and other headers
4. **Request Count**: Tracks number of API calls for retry verification

### Response Simulation

Mock servers provide:
1. **Realistic Payloads**: JSON responses matching actual Keybase API format
2. **Error Conditions**: HTTP error codes (400, 429, 500)
3. **Timing Control**: Delays to test timeouts and backoff
4. **State Tracking**: Counters to verify retry behavior

## Running the Tests

```bash
# Run all integration tests
go test -v ./keybase/api/... -run TestIntegration

# Run specific integration test
go test -v ./keybase/api/... -run TestIntegrationRateLimiting429

# Run with race detector
go test -race ./keybase/api/...

# Run with coverage
go test -cover ./keybase/api/...
```

## Test Metrics

| Metric | Value |
|--------|-------|
| Total Integration Tests | 14 |
| Total Test Time | ~0.8 seconds |
| Coverage Areas | API calls, errors, retries, timeouts, caching |
| Mock Servers Used | 14 (one per test) |

## Integration with Cache Tests

These API integration tests complement the cache integration tests in `keybase/cache/integration_test.go`. Together they provide:

- API client functionality (this file)
- Cache + API integration (`cache/integration_test.go`)
- End-to-end workflows with caching behavior

## Future Enhancements

Potential additions:
1. Performance benchmarks for API calls
2. Stress tests with thousands of users
3. Concurrent request tests
4. Real API integration tests (with live Keybase API)
5. Chaos testing with intermittent failures
