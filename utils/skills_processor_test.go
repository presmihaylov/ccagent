package utils

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSkillNameFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "standard format with 6-char ID",
			filename: "code-reviewer-a1b2c3.zip",
			want:     "code-reviewer",
		},
		{
			name:     "multi-part name with 6-char ID",
			filename: "pdf-processor-x4y5z6.zip",
			want:     "pdf-processor",
		},
		{
			name:     "single word name",
			filename: "formatter-abc123.zip",
			want:     "formatter",
		},
		{
			name:     "name with numbers and 6-char ID",
			filename: "skill-v2-beta-xyz789.zip",
			want:     "skill-v2-beta",
		},
		{
			name:     "no attachment ID suffix",
			filename: "my-skill.zip",
			want:     "my-skill",
		},
		{
			name:     "short name",
			filename: "foo.zip",
			want:     "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractSkillNameFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("ExtractSkillNameFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractZipToDirectory(t *testing.T) {
	tests := []struct {
		name           string
		zipStructure   map[string]string // path -> content
		expectedFiles  map[string]string // path -> content after extraction
		shouldFail     bool
		expectedError  string
		singleRootDir  string // if not empty, structure has single root dir
	}{
		{
			name: "ZIP without root directory",
			zipStructure: map[string]string{
				"SKILL.md":            "# Skill Instructions",
				"scripts/deploy.sh":   "#!/bin/bash\necho deploy",
				"references/docs.md":  "# Documentation",
			},
			expectedFiles: map[string]string{
				"SKILL.md":            "# Skill Instructions",
				"scripts/deploy.sh":   "#!/bin/bash\necho deploy",
				"references/docs.md":  "# Documentation",
			},
		},
		{
			name: "ZIP with single root directory",
			zipStructure: map[string]string{
				"code-reviewer/SKILL.md":           "# Code Review Skill",
				"code-reviewer/scripts/review.sh":  "#!/bin/bash\nreview code",
				"code-reviewer/assets/template.md": "Template content",
			},
			singleRootDir: "code-reviewer",
			expectedFiles: map[string]string{
				"SKILL.md":           "# Code Review Skill",
				"scripts/review.sh":  "#!/bin/bash\nreview code",
				"assets/template.md": "Template content",
			},
		},
		{
			name: "ZIP with multiple root directories",
			zipStructure: map[string]string{
				"skill1/SKILL.md": "Skill 1",
				"skill2/SKILL.md": "Skill 2",
			},
			expectedFiles: map[string]string{
				"skill1/SKILL.md": "Skill 1",
				"skill2/SKILL.md": "Skill 2",
			},
		},
		{
			name: "ZIP with zip slip attempt",
			zipStructure: map[string]string{
				"../../../etc/passwd": "malicious content",
			},
			shouldFail:    true,
			expectedError: "zip slip attack detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for testing
			tmpDir := t.TempDir()
			zipPath := filepath.Join(tmpDir, "test.zip")
			extractDir := filepath.Join(tmpDir, "extract")

			// Create the ZIP file
			err := createTestZip(zipPath, tt.zipStructure)
			if err != nil {
				t.Fatalf("Failed to create test ZIP: %v", err)
			}

			// Extract the ZIP
			err = ExtractZipToDirectory(zipPath, extractDir)

			// Check for expected errors
			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected error containing %q, but got no error", tt.expectedError)
				} else if tt.expectedError != "" && !contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("ExtractZipToDirectory failed: %v", err)
			}

			// Verify extracted files
			for expectedPath, expectedContent := range tt.expectedFiles {
				fullPath := filepath.Join(extractDir, expectedPath)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("Failed to read extracted file %s: %v", expectedPath, err)
					continue
				}

				if string(content) != expectedContent {
					t.Errorf("File %s content = %q, want %q", expectedPath, string(content), expectedContent)
				}
			}
		})
	}
}

func TestDetectSingleRootDirectory(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantRoot string
	}{
		{
			name:     "single root directory",
			files:    []string{"skill/SKILL.md", "skill/scripts/test.sh", "skill/assets/file.txt"},
			wantRoot: "skill",
		},
		{
			name:     "multiple root directories",
			files:    []string{"skill1/file.txt", "skill2/file.txt"},
			wantRoot: "",
		},
		{
			name:     "no root directory",
			files:    []string{"SKILL.md", "scripts/test.sh"},
			wantRoot: "",
		},
		{
			name:     "empty ZIP",
			files:    []string{},
			wantRoot: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock ZIP reader
			buf := new(bytes.Buffer)
			w := zip.NewWriter(buf)

			for _, file := range tt.files {
				f, err := w.Create(file)
				if err != nil {
					t.Fatalf("Failed to create ZIP entry: %v", err)
				}
				_, err = f.Write([]byte("content"))
				if err != nil {
					t.Fatalf("Failed to write ZIP entry: %v", err)
				}
			}

			err := w.Close()
			if err != nil {
				t.Fatalf("Failed to close ZIP writer: %v", err)
			}

			// Create ZIP reader
			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				t.Fatalf("Failed to create ZIP reader: %v", err)
			}

			// Test the function
			got := detectSingleRootDirectory(r)
			if got != tt.wantRoot {
				t.Errorf("detectSingleRootDirectory() = %q, want %q", got, tt.wantRoot)
			}
		})
	}
}

// Helper function to create a test ZIP file
func createTestZip(zipPath string, files map[string]string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	for path, content := range files {
		f, err := w.Create(path)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
