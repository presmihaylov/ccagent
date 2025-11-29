package usecases

import "fmt"

// CommitMessageGenerationPrompt creates a prompt for Claude to generate commit messages
func CommitMessageGenerationPrompt(branchName string) string {
	return `Generate a commit message for the changes we just made in our conversation.

CRITICAL INSTRUCTIONS:
1. You MUST respond with ONLY the commit message text
2. NO explanations, NO additional text, NO formatting markup
3. NO "Here is the commit message:" or similar phrases
4. Maximum 50 characters total (STRICT LIMIT)
5. Start with action verb (Add, Fix, Update, etc.)
6. Use imperative mood
7. Base it on what we actually worked on in this conversation

FORMAT EXAMPLE:
Fix user authentication validation

YOUR RESPONSE MUST BE THE COMMIT MESSAGE ONLY.`
}

// PRTitleGenerationPrompt creates a prompt for Claude to generate PR titles
func PRTitleGenerationPrompt(branchName string) string {
	return `Generate a SHORT pull request title for the work we completed in this conversation.

Follow these strict rules:
- Maximum 50 characters (STRICT LIMIT)
- Use conventional commits format: "type: description"
- Choose appropriate type: feat, fix, docs, chore, refactor, test, perf, ci, build, style
- Be concise and specific in description
- No unnecessary words or phrases
- Don't mention "Claude", "agent", or implementation details
- Base the title on what we actually worked on in our conversation

Type Guidelines:
- feat: new features or functionality
- fix: bug fixes
- docs: documentation changes
- refactor: code restructuring without behavior change
- chore: maintenance, dependencies, configuration
- test: adding or fixing tests
- perf: performance improvements
- ci: CI/CD pipeline changes
- build: build system or external dependencies
- style: formatting, missing semicolons (no code change)

Examples:
- "fix: resolve error handling in message processor"
- "feat: add user authentication middleware" 
- "docs: update API response format documentation"
- "chore: bump dependency versions"

CRITICAL: Your response must contain ONLY the PR title text. Do not include:
- Any explanations or reasoning
- Quotes around the title
- "Here is the title:" or similar phrases
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the short title text, nothing else.`
}

// PRDescriptionGenerationPrompt creates a prompt for Claude to generate PR descriptions
func PRDescriptionGenerationPrompt(branchName string, prTemplate string) string {
	if prTemplate != "" {
		// Template exists - use it as a guideline
		return fmt.Sprintf(`Generate a pull request description for the work we completed in this conversation.

The repository has a PR template that you should follow as a guideline:

--- PR TEMPLATE ---
%s
--- END TEMPLATE ---

INSTRUCTIONS:
- Follow the structure and sections defined in the template above
- Fill in all relevant sections based on our conversation
- Keep it professional and concise
- If the template has checkboxes (- [ ]), include them in your response
- If the template has placeholders or instructions (like "describe your changes"), replace them with actual content
- Base your description on what we actually worked on in our conversation
- Use proper markdown formatting

IMPORTANT:
- Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.
- Do NOT include any introductory text like "Here is your description"

CRITICAL: Your response must contain ONLY the PR description in markdown format. Do not include:
- Any explanations or reasoning about your response
- "Here is the description:" or similar phrases
- Any text before or after the description
- Any commentary about the changes
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the PR description in markdown format, nothing else.`, prTemplate)
	}

	// No template - use default format
	return `Generate a concise pull request description for the work we completed in this conversation.

Format:
- ## Summary: High-level overview of what changed (2-3 bullet points max)
- ## Why: Brief explanation of the motivation/reasoning behind the change

Keep it professional but brief. Focus on WHAT changed at a high level and WHY the change was necessary, not detailed implementation specifics.

Use proper markdown formatting. Base it on what we actually worked on in our conversation.

IMPORTANT:
- Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.
- Keep the summary concise - avoid listing every single file or detailed code changes
- Focus on the business/functional purpose of the changes
- Do NOT include any introductory text like "Here is your description"

CRITICAL: Your response must contain ONLY the PR description in markdown format. Do not include:
- Any explanations or reasoning about your response
- "Here is the description:" or similar phrases
- Any text before or after the description
- Any commentary about the changes
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the PR description in markdown format, nothing else.`
}

