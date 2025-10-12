package env

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestEnvManager_Basic(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ccagent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test .env file
	envPath := filepath.Join(tempDir, ".env")
	envContent := "TEST_VAR=test_value\nANOTHER_VAR=another_value\n"
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write test .env file: %v", err)
	}

	// Create EnvManager with custom path
	em := &EnvManager{
		envVars:  make(map[string]string),
		envPath:  envPath,
		stopChan: make(chan struct{}),
	}

	// Test Load
	if err := em.Load(); err != nil {
		t.Fatalf("Failed to load env vars: %v", err)
	}

	// Test Get for loaded var
	if got := em.Get("TEST_VAR"); got != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", got)
	}

	if got := em.Get("ANOTHER_VAR"); got != "another_value" {
		t.Errorf("Expected 'another_value', got '%s'", got)
	}

	// Test Get for non-existent var (should fall back to os.Getenv)
	os.Setenv("OS_VAR", "os_value")
	defer os.Unsetenv("OS_VAR")

	if got := em.Get("OS_VAR"); got != "os_value" {
		t.Errorf("Expected 'os_value', got '%s'", got)
	}
}

func TestEnvManager_Reload(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ccagent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create initial test .env file
	envPath := filepath.Join(tempDir, ".env")
	envContent := "TEST_VAR=initial_value\n"
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write test .env file: %v", err)
	}

	// Create EnvManager with custom path
	em := &EnvManager{
		envVars:  make(map[string]string),
		envPath:  envPath,
		stopChan: make(chan struct{}),
	}

	// Load initial values
	if err := em.Load(); err != nil {
		t.Fatalf("Failed to load env vars: %v", err)
	}

	if got := em.Get("TEST_VAR"); got != "initial_value" {
		t.Errorf("Expected 'initial_value', got '%s'", got)
	}

	// Update .env file
	updatedContent := "TEST_VAR=updated_value\nNEW_VAR=new_value\n"
	if err := os.WriteFile(envPath, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update test .env file: %v", err)
	}

	// Reload
	if err := em.Reload(); err != nil {
		t.Fatalf("Failed to reload env vars: %v", err)
	}

	// Test updated value
	if got := em.Get("TEST_VAR"); got != "updated_value" {
		t.Errorf("Expected 'updated_value', got '%s'", got)
	}

	// Test new value
	if got := em.Get("NEW_VAR"); got != "new_value" {
		t.Errorf("Expected 'new_value', got '%s'", got)
	}
}

func TestEnvManager_ThreadSafety(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "ccagent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test .env file
	envPath := filepath.Join(tempDir, ".env")
	envContent := "TEST_VAR=test_value\n"
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write test .env file: %v", err)
	}

	// Create EnvManager with custom path
	em := &EnvManager{
		envVars:  make(map[string]string),
		envPath:  envPath,
		stopChan: make(chan struct{}),
	}

	if err := em.Load(); err != nil {
		t.Fatalf("Failed to load env vars: %v", err)
	}

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	const numRoutines = 10
	const numOperations = 100

	// Start goroutines that read
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = em.Get("TEST_VAR")
			}
		}()
	}

	// Start goroutines that reload
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = em.Reload()
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()
}

func TestEnvManager_MissingFile(t *testing.T) {
	// Create EnvManager with non-existent file path
	em := &EnvManager{
		envVars:  make(map[string]string),
		envPath:  "/non/existent/path/.env",
		stopChan: make(chan struct{}),
	}

	// Load should not fail, just log a debug message
	if err := em.Load(); err != nil {
		t.Errorf("Load should not fail with missing file: %v", err)
	}

	// Reload should not fail either
	if err := em.Reload(); err != nil {
		t.Errorf("Reload should not fail with missing file: %v", err)
	}

	// Should fall back to system env vars
	os.Setenv("FALLBACK_VAR", "fallback_value")
	defer os.Unsetenv("FALLBACK_VAR")

	if got := em.Get("FALLBACK_VAR"); got != "fallback_value" {
		t.Errorf("Expected 'fallback_value', got '%s'", got)
	}
}
