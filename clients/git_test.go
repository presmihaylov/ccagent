package clients

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	configCmds := [][]string{
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test User"},
	}
	for _, cmdArgs := range configCmds {
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to configure git: %v", err)
		}
	}

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to commit: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestRemoteBranchExists_NonExistentRemote(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	// Test checking for a remote branch when no remote is configured
	// This should return an error since git ls-remote needs a remote
	_, err := client.RemoteBranchExists("some-branch")
	if err == nil {
		t.Error("Expected error when checking remote branch without remote configured, got nil")
	}
}

func TestRemoteBranchExists_WithRemote(t *testing.T) {
	// This test requires a real git repository with remote
	// Skip if not in a git repo or if origin remote doesn't exist
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: no origin remote configured")
	}

	client := NewGitClient()
	// Use current directory (should be in ccagent repo)
	client.SetRepoPathProvider(func() string { return "" })

	// Test with a branch name that likely doesn't exist
	exists, err := client.RemoteBranchExists("nonexistent-branch-12345-test")
	if err != nil {
		t.Fatalf("Unexpected error checking remote branch: %v", err)
	}

	if exists {
		t.Error("Expected nonexistent branch to return false, got true")
	}
}

func TestGetLocalBranches_EmptyRepo(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	branches, err := client.GetLocalBranches()
	if err != nil {
		t.Fatalf("Failed to get local branches: %v", err)
	}

	// Should have at least the main/master branch
	if len(branches) == 0 {
		t.Error("Expected at least one branch, got 0")
	}

	// Check that main or master is in the list
	hasDefaultBranch := false
	for _, branch := range branches {
		if branch == "main" || branch == "master" {
			hasDefaultBranch = true
			break
		}
	}

	if !hasDefaultBranch {
		t.Errorf("Expected main or master branch in list, got: %v", branches)
	}
}

func TestGetLocalBranches_WithMultipleBranches(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create additional branches
	branchNames := []string{"feature/test", "bugfix/issue-123"}
	for _, branchName := range branchNames {
		cmd := exec.Command("git", "branch", branchName)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create branch %s: %v", branchName, err)
		}
	}

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	branches, err := client.GetLocalBranches()
	if err != nil {
		t.Fatalf("Failed to get local branches: %v", err)
	}

	// Should have at least 3 branches (main/master + 2 created)
	if len(branches) < 3 {
		t.Errorf("Expected at least 3 branches, got %d", len(branches))
	}

	// Check that our created branches are in the list
	for _, expectedBranch := range branchNames {
		found := false
		for _, branch := range branches {
			if branch == expectedBranch {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected branch %s in list, got: %v", expectedBranch, branches)
		}
	}
}

func TestCheckoutBranch_ExistingBranch(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create a test branch
	branchName := "test-branch"
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	// Checkout the branch
	err := client.CheckoutBranch(branchName)
	if err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Verify we're on the correct branch
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	currentBranch := strings.TrimSpace(string(output))
	if currentBranch != branchName {
		t.Errorf("Expected current branch to be %s, got %s", branchName, currentBranch)
	}
}

func TestCheckoutBranch_NonExistentBranch(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	// Try to checkout non-existent branch
	err := client.CheckoutBranch("nonexistent-branch")
	if err == nil {
		t.Error("Expected error when checking out non-existent branch, got nil")
	}

	if !strings.Contains(err.Error(), "git checkout failed") {
		t.Errorf("Expected 'git checkout failed' in error, got: %v", err)
	}
}

func TestDeleteLocalBranch_ExistingBranch(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	// Create and then delete a branch
	branchName := "branch-to-delete"
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	// Delete the branch
	err := client.DeleteLocalBranch(branchName)
	if err != nil {
		t.Fatalf("Failed to delete branch: %v", err)
	}

	// Verify branch is gone
	branches, err := client.GetLocalBranches()
	if err != nil {
		t.Fatalf("Failed to get local branches: %v", err)
	}

	for _, branch := range branches {
		if branch == branchName {
			t.Errorf("Branch %s should have been deleted but still exists", branchName)
		}
	}
}

func TestDeleteLocalBranch_NonExistentBranch(t *testing.T) {
	repoPath, cleanup := setupTestGitRepo(t)
	defer cleanup()

	client := NewGitClient()
	client.SetRepoPathProvider(func() string { return repoPath })

	// Try to delete non-existent branch
	err := client.DeleteLocalBranch("nonexistent-branch")
	if err == nil {
		t.Error("Expected error when deleting non-existent branch, got nil")
	}
}
