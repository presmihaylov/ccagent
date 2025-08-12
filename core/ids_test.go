package core

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "valid prefix",
			prefix: "u",
		},
		{
			name:   "valid multi-character prefix",
			prefix: "si",
		},
		{
			name:   "uppercase prefix gets lowercased",
			prefix: "USER",
		},
		{
			name:   "prefix with spaces gets trimmed",
			prefix: "  job  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewID(tt.prefix)

			// Check the format: prefix_ULID
			expectedPrefix := strings.ToLower(strings.TrimSpace(tt.prefix)) + "_"
			if !strings.HasPrefix(got, expectedPrefix) {
				t.Errorf("NewID() = %v, want prefix %v", got, expectedPrefix)
			}

			// Check ULID pattern: 26 characters, base32 encoded
			ulidPart := strings.TrimPrefix(got, expectedPrefix)
			if len(ulidPart) != 26 {
				t.Errorf("NewID() ULID part length = %v, want 26", len(ulidPart))
			}

			// Check ULID format using regex (base32 characters: 0-9, A-Z excluding I, L, O, U)
			ulidPattern := regexp.MustCompile(`^[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
			if !ulidPattern.MatchString(ulidPart) {
				t.Errorf("NewID() ULID part %v does not match expected pattern", ulidPart)
			}

			// Verify format matches the example pattern: w_01G0EZ1XTM37C5X11SQTDNCTM1
			fullPattern := regexp.MustCompile(`^[a-z0-9]+_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
			if !fullPattern.MatchString(got) {
				t.Errorf("NewID() = %v does not match expected format pattern", got)
			}
		})
	}
}

func TestNewIDPanic(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "empty prefix panics",
			prefix: "",
		},
		{
			name:   "whitespace-only prefix panics",
			prefix: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewID() expected panic but got none")
				}
			}()

			NewID(tt.prefix)
		})
	}
}

func TestNewIDUniqueness(t *testing.T) {
	prefix := "test"
	ids := make(map[string]bool)
	numTests := 1000

	for i := 0; i < numTests; i++ {
		id := NewID(prefix)

		if ids[id] {
			t.Errorf("NewID() generated duplicate ID: %v", id)
		}
		ids[id] = true
	}

	if len(ids) != numTests {
		t.Errorf("Expected %d unique IDs, got %d", numTests, len(ids))
	}
}

func TestIsValidULID(t *testing.T) {
	// Generate a valid ULID to test with
	validID := NewID("test")

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid ULID with single char prefix",
			id:   validID,
			want: true,
		},
		{
			name: "valid ULID with multi char prefix",
			id:   NewID("user"),
			want: true,
		},
		{
			name: "valid ULID with numeric prefix",
			id:   NewID("v1"),
			want: true,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
		{
			name: "no underscore separator",
			id:   "test01G0EZ1XTM37C5X11SQTDNCTM1",
			want: false,
		},
		{
			name: "multiple underscores",
			id:   "test_01G0_EZ1XTM37C5X11SQTDNCTM1",
			want: false,
		},
		{
			name: "empty prefix",
			id:   "_01G0EZ1XTM37C5X11SQTDNCTM1",
			want: false,
		},
		{
			name: "uppercase prefix",
			id:   "USER_01G0EZ1XTM37C5X11SQTDNCTM1",
			want: false,
		},
		{
			name: "prefix with special chars",
			id:   "test-user_01G0EZ1XTM37C5X11SQTDNCTM1",
			want: false,
		},
		{
			name: "ULID part too short",
			id:   "test_01G0EZ1XTM37C5X11SQTDNCT",
			want: false,
		},
		{
			name: "ULID part too long",
			id:   "test_01G0EZ1XTM37C5X11SQTDNCTM12",
			want: false,
		},
		{
			name: "invalid ULID characters",
			id:   "test_01G0EZ1XTM37C5X11SQTDNCTL1",
			want: false,
		},
		{
			name: "lowercase ULID part",
			id:   "test_01g0ez1xtm37c5x11sqtdnctm1",
			want: false,
		},
		{
			name: "empty ULID part",
			id:   "test_",
			want: false,
		},
		{
			name: "just prefix",
			id:   "test",
			want: false,
		},
		{
			name: "just underscore",
			id:   "_",
			want: false,
		},
		{
			name: "random string",
			id:   "not-a-ulid-at-all",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidULID(tt.id)
			if got != tt.want {
				t.Errorf("IsValidULID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidULIDWithGeneratedIDs(t *testing.T) {
	// Test with various prefixes to ensure all generated IDs are valid
	prefixes := []string{"u", "si", "j", "a", "msg", "sess", "client", "test123", "v2"}

	for _, prefix := range prefixes {
		t.Run("prefix_"+prefix, func(t *testing.T) {
			id := NewID(prefix)
			if !IsValidULID(id) {
				t.Errorf("Generated ID %q should be valid but IsValidULID returned false", id)
			}
		})
	}
}
