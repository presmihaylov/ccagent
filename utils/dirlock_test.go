package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeDirPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/home/user/project", "home--user--project"},
		{"C:\\Users\\user\\project", "C----Users--user--project"},
		{"/path/with:special*chars?", "path--with--special-star-chars-q"},
		{"/path/with<>|\"quotes", "path--with-lt--gt--pipe--quote-quotes"},
		{"", "default"},
		{"...", "default"},
		{"---", "default"},
		{"/normal/path", "normal--path"},
	}

	for _, test := range tests {
		result := sanitizeDirPath(test.input)
		if result != test.expected {
			t.Errorf("sanitizeDirPath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestNewDirLock(t *testing.T) {
	lock, err := NewDirLock()
	if err != nil {
		t.Fatalf("NewDirLock() failed: %v", err)
	}

	// Verify lock path contains expected components
	lockPath := lock.GetLockPath()
	if !strings.Contains(lockPath, "ccagent") {
		t.Errorf("Lock path should contain 'ccagent': %s", lockPath)
	}

	if !strings.HasSuffix(lockPath, ".lock") {
		t.Errorf("Lock path should end with '.lock': %s", lockPath)
	}

	// Verify the ccagent directory was created
	ccagentDir := filepath.Dir(lockPath)
	if _, err := os.Stat(ccagentDir); os.IsNotExist(err) {
		t.Errorf("ccagent directory should be created: %s", ccagentDir)
	}
}

func TestDirLockTryLockAndUnlock(t *testing.T) {
	lock1, err := NewDirLock()
	if err != nil {
		t.Fatalf("NewDirLock() failed: %v", err)
	}

	// First lock should succeed
	err = lock1.TryLock()
	if err != nil {
		t.Fatalf("First TryLock() should succeed: %v", err)
	}

	// Second lock from same directory should fail
	lock2, err := NewDirLock()
	if err != nil {
		t.Fatalf("Second NewDirLock() failed: %v", err)
	}

	err = lock2.TryLock()
	if err == nil {
		t.Errorf("Second TryLock() should fail when directory is already locked")
		// Clean up the unexpected lock
		lock2.Unlock()
	}

	// Unlock the first lock
	err = lock1.Unlock()
	if err != nil {
		t.Errorf("Unlock() failed: %v", err)
	}

	// Verify lock file was removed
	if _, err := os.Stat(lock1.GetLockPath()); !os.IsNotExist(err) {
		t.Errorf("Lock file should be removed after unlock: %s", lock1.GetLockPath())
	}

	// Third lock should now succeed after first was unlocked
	lock3, err := NewDirLock()
	if err != nil {
		t.Fatalf("Third NewDirLock() failed: %v", err)
	}

	err = lock3.TryLock()
	if err != nil {
		t.Errorf("Third TryLock() should succeed after first was unlocked: %v", err)
	}

	// Clean up
	lock3.Unlock()
}

func TestDirLockUnlockIdempotent(t *testing.T) {
	lock, err := NewDirLock()
	if err != nil {
		t.Fatalf("NewDirLock() failed: %v", err)
	}

	// Lock first
	err = lock.TryLock()
	if err != nil {
		t.Fatalf("TryLock() failed: %v", err)
	}

	// Unlock should succeed
	err = lock.Unlock()
	if err != nil {
		t.Errorf("First Unlock() failed: %v", err)
	}

	// Second unlock should not fail
	err = lock.Unlock()
	if err != nil {
		t.Errorf("Second Unlock() should not fail: %v", err)
	}
}
