# PUL-24: Decryption Error Handling Implementation

## Overview

Implemented comprehensive error handling for decryption operations, mapping Saltpack errors to Go Cloud error codes as specified in PUL-24 requirements.

## Error Mapping (Per PUL-24)

### Saltpack Errors → GCError Codes

| Saltpack Error | GCError Code | Description |
|----------------|--------------|-------------|
| `ErrNoDecryptionKey` | `NotFound` | No matching private key found in keyring |
| `ErrBadCiphertext` | `InvalidArgument` | Corrupted or tampered ciphertext (authentication failure) |
| `ErrBadTag` | `InvalidArgument` | Bad authentication tag (likely tampering) |
| `ErrBadFrame` | `InvalidArgument` | Malformed armor format |
| `ErrWrongMessageType` | `InvalidArgument` | Not an encryption message |
| `ErrBadVersion` | `InvalidArgument` | Unsupported Saltpack version |
| `context.DeadlineExceeded` | `DeadlineExceeded` | Operation timeout |
| `context.Canceled` | `Canceled` | Operation canceled |
| Other errors | `Internal` | Unexpected failure |

## Implementation Details

### 1. New Helper Functions

#### `classifyDecryptionError(err error) *KeeperError`
- Maps Saltpack errors to appropriate GCError codes
- Provides detailed, user-friendly error messages
- Preserves underlying error for debugging
- Uses `errors.As()` for type checking (handles wrapped errors)

#### `classifyContextError(err error, operation string) *KeeperError`
- Maps context errors (timeout, cancellation) to appropriate codes
- Includes operation name in error message for better context

### 2. Enhanced Methods

#### `Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)`
- Added context error checking before and after decryption
- Uses `classifyDecryptionError()` for all decryption failures
- Provides clear error messages distinguishing different failure modes
- Updated documentation with error handling behavior

#### `decryptStreaming(ciphertext []byte) ([]byte, error)`
- Uses `classifyDecryptionError()` for streaming decryption errors
- Consistent error handling with in-memory decryption

#### `ErrorCode(err error) gcerrors.ErrorCode`
- Enhanced to handle Saltpack errors directly (not just wrapped in KeeperError)
- Checks for `ErrNoDecryptionKey` sentinel error
- Checks for context errors (DeadlineExceeded, Canceled)
- Uses `errors.As()` to detect Saltpack error types

#### `ErrorAs(err error, target interface{}) bool`
- Enhanced to unwrap KeeperError and check underlying Saltpack errors
- Allows callers to extract specific Saltpack error types
- Maintains backward compatibility with existing error checking

### 3. Error Messages

All error messages are detailed and actionable:

- **ErrNoDecryptionKey**: "no decryption key found: ensure your Keybase account has access to decrypt this message"
- **ErrBadCiphertext**: "corrupted ciphertext: authentication failed at packet N (message may have been tampered with)"
- **ErrBadTag**: "corrupted ciphertext: bad authentication tag at packet N"
- **ErrBadFrame**: "malformed ciphertext: invalid armor format"
- **Context timeout**: "decryption timed out: operation took too long"
- **Context canceled**: "decryption canceled: operation was interrupted"

## Test Coverage

### New Tests

1. **TestDecryptionErrorMapping**
   - Tests all Saltpack error types
   - Tests context errors (DeadlineExceeded, Canceled)
   - Verifies correct error codes
   - Verifies error messages contain expected substrings

2. **TestClassifyContextError**
   - Tests deadline exceeded handling
   - Tests cancellation handling
   - Verifies correct error codes and messages

3. **TestErrorAsWithSaltpackErrors**
   - Tests unwrapping KeeperError to extract Saltpack errors
   - Verifies ErrorAs works with wrapped errors

### Test Results

All tests pass successfully:
- `TestDecryptionErrorMapping`: ✅ PASS
- `TestClassifyContextError`: ✅ PASS
- `TestErrorAsWithSaltpackErrors`: ✅ PASS
- `TestKeeperErrorCode`: ✅ PASS (updated)
- `TestKeeperErrorAs`: ✅ PASS (updated)
- All existing Keeper tests: ✅ PASS

## Code Changes

- **Files Modified**: 2
  - `keybase/keeper.go` (206 lines modified)
  - `keybase/keeper_test.go` (188 lines added)
- **Total Changes**: 394 lines

## Usage Example

```go
keeper, _ := keybase.NewKeeper(config)
plaintext, err := keeper.Decrypt(ctx, ciphertext)
if err != nil {
    // Check error code
    code := keeper.ErrorCode(err)
    
    switch code {
    case gcerrors.NotFound:
        // No matching private key - user needs to be added as recipient
        fmt.Println("You don't have permission to decrypt this message")
    
    case gcerrors.InvalidArgument:
        // Corrupted or tampered ciphertext
        var badCiphertext saltpack.ErrBadCiphertext
        if keeper.ErrorAs(err, &badCiphertext) {
            fmt.Printf("Message corrupted at packet %d\n", badCiphertext)
        }
    
    case gcerrors.DeadlineExceeded:
        // Timeout
        fmt.Println("Decryption timed out")
    
    case gcerrors.Canceled:
        // Canceled
        fmt.Println("Decryption was canceled")
    }
}
```

## Benefits

1. **Standards Compliance**: Maps to Go Cloud error codes as required by Pulumi
2. **Detailed Messages**: User-friendly error messages help diagnose issues
3. **Type Safety**: Callers can check specific error types using ErrorAs
4. **Context Awareness**: Properly handles timeout and cancellation scenarios
5. **Security**: Distinguishes between corrupted data and missing keys
6. **Debugging**: Preserves underlying errors for troubleshooting

## Compliance

This implementation fully satisfies PUL-24 requirements:
- ✅ Map `saltpack.ErrBadCiphertext` to GCError InvalidArgument
- ✅ Map `ErrNoDecryptionKey` to GCError NotFound
- ✅ Map timeout/network errors to appropriate codes (DeadlineExceeded/Canceled)
- ✅ Provide detailed error messages
- ✅ Comprehensive test coverage
