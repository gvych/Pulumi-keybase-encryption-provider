package crypto

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// mockReader is a mock io.Reader for testing
type mockReader struct {
	data []byte
	err  error
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

// TestNewEphemeralKeyCreator tests the default constructor
func TestNewEphemeralKeyCreator(t *testing.T) {
	creator := NewEphemeralKeyCreator()
	
	if creator == nil {
		t.Fatal("NewEphemeralKeyCreator() returned nil")
	}
	
	if creator.randReader == nil {
		t.Error("NewEphemeralKeyCreator() created creator with nil randReader")
	}
}

// TestNewEphemeralKeyCreatorWithReader tests the constructor with custom reader
func TestNewEphemeralKeyCreatorWithReader(t *testing.T) {
	tests := []struct {
		name     string
		reader   io.Reader
		wantNil  bool
	}{
		{
			name:    "with valid reader",
			reader:  &mockReader{data: make([]byte, 1024)},
			wantNil: false,
		},
		{
			name:    "with nil reader (should use default)",
			reader:  nil,
			wantNil: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := NewEphemeralKeyCreatorWithReader(tt.reader)
			
			if creator == nil {
				t.Fatal("NewEphemeralKeyCreatorWithReader() returned nil")
			}
			
			if creator.randReader == nil {
				t.Error("NewEphemeralKeyCreatorWithReader() created creator with nil randReader")
			}
		})
	}
}

// TestGenerateKey tests basic key generation
func TestGenerateKey(t *testing.T) {
	creator := NewEphemeralKeyCreator()
	
	pair, err := creator.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}
	
	if pair == nil {
		t.Fatal("GenerateKey() returned nil pair")
	}
	
	// Check that keys are not all zeros
	allZerosPub := true
	for _, b := range pair.PublicKey {
		if b != 0 {
			allZerosPub = false
			break
		}
	}
	if allZerosPub {
		t.Error("GenerateKey() generated all-zeros public key")
	}
	
	allZerosSec := true
	for _, b := range pair.SecretKey {
		if b != 0 {
			allZerosSec = false
			break
		}
	}
	if allZerosSec {
		t.Error("GenerateKey() generated all-zeros secret key")
	}
}

// TestGenerateKey_Uniqueness tests that generated keys are unique
func TestGenerateKey_Uniqueness(t *testing.T) {
	creator := NewEphemeralKeyCreator()
	
	pair1, err := creator.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}
	
	pair2, err := creator.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v, want nil", err)
	}
	
	// Keys should be different
	if bytes.Equal(pair1.PublicKey[:], pair2.PublicKey[:]) {
		t.Error("GenerateKey() generated identical public keys")
	}
	
	if bytes.Equal(pair1.SecretKey[:], pair2.SecretKey[:]) {
		t.Error("GenerateKey() generated identical secret keys")
	}
}

// TestGenerateKey_InsufficientEntropy tests error handling for entropy issues
func TestGenerateKey_InsufficientEntropy(t *testing.T) {
	// Create a reader that returns an entropy error
	mockErr := errors.New("insufficient entropy available")
	creator := NewEphemeralKeyCreatorWithReader(&mockReader{err: mockErr})
	
	_, err := creator.GenerateKey()
	if err == nil {
		t.Fatal("GenerateKey() error = nil, want error")
	}
	
	// Should wrap the error appropriately
	if !errors.Is(err, ErrInsufficientEntropy) && !errors.Is(err, ErrKeyGenerationFailed) {
		t.Errorf("GenerateKey() error = %v, want ErrInsufficientEntropy or ErrKeyGenerationFailed", err)
	}
}

// TestGenerateKey_NilReader tests error handling for nil reader
func TestGenerateKey_NilReader(t *testing.T) {
	creator := &EphemeralKeyCreator{
		randReader: nil,
	}
	
	_, err := creator.GenerateKey()
	if err == nil {
		t.Fatal("GenerateKey() with nil reader error = nil, want error")
	}
	
	if !errors.Is(err, ErrKeyGenerationFailed) {
		t.Errorf("GenerateKey() error = %v, want ErrKeyGenerationFailed", err)
	}
}

