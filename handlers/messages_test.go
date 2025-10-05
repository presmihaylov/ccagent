package handlers

import "testing"

func TestStripAccessTokenFromURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with x-access-token",
			input:    "https://x-access-token:ghs_1234567890abcdefghijklmnop@github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "URL without x-access-token",
			input:    "https://github.com/owner/repo",
			expected: "https://github.com/owner/repo",
		},
		{
			name:     "Empty URL",
			input:    "",
			expected: "",
		},
		{
			name:     "URL with x-access-token and path",
			input:    "https://x-access-token:token123@github.com/owner/repo/commit/abc123",
			expected: "https://github.com/owner/repo/commit/abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripAccessTokenFromURL(tt.input)
			if result != tt.expected {
				t.Errorf("stripAccessTokenFromURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
