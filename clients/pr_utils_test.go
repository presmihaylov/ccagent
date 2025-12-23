package clients

import (
	"strings"
	"testing"
)

func TestValidateAndTruncatePRTitle_WithinLimit(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
	}{
		{
			name:        "short title",
			title:       "feat: add new feature",
			description: "This is a test description",
		},
		{
			name:        "empty title",
			title:       "",
			description: "Description",
		},
		{
			name:        "exactly at limit",
			title:       strings.Repeat("a", MaxGitHubPRTitleLength),
			description: "Description",
		},
		{
			name:        "one character",
			title:       "x",
			description: "Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAndTruncatePRTitle(tt.title, tt.description)

			// Title should be unchanged
			if result.Title != tt.title {
				t.Errorf("Expected title to be unchanged. Got: %q, Want: %q", result.Title, tt.title)
			}

			// Description prefix should be empty
			if result.DescriptionPrefix != "" {
				t.Errorf("Expected empty description prefix. Got: %q", result.DescriptionPrefix)
			}
		})
	}
}

func TestValidateAndTruncatePRTitle_ExceedsLimit(t *testing.T) {
	tests := []struct {
		name                      string
		title                     string
		description               string
		expectedTitleLength       int
		expectedTitleSuffix       string
		expectedOverflowInPrefix  string
		expectedSeparatorInPrefix bool
	}{
		{
			name:                      "one character over limit",
			title:                     strings.Repeat("a", MaxGitHubPRTitleLength+1),
			description:               "Test description",
			expectedTitleLength:       MaxGitHubPRTitleLength,
			expectedTitleSuffix:       "...",
			expectedOverflowInPrefix:  "a",
			expectedSeparatorInPrefix: true,
		},
		{
			name:                      "many characters over limit",
			title:                     strings.Repeat("b", MaxGitHubPRTitleLength+50),
			description:               "Test description",
			expectedTitleLength:       MaxGitHubPRTitleLength,
			expectedTitleSuffix:       "...",
			expectedOverflowInPrefix:  strings.Repeat("b", 50),
			expectedSeparatorInPrefix: true,
		},
		{
			name:                      "realistic long title",
			title:                     "feat: implement comprehensive user authentication system with OAuth2, JWT tokens, session management, role-based access control, password reset functionality, email verification, two-factor authentication, and extensive security features including rate limiting, brute force protection, and audit logging for compliance requirements",
			description:               "Detailed implementation notes",
			expectedTitleLength:       MaxGitHubPRTitleLength,
			expectedTitleSuffix:       "...",
			expectedOverflowInPrefix:  " and audit logging for compliance requirements",
			expectedSeparatorInPrefix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAndTruncatePRTitle(tt.title, tt.description)

			// Check title length
			if len(result.Title) != tt.expectedTitleLength {
				t.Errorf("Expected title length %d, got %d", tt.expectedTitleLength, len(result.Title))
			}

			// Check title ends with ellipsis
			if !strings.HasSuffix(result.Title, tt.expectedTitleSuffix) {
				t.Errorf("Expected title to end with %q, got: %q", tt.expectedTitleSuffix, result.Title)
			}

			// Check description prefix is not empty
			if result.DescriptionPrefix == "" {
				t.Errorf("Expected non-empty description prefix for title exceeding limit")
			}

			// Check overflow text is in the prefix
			if !strings.Contains(result.DescriptionPrefix, tt.expectedOverflowInPrefix) {
				t.Errorf("Expected description prefix to contain overflow text %q, got: %q",
					tt.expectedOverflowInPrefix, result.DescriptionPrefix)
			}

			// Check separator is present
			if tt.expectedSeparatorInPrefix && !strings.Contains(result.DescriptionPrefix, "---") {
				t.Errorf("Expected description prefix to contain separator '---', got: %q", result.DescriptionPrefix)
			}

			// Verify that title + overflow equals original title (minus ellipsis)
			titleWithoutEllipsis := strings.TrimSuffix(result.Title, "...")
			reconstructed := titleWithoutEllipsis + strings.Split(result.DescriptionPrefix, "\n\n---\n\n")[0]
			if reconstructed != tt.title {
				t.Errorf("Title reconstruction failed. Original: %q, Reconstructed: %q", tt.title, reconstructed)
			}
		})
	}
}

func TestValidateAndTruncatePRTitle_ExactBoundaries(t *testing.T) {
	tests := []struct {
		name                string
		titleLength         int
		expectTruncation    bool
		expectedTitleLength int
	}{
		{
			name:                "at limit - 1",
			titleLength:         MaxGitHubPRTitleLength - 1,
			expectTruncation:    false,
			expectedTitleLength: MaxGitHubPRTitleLength - 1,
		},
		{
			name:                "at limit",
			titleLength:         MaxGitHubPRTitleLength,
			expectTruncation:    false,
			expectedTitleLength: MaxGitHubPRTitleLength,
		},
		{
			name:                "at limit + 1",
			titleLength:         MaxGitHubPRTitleLength + 1,
			expectTruncation:    true,
			expectedTitleLength: MaxGitHubPRTitleLength,
		},
		{
			name:                "at limit + 10",
			titleLength:         MaxGitHubPRTitleLength + 10,
			expectTruncation:    true,
			expectedTitleLength: MaxGitHubPRTitleLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := strings.Repeat("x", tt.titleLength)
			result := ValidateAndTruncatePRTitle(title, "description")

			if len(result.Title) != tt.expectedTitleLength {
				t.Errorf("Expected title length %d, got %d", tt.expectedTitleLength, len(result.Title))
			}

			if tt.expectTruncation {
				if result.DescriptionPrefix == "" {
					t.Errorf("Expected non-empty description prefix for truncated title")
				}
				if !strings.HasSuffix(result.Title, "...") {
					t.Errorf("Expected truncated title to end with '...'")
				}
			} else {
				if result.DescriptionPrefix != "" {
					t.Errorf("Expected empty description prefix for non-truncated title")
				}
				if result.Title != title {
					t.Errorf("Expected title to be unchanged. Got: %q, Want: %q", result.Title, title)
				}
			}
		})
	}
}