// PRTitleUpdatePrompt creates a prompt for Claude to update existing PR titles
func PRTitleUpdatePrompt(currentTitle, branchName string) string {
	return fmt.Sprintf(`I have an existing pull request with this title:
CURRENT TITLE: "%s"

Based on our ongoing conversation, review whether this title still accurately reflects the work we've done.

INSTRUCTIONS:
- Review the current title and what we've worked on in our conversation
- ONLY update the title if the current title has become obsolete or doesn't accurately reflect the work
- If the current title still accurately captures the main purpose, return it unchanged
- If updating, use conventional commits format: "type: description"
- Choose appropriate type: feat, fix, docs, chore, refactor, test, perf, ci, build, style
- Maximum 50 characters (STRICT LIMIT)
- Be concise and specific in description
- Don't mention "Claude", "agent", or implementation details

Type Guidelines:
- feat: new features or functionality
- fix: bug fixes  
- docs: documentation changes
- refactor: code restructuring without behavior change
- chore: maintenance, dependencies, configuration
- test: adding or fixing tests
- perf: performance improvements
- ci: CI/CD pipeline changes
- build: build system or external dependencies
- style: formatting, missing semicolons (no code change)

Examples of when to update:
- Current: "Fix error handling" → New work adds user auth → Updated: "feat: add auth and fix error handling"
- Current: "Add basic feature" → New work improves performance → Updated: "feat: add feature with performance improvements"

Examples of when NOT to update:
- Current: "fix: authentication issues" → More auth bug fixes → Keep: "fix: authentication issues"
- Current: "feat: add user dashboard" → Small UI bug fixes → Keep: "feat: add user dashboard"

CRITICAL: Your response must contain ONLY the PR title text. Do not include:
- Any explanations or reasoning about your decision
- Quotes around the title
- "The title should be:" or similar phrases
- Commentary about whether you updated it or not
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the title text (updated or unchanged), nothing else.`, currentTitle)
}

// PRDescriptionUpdatePrompt creates a prompt for Claude to update existing PR descriptions
func PRDescriptionUpdatePrompt(currentDescriptionClean, branchName string) string {
	return fmt.Sprintf(`I have an existing pull request with this description:

CURRENT DESCRIPTION:
%s

Based on our ongoing conversation, review whether this description still accurately captures all the work we've done.

INSTRUCTIONS:
- Review the current description and what we've worked on in our conversation
- ONLY update the description if significant new functionality has been added that warrants description updates
- If the current description still accurately captures the work, return it unchanged (without footer)
- If updating, make it additive - enhance the existing description rather than replacing it
- Keep the same structure: ## Summary and ## Why sections
- Focus on WHAT changed at a high level and WHY the change was necessary
- Use proper markdown formatting
- Keep it professional but brief
- Do NOT mention implementation details

Examples of when to update:
- Current description only mentions "Fix auth bug" → New work adds complete user management → Update to include both
- Current description is "Add dashboard" → New work adds charts and filters → Update to "Add dashboard with charts and filtering"

Examples of when NOT to update:
- Current description covers "User authentication system" → New work just fixes small auth bugs → Keep current
- Current description mentions "Performance improvements" → New work makes minor tweaks → Keep current

IMPORTANT: 
- Do NOT include any "Generated with Claude Control" or similar footer text. I will add that separately.
- Return only the description content in markdown format, nothing else.
- If no update is needed, return the current description exactly as provided (minus any footer).

CRITICAL: Your response must contain ONLY the PR description in markdown format. Do not include:
- Any explanations or reasoning about your decision
- "Here is the updated description:" or similar phrases
- Commentary about whether you updated it or not
- Any text before or after the description
- Any analysis of the changes
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the PR description in markdown format, nothing else.`, currentDescriptionClean)
}
