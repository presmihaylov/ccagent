package clients

import "strings"

const (
	// MaxGitHubPRTitleLength is the maximum character limit for GitHub PR titles
	MaxGitHubPRTitleLength = 256
)

// PRTitleValidationResult contains the validated title and optional overflow description
type PRTitleValidationResult struct {
	// Title is the validated PR title (truncated if necessary)
	Title string
	// DescriptionPrefix is the overflow text that should be added to the top of the description
	// This is empty if the title didn't need truncation
	DescriptionPrefix string
}

// ValidateAndTruncatePRTitle ensures the PR title fits within GitHub's 256 character limit.
// If the title exceeds the limit, it truncates it with "..." and returns the overflow
// text to be prepended to the PR description.
//
// Parameters:
//   - title: The proposed PR title
//   - description: The PR description (used to prepend overflow text if needed)
//
// Returns:
//   - PRTitleValidationResult with validated title and optional description prefix
func ValidateAndTruncatePRTitle(title, description string) PRTitleValidationResult {
	// If title is within limit, return as-is
	if len(title) <= MaxGitHubPRTitleLength {
		return PRTitleValidationResult{
			Title:             title,
			DescriptionPrefix: "",
		}
	}

	// Title exceeds limit - need to truncate
	// Reserve 3 characters for "..."
	truncateAt := MaxGitHubPRTitleLength - 3

	// Truncate at the last complete character before the limit
	truncatedTitle := title[:truncateAt] + "..."

	// Extract the overflow text (everything after truncation point)
	overflowText := title[truncateAt:]

	// Create description prefix with the overflow and separator
	var descriptionPrefix strings.Builder
	descriptionPrefix.WriteString(overflowText)
	descriptionPrefix.WriteString("\n\n---\n\n")

	return PRTitleValidationResult{
		Title:             truncatedTitle,
		DescriptionPrefix: descriptionPrefix.String(),
	}
}