func TestValidateAndTruncatePRTitle_DescriptionPrefixFormat(t *testing.T) {
	// Create a title that exceeds the limit
	overflowAmount := 20
	longTitle := strings.Repeat("a", MaxGitHubPRTitleLength+overflowAmount)
	result := ValidateAndTruncatePRTitle(longTitle, "Original description")

	// Check prefix format: should be "overflow\n\n---\n\n"
	parts := strings.Split(result.DescriptionPrefix, "\n\n---\n\n")
	if len(parts) != 2 {
		t.Errorf("Expected description prefix to have format 'overflow\\n\\n---\\n\\n', got: %q", result.DescriptionPrefix)
	}

	// First part should be the overflow text
	overflow := parts[0]
	// The overflow should include the 3 characters taken by "..." plus the extra characters
	expectedOverflowLength := overflowAmount + 3
	if len(overflow) != expectedOverflowLength {
		t.Errorf("Expected overflow text length %d, got %d", expectedOverflowLength, len(overflow))
	}

	// Second part should be empty (nothing after separator)
	if parts[1] != "" {
		t.Errorf("Expected nothing after separator, got: %q", parts[1])
	}
}

func TestValidateAndTruncatePRTitle_UnicodeCharacters(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
	}{
		{
			name:        "emoji in short title",
			title:       "feat: add 🎉 celebration feature",
			description: "Description",
		},
		{
			name:        "emoji in long title",
			title:       strings.Repeat("🎉", MaxGitHubPRTitleLength/4+1), // Each emoji is 4 bytes
			description: "Description",
		},
		{
			name:        "mixed unicode",
			title:       "fix: résoudre le problème avec les caractères spéciaux",
			description: "Description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAndTruncatePRTitle(tt.title, tt.description)

			// Basic validation: result should have a title
			if result.Title == "" {
				t.Errorf("Expected non-empty title")
			}

			// Title should not exceed limit
			if len(result.Title) > MaxGitHubPRTitleLength {
				t.Errorf("Title exceeds limit. Length: %d, Max: %d", len(result.Title), MaxGitHubPRTitleLength)
			}

			// If original was short, should be unchanged
			if len(tt.title) <= MaxGitHubPRTitleLength {
				if result.Title != tt.title {
					t.Errorf("Expected title unchanged for short unicode title")
				}
				if result.DescriptionPrefix != "" {
					t.Errorf("Expected empty description prefix for short title")
				}
			}
		})
	}
}

func TestValidateAndTruncatePRTitle_EmptyDescription(t *testing.T) {
	// Test that validation works even with empty description
	longTitle := strings.Repeat("a", MaxGitHubPRTitleLength+10)
	result := ValidateAndTruncatePRTitle(longTitle, "")

	if len(result.Title) != MaxGitHubPRTitleLength {
		t.Errorf("Expected title length %d, got %d", MaxGitHubPRTitleLength, len(result.Title))
	}

	if result.DescriptionPrefix == "" {
		t.Errorf("Expected non-empty description prefix")
	}

	// The description prefix should still be properly formatted
	if !strings.Contains(result.DescriptionPrefix, "---") {
		t.Errorf("Expected separator in description prefix")
	}
}

func TestMaxGitHubPRTitleLength_Constant(t *testing.T) {
	// Verify the constant is set to GitHub's actual limit
	expectedLimit := 256
	if MaxGitHubPRTitleLength != expectedLimit {
		t.Errorf("MaxGitHubPRTitleLength should be %d, got %d", expectedLimit, MaxGitHubPRTitleLength)
	}
}

func TestValidateAndTruncatePRTitle_ConventionalCommitFormats(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "feat with scope",
			title: "feat(api): " + strings.Repeat("x", MaxGitHubPRTitleLength),
		},
		{
			name:  "fix with breaking change",
			title: "fix!: " + strings.Repeat("x", MaxGitHubPRTitleLength),
		},
		{
			name:  "chore with scope and breaking",
			title: "chore(deps)!: " + strings.Repeat("x", MaxGitHubPRTitleLength),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAndTruncatePRTitle(tt.title, "Description")

			// Should be truncated since it exceeds limit
			if len(result.Title) != MaxGitHubPRTitleLength {
				t.Errorf("Expected title length %d, got %d", MaxGitHubPRTitleLength, len(result.Title))
			}

			// Should preserve the conventional commit prefix
			prefixes := []string{"feat(api):", "fix!:", "chore(deps)!:"}
			foundPrefix := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(result.Title, prefix) {
					foundPrefix = true
					break
				}
			}

			if !foundPrefix {
				t.Errorf("Expected title to preserve conventional commit prefix, got: %q", result.Title[:20])
			}

			// Should have overflow in description prefix
			if result.DescriptionPrefix == "" {
				t.Errorf("Expected non-empty description prefix")
			}
		})
	}
}