// TestGenerateKeys tests batch key generation
func TestGenerateKeys(t *testing.T) {
	creator := NewEphemeralKeyCreator()
	
	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{
			name:    "generate 1 key",
			count:   1,
			wantErr: false,
		},
		{
			name:    "generate 5 keys",
			count:   5,
			wantErr: false,
		},
		{
			name:    "generate 10 keys",
			count:   10,
			wantErr: false,
		},
		{
			name:    "zero count",
			count:   0,
			wantErr: true,
		},
		{
			name:    "negative count",
			count:   -1,
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs, err := creator.GenerateKeys(tt.count)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			if len(pairs) != tt.count {
				t.Errorf("GenerateKeys() returned %d pairs, want %d", len(pairs), tt.count)
			}
			
			// Check that all keys are unique
			for i := 0; i < len(pairs); i++ {
				for j := i + 1; j < len(pairs); j++ {
					if bytes.Equal(pairs[i].PublicKey[:], pairs[j].PublicKey[:]) {
						t.Errorf("GenerateKeys() generated duplicate public keys at indices %d and %d", i, j)
					}
					if bytes.Equal(pairs[i].SecretKey[:], pairs[j].SecretKey[:]) {
						t.Errorf("GenerateKeys() generated duplicate secret keys at indices %d and %d", i, j)
					}
				}
			}
		})
	}
}

// TestGenerateKeys_ErrorPropagation tests that errors in batch generation are propagated
func TestGenerateKeys_ErrorPropagation(t *testing.T) {
	// Create a reader that will fail after a few reads
	mockErr := errors.New("entropy failure")
	creator := NewEphemeralKeyCreatorWithReader(&mockReader{err: mockErr})
	
	_, err := creator.GenerateKeys(5)
	if err == nil {
		t.Fatal("GenerateKeys() with failing reader error = nil, want error")
	}
}

