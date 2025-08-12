package usecases

import "fmt"

// CommitMessageGenerationPrompt creates a prompt for Claude to generate commit messages
func CommitMessageGenerationPrompt(branchName string) string {
	return fmt.Sprintf(`I'm completing work on Git branch: "%s"

CRITICAL INSTRUCTIONS - READ CAREFULLY:
1. You MUST respond with ONLY the commit message text
2. NO explanations, NO additional text, NO formatting markup
3. NO "Here is the commit message:" or similar phrases
4. Maximum 50 characters total (STRICT LIMIT)
5. Start with action verb (Add, Fix, Update, etc.)
6. Use imperative mood

FORMAT EXAMPLE:
Fix user authentication validation

YOUR RESPONSE MUST BE THE COMMIT MESSAGE ONLY.`, branchName)
}

// PRTitleGenerationPrompt creates a prompt for Claude to generate PR titles
func PRTitleGenerationPrompt(branchName, commitInfo, diffSummary, diffContent string) string {
	return fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Here are the commits made on this branch (not including main branch commits):
%s

Files changed:
%s

Actual code changes:
%s

Generate a SHORT pull request title following conventional commits format. Follow these strict rules:
- Maximum 50 characters (STRICT LIMIT)
- Use conventional commits format: "type: description"
- Choose appropriate type: feat, fix, docs, chore, refactor, test, perf, ci, build, style
- Be concise and specific in description
- No unnecessary words or phrases
- Don't mention "Claude", "agent", or implementation details
- Base the title on the actual changes shown above

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

Respond with ONLY the short title text, nothing else.`, branchName, commitInfo, diffSummary, diffContent)
}

// PRDescriptionGenerationPrompt creates a prompt for Claude to generate PR descriptions
func PRDescriptionGenerationPrompt(branchName, commitInfo, diffSummary, diffContent string) string {
	return fmt.Sprintf(`I'm creating a pull request for Git branch: "%s"

Here are the commits made on this branch (not including main branch commits):
%s

Files changed:
%s

Actual code changes:
%s

Generate a concise pull request description with:
- ## Summary: High-level overview of what changed (2-3 bullet points max)
- ## Why: Brief explanation of the motivation/reasoning behind the change

Keep it professional but brief. Focus on WHAT changed at a high level and WHY the change was necessary, not detailed implementation specifics.

Use proper markdown formatting.

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

Respond with ONLY the PR description in markdown format, nothing else.`, branchName, commitInfo, diffSummary, diffContent)
}

// PRTitleUpdatePrompt creates a prompt for Claude to update existing PR titles
func PRTitleUpdatePrompt(currentTitle, branchName, commitInfo, diffSummary string) string {
	return fmt.Sprintf(`I have an existing pull request with this title:
CURRENT TITLE: "%s"

The branch "%s" now has these commits and changes:

%s

Files changed:
%s

INSTRUCTIONS:
- Review the current title and the latest changes made to this branch
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
- Current: "Fix error handling" → New commits add user auth → Updated: "feat: add auth and fix error handling"
- Current: "Add basic feature" → New commits improve performance → Updated: "feat: add feature with performance improvements"

Examples of when NOT to update:
- Current: "fix: authentication issues" → New commits fix more auth bugs → Keep: "fix: authentication issues"
- Current: "feat: add user dashboard" → New commits fix small UI bugs → Keep: "feat: add user dashboard"

CRITICAL: Your response must contain ONLY the PR title text. Do not include:
- Any explanations or reasoning about your decision
- Quotes around the title
- "The title should be:" or similar phrases
- Commentary about whether you updated it or not
- Any other text whatsoever
- Do NOT execute any git or gh commands
- Do NOT create, update, or modify any pull requests
- Do NOT perform any actions - this is a text-only request

Respond with ONLY the title text (updated or unchanged), nothing else.`, currentTitle, branchName, commitInfo, diffSummary)
}

// PRDescriptionUpdatePrompt creates a prompt for Claude to update existing PR descriptions
func PRDescriptionUpdatePrompt(currentDescriptionClean, branchName, commitInfo, diffSummary string) string {
	return fmt.Sprintf(`I have an existing pull request with this description:

CURRENT DESCRIPTION:
%s

The branch "%s" now has these commits and changes:

All commits on this branch:
%s

Files changed:
%s

INSTRUCTIONS:
- Review the current description and the latest changes made to this branch
- ONLY update the description if significant new functionality has been added that warrants description updates
- If the current description still accurately captures the work, return it unchanged (without footer)
- If updating, make it additive - enhance the existing description rather than replacing it
- Keep the same structure: ## Summary and ## Why sections
- Focus on WHAT changed at a high level and WHY the change was necessary
- Use proper markdown formatting
- Keep it professional but brief
- Do NOT mention implementation details

Examples of when to update:
- Current description only mentions "Fix auth bug" → New commits add complete user management → Update to include both
- Current description is "Add dashboard" → New commits add charts and filters → Update to "Add dashboard with charts and filtering"

Examples of when NOT to update:
- Current description covers "User authentication system" → New commits just fix small auth bugs → Keep current
- Current description mentions "Performance improvements" → New commits make minor tweaks → Keep current

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

Respond with ONLY the PR description in markdown format, nothing else.`, currentDescriptionClean, branchName, commitInfo, diffSummary)
}