// TestIsEntropyError tests the entropy error detection
func TestIsEntropyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "entropy error",
			err:  errors.New("insufficient entropy"),
			want: true,
		},
		{
			name: "random error",
			err:  errors.New("random device not available"),
			want: true,
		},
		{
			name: "urandom error",
			err:  errors.New("cannot read from /dev/urandom"),
			want: true,
		},
		{
			name: "RNG error",
			err:  errors.New("RNG initialization failed"),
			want: true,
		},
		{
			name: "PRNG error",
			err:  errors.New("PRNG seeding error"),
			want: true,
		},
		{
			name: "uppercase entropy",
			err:  errors.New("ENTROPY ERROR"),
			want: true,
		},
		{
			name: "mixed case",
			err:  errors.New("Random Number Generator failure"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("network timeout"),
			want: false,
		},
		{
			name: "empty error",
			err:  errors.New(""),
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEntropyError(tt.err)
			if got != tt.want {
				t.Errorf("isEntropyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestContains tests the contains helper function
func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "simple match",
			s:      "hello world",
			substr: "world",
			want:   true,
		},
		{
			name:   "no match",
			s:      "hello world",
			substr: "foo",
			want:   false,
		},
		{
			name:   "case insensitive match",
			s:      "Hello World",
			substr: "world",
			want:   true,
		},
		{
			name:   "case insensitive uppercase",
			s:      "hello world",
			substr: "WORLD",
			want:   true,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "test",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "test",
			substr: "",
			want:   false,
		},
		{
			name:   "both empty",
			s:      "",
			substr: "",
			want:   false,
		},
		{
			name:   "substring longer than string",
			s:      "hi",
			substr: "hello",
			want:   false,
		},
		{
			name:   "exact match",
			s:      "test",
			substr: "test",
			want:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// TestRawPublicKey_Bytes tests the Bytes method
func TestRawPublicKey_Bytes(t *testing.T) {
	var key RawPublicKey
	for i := range key {
		key[i] = byte(i)
	}
	
	bytes := key.Bytes()
	if len(bytes) != 32 {
		t.Errorf("RawPublicKey.Bytes() length = %d, want 32", len(bytes))
	}
	
	for i := range key {
		if bytes[i] != byte(i) {
			t.Errorf("RawPublicKey.Bytes()[%d] = %d, want %d", i, bytes[i], byte(i))
		}
	}
}

// TestRawSecretKey_Bytes tests the Bytes method
func TestRawSecretKey_Bytes(t *testing.T) {
	var key RawSecretKey
	for i := range key {
		key[i] = byte(i)
	}
	
	bytes := key.Bytes()
	if len(bytes) != 32 {
		t.Errorf("RawSecretKey.Bytes() length = %d, want 32", len(bytes))
	}
	
	for i := range key {
		if bytes[i] != byte(i) {
			t.Errorf("RawSecretKey.Bytes()[%d] = %d, want %d", i, bytes[i], byte(i))
		}
	}
}

// TestRawSecretKey_Zero tests that secret keys can be zeroed
func TestRawSecretKey_Zero(t *testing.T) {
	var key RawSecretKey
	for i := range key {
		key[i] = byte(i)
	}
	
	key.Zero()
	
	for i := range key {
		if key[i] != 0 {
			t.Errorf("RawSecretKey.Zero() did not zero byte at index %d, got %d", i, key[i])
		}
	}
}

// TestEphemeralKeyPair_Zero tests that key pairs can be zeroed
func TestEphemeralKeyPair_Zero(t *testing.T) {
	creator := NewEphemeralKeyCreator()
	pair, err := creator.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	
	// Verify keys are not zero before
	allZerosPub := true
	for _, b := range pair.PublicKey {
		if b != 0 {
			allZerosPub = false
			break
		}
	}
	if allZerosPub {
		t.Error("Generated key pair has all-zeros public key before Zero()")
	}
	
	allZerosSec := true
	for _, b := range pair.SecretKey {
		if b != 0 {
			allZerosSec = false
			break
		}
	}
	if allZerosSec {
		t.Error("Generated key pair has all-zeros secret key before Zero()")
	}
	
	// Zero the pair
	pair.Zero()
	
	// Verify keys are zero after
	for i := range pair.PublicKey {
		if pair.PublicKey[i] != 0 {
			t.Errorf("EphemeralKeyPair.Zero() did not zero public key byte at index %d", i)
		}
	}
	
	for i := range pair.SecretKey {
		if pair.SecretKey[i] != 0 {
			t.Errorf("EphemeralKeyPair.Zero() did not zero secret key byte at index %d", i)
		}
	}
}

// TestEphemeralKeyPair_Zero_Nil tests that zeroing nil pair doesn't panic
func TestEphemeralKeyPair_Zero_Nil(t *testing.T) {
	var pair *EphemeralKeyPair
	
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("EphemeralKeyPair.Zero() panicked with nil pair: %v", r)
		}
	}()
	
	pair.Zero()
}

// BenchmarkGenerateKey benchmarks single key generation
func BenchmarkGenerateKey(b *testing.B) {
	creator := NewEphemeralKeyCreator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := creator.GenerateKey()
		if err != nil {
			b.Fatalf("GenerateKey() error = %v", err)
		}
	}
}

// BenchmarkGenerateKeys benchmarks batch key generation
func BenchmarkGenerateKeys(b *testing.B) {
	creator := NewEphemeralKeyCreator()
	
	benchmarks := []struct {
		name  string
		count int
	}{
		{"1_key", 1},
		{"10_keys", 10},
		{"100_keys", 100},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := creator.GenerateKeys(bm.count)
				if err != nil {
					b.Fatalf("GenerateKeys() error = %v", err)
				}
			}
		})
	}
}
