package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRepositoryStatus(t *testing.T) {
	tests := []struct {
		name               string
		setupRepo          func(t *testing.T, repoDir string)
		expectError        bool
		expectWarnings     bool
		expectCanProceed   bool
		minValidationSteps int
	}{
		{
			name: "clean repository",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
			},
			expectError:        false,
			expectWarnings:     true, // Remote connectivity check typically generates warnings
			expectCanProceed:   true,
			minValidationSteps: ValidationStepCount,
		},
		{
			name: "repository with uncommitted changes",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
				// Add uncommitted changes
				writeFile(t, filepath.Join(repoDir, "test.txt"), "modified content")
			},
			expectError:        true,
			expectWarnings:     true, // Will also have connectivity warnings
			expectCanProceed:   false,
			minValidationSteps: ValidationStepCount,
		},
		{
			name: "repository with ignored untracked files",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
				// Add .gitignore to ignore some files
				writeFile(t, filepath.Join(repoDir, ".gitignore"), "*.tmp\n")
				runGitCommand(t, repoDir, "add", ".gitignore")
				runGitCommand(t, repoDir, "commit", "-m", "add gitignore")
				// Add untracked file that's ignored (won't show in git status --porcelain)
				writeFile(t, filepath.Join(repoDir, "ignored.tmp"), "ignored content")
			},
			expectError:        false, // Ignored files don't cause errors
			expectWarnings:     true,  // Still have remote warnings
			expectCanProceed:   true,
			minValidationSteps: ValidationStepCount,
		},
		{
			name: "repository with visible untracked files",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
				// Add untracked file that's not ignored (this will cause uncommitted changes)
				writeFile(t, filepath.Join(repoDir, "untracked.txt"), "untracked content")
			},
			expectError:        true, // Visible untracked files cause uncommitted changes error
			expectWarnings:     true, // Also have untracked file warnings + remote warnings
			expectCanProceed:   false,
			minValidationSteps: ValidationStepCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test repository
			repoDir := createTempDir(t)
			defer func() {
				if err := os.RemoveAll(repoDir); err != nil {
					t.Logf("Warning: failed to remove temp dir: %v", err)
				}
			}()

			// Change to the repo directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original dir: %v", err)
				}
			}()

			if err := os.Chdir(repoDir); err != nil {
				t.Fatalf("Failed to change to repo directory: %v", err)
			}

			// Setup the repository
			tt.setupRepo(t, repoDir)

			// Run validation
			manager := NewManager()
			summary, err := manager.ValidateRepositoryStatus()

			// Check basic expectations
			if tt.expectError && err != nil {
				return // Expected error case
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if summary == nil {
				t.Fatal("Expected validation summary, got nil")
			}

			// Check validation results
			if len(summary.Results) < tt.minValidationSteps {
				t.Errorf("Expected at least %d validation steps, got %d", tt.minValidationSteps, len(summary.Results))
			}

			if summary.HasErrors != tt.expectError {
				t.Errorf("Expected HasErrors=%v, got %v", tt.expectError, summary.HasErrors)
				// Debug: print all errors
				for i, result := range summary.Results {
					if len(result.Errors) > 0 {
						t.Logf("Step %d (%s) errors: %v", i, result.Step.Name, result.Errors)
					}
				}
			}

			if summary.HasWarnings != tt.expectWarnings {
				t.Errorf("Expected HasWarnings=%v, got %v", tt.expectWarnings, summary.HasWarnings)
				// Debug: print all warnings
				for i, result := range summary.Results {
					if len(result.Warnings) > 0 {
						t.Logf("Step %d (%s) warnings: %v", i, result.Step.Name, result.Warnings)
					}
				}
			}

			if summary.CanProceed != tt.expectCanProceed {
				t.Errorf("Expected CanProceed=%v, got %v", tt.expectCanProceed, summary.CanProceed)
			}
		})
	}
}

func TestGetSubmodules(t *testing.T) {
	tests := []struct {
		name            string
		submoduleOutput string
		expectedCount   int
		expectedNames   []string
	}{
		{
			name:            "no submodules",
			submoduleOutput: "",
			expectedCount:   0,
			expectedNames:   []string{},
		},
		{
			name:            "single submodule with status char",
			submoduleOutput: " 1234567890abcdef1234567890abcdef12345678 path/to/submodule (v1.0.0)",
			expectedCount:   1,
			expectedNames:   []string{"submodule"},
		},
		{
			name:            "single submodule without status char",
			submoduleOutput: "1234567890abcdef1234567890abcdef12345678 path/to/submodule",
			expectedCount:   1,
			expectedNames:   []string{"submodule"},
		},
		{
			name: "multiple submodules mixed format",
			submoduleOutput: ` 1234567890abcdef1234567890abcdef12345678 libs/first-lib (v1.0.0)
+abcdef1234567890abcdef1234567890abcdef12 libs/second-lib (v2.0.0-1-gabcdef1)
 fedcba0987654321fedcba0987654321fedcba09 third-lib`,
			expectedCount: 3,
			expectedNames: []string{"first-lib", "second-lib", "third-lib"},
		},
		{
			name:            "malformed line - too short",
			submoduleOutput: "short",
			expectedCount:   0,
			expectedNames:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory with fake git setup
			repoDir := createTempDir(t)
			defer func() {
				if err := os.RemoveAll(repoDir); err != nil {
					t.Logf("Warning: failed to remove temp dir: %v", err)
				}
			}()

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original dir: %v", err)
				}
			}()

			if err := os.Chdir(repoDir); err != nil {
				t.Fatalf("Failed to change to repo directory: %v", err)
			}

			// Initialize git repo
			runGitCommand(t, repoDir, "init")

			// Create .gitmodules file if we expect submodules
			if tt.expectedCount > 0 {
				writeFile(t, filepath.Join(repoDir, ".gitmodules"), "[submodule \"test\"]\n")
				runGitCommand(t, repoDir, "add", ".gitmodules")
			}

			// We can't easily mock git submodule status, so we'll test the parsing logic directly
			// This test would need to be enhanced with a proper git command mocker
			t.Skip("Skipping integration test - would need git command mocking")
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1234567890abcdef", true},
		{"1234567890ABCDEF", true},
		{"1234567890abcdef1234567890abcdef12345678", true}, // 40 chars
		{"1234567890abcdefg", false},                       // contains 'g'
		{"", true},                                         // empty string
		{"xyz", false},
		{"123", true},
		{"abc", true},
		{"ABC", true},
		{"123456789abcdefghijklmnopqrstuv", false}, // contains non-hex chars
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isHexString(tt.input)
			if result != tt.expected {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	tests := []struct {
		name          string
		setupRepo     func(t *testing.T, repoDir string)
		expectChanges bool
		expectError   bool
	}{
		{
			name: "clean repository",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
			},
			expectChanges: false,
			expectError:   false,
		},
		{
			name: "modified files",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
				// Modify file
				writeFile(t, filepath.Join(repoDir, "test.txt"), "modified content")
			},
			expectChanges: true,
			expectError:   false,
		},
		{
			name: "staged changes",
			setupRepo: func(t *testing.T, repoDir string) {
				runGitCommand(t, repoDir, "init")
				runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
				runGitCommand(t, repoDir, "config", "user.name", "Test User")
				writeFile(t, filepath.Join(repoDir, "test.txt"), "test content")
				runGitCommand(t, repoDir, "add", "test.txt")
				runGitCommand(t, repoDir, "commit", "-m", "initial commit")
				// Add new file and stage it
				writeFile(t, filepath.Join(repoDir, "new.txt"), "new content")
				runGitCommand(t, repoDir, "add", "new.txt")
			},
			expectChanges: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := createTempDir(t)
			defer func() {
				if err := os.RemoveAll(repoDir); err != nil {
					t.Logf("Warning: failed to remove temp dir: %v", err)
				}
			}()

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original dir: %v", err)
				}
			}()

			if err := os.Chdir(repoDir); err != nil {
				t.Fatalf("Failed to change to repo directory: %v", err)
			}

			tt.setupRepo(t, repoDir)

			manager := NewManager()
			hasChanges, err := manager.HasUncommittedChanges()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if hasChanges != tt.expectChanges {
				t.Errorf("Expected HasUncommittedChanges=%v, got %v", tt.expectChanges, hasChanges)
			}
		})
	}
}

func TestGetUntrackedFiles(t *testing.T) {
	repoDir := createTempDir(t)
	defer func() {
		if err := os.RemoveAll(repoDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: failed to change back to original dir: %v", err)
		}
	}()

	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}

	runGitCommand(t, repoDir, "init")
	runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
	runGitCommand(t, repoDir, "config", "user.name", "Test User")

	// Initial state - no untracked files
	manager := NewManager()
	untracked, err := manager.getUntrackedFiles()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(untracked) != 0 {
		t.Errorf("Expected 0 untracked files, got %d", len(untracked))
	}

	// Add an untracked file
	writeFile(t, filepath.Join(repoDir, "untracked.txt"), "untracked content")
	untracked, err = manager.getUntrackedFiles()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(untracked) != 1 {
		t.Errorf("Expected 1 untracked file, got %d", len(untracked))
	}
	if len(untracked) > 0 && untracked[0] != "untracked.txt" {
		t.Errorf("Expected untracked file 'untracked.txt', got '%s'", untracked[0])
	}

	// Track the file
	runGitCommand(t, repoDir, "add", "untracked.txt")
	untracked, err = manager.getUntrackedFiles()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(untracked) != 0 {
		t.Errorf("Expected 0 untracked files after adding, got %d", len(untracked))
	}
}

func TestParseSubmoduleStatusLine(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedName   string
		expectedPath   string
		expectedCommit string
		expectError    bool
	}{
		{
			name:           "normal status with leading space",
			line:           " 1234567890abcdef1234567890abcdef12345678 path/to/submodule (v1.0.0)",
			expectedName:   "submodule",
			expectedPath:   "path/to/submodule",
			expectedCommit: "1234567890abcdef1234567890abcdef12345678",
			expectError:    false,
		},
		{
			name:           "status without leading space",
			line:           "1234567890abcdef1234567890abcdef12345678 path/to/submodule",
			expectedName:   "submodule",
			expectedPath:   "path/to/submodule",
			expectedCommit: "1234567890abcdef1234567890abcdef12345678",
			expectError:    false,
		},
		{
			name:           "status with different character",
			line:           "+abcdef1234567890abcdef1234567890abcdef12 libs/second-lib (v2.0.0-1-gabcdef1)",
			expectedName:   "second-lib",
			expectedPath:   "libs/second-lib",
			expectedCommit: "abcdef1234567890abcdef1234567890abcdef12",
			expectError:    false,
		},
		{
			name:           "simple path",
			line:           " fedcba0987654321fedcba0987654321fedcba09 third-lib",
			expectedName:   "third-lib",
			expectedPath:   "third-lib",
			expectedCommit: "fedcba0987654321fedcba0987654321fedcba09",
			expectError:    false,
		},
		{
			name:        "line too short",
			line:        "short",
			expectError: true,
		},
		{
			name:        "invalid commit hash",
			line:        " xyz4567890abcdef1234567890abcdef12345678 path/to/submodule",
			expectError: true,
		},
		{
			name:        "commit hash too short",
			line:        " 1234567890abcdef123 path/to/submodule",
			expectError: true,
		},
		{
			name:        "missing path",
			line:        " 1234567890abcdef1234567890abcdef12345678",
			expectError: true,
		},
	}

	manager := NewManager()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			submodule, err := manager.parseSubmoduleStatusLine(tt.line)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for line %q but got none", tt.line)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for line %q: %v", tt.line, err)
				return
			}

			if submodule.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, submodule.Name)
			}

			if submodule.Path != tt.expectedPath {
				t.Errorf("Expected path %q, got %q", tt.expectedPath, submodule.Path)
			}

			if submodule.Commit != tt.expectedCommit {
				t.Errorf("Expected commit %q, got %q", tt.expectedCommit, submodule.Commit)
			}
		})
	}
}

func TestValidateSubmodulePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid relative path",
			path:        "libs/mylib",
			expectError: false,
		},
		{
			name:        "valid simple path",
			path:        "submodule",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
		},
		{
			name:        "absolute path",
			path:        "/etc/passwd",
			expectError: true,
		},
		{
			name:        "path traversal with dots",
			path:        "../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "path traversal in middle",
			path:        "libs/../../../etc/passwd",
			expectError: true,
		},
		{
			name:        "Windows drive letter",
			path:        "C:\\Windows\\System32",
			expectError: true,
		},
		{
			name:        "Home directory",
			path:        "~/malicious",
			expectError: true,
		},
		{
			name:        "Valid nested path",
			path:        "third-party/libs/openssl",
			expectError: false,
		},
		{
			name:        "Path that cleans to parent directory",
			path:        "valid/../..",
			expectError: true,
		},
		{
			name:        "Path that cleans to current directory",
			path:        "submodule/.",
			expectError: false,
		},
		{
			name:        "Path with consecutive separators",
			path:        "lib//submodule",
			expectError: false,
		},
		{
			name:        "Path with null byte",
			path:        "lib\x00/malicious",
			expectError: true,
		},
		{
			name:        "Path with newline",
			path:        "lib\n/malicious",
			expectError: true,
		},
		{
			name:        "Complex path traversal that normalizes",
			path:        "a/b/../../../c",
			expectError: true,
		},
	}

	manager := NewManager()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateSubmodulePath(tt.path)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for path %q but got none", tt.path)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T, dir string)
		expectError bool
	}{
		{
			name: "valid git repository",
			setupDir: func(t *testing.T, dir string) {
				runGitCommand(t, dir, "init")
			},
			expectError: false,
		},
		{
			name: "not a git repository",
			setupDir: func(t *testing.T, dir string) {
				// Don't initialize git
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := createTempDir(t)
			defer func() {
				if err := os.RemoveAll(testDir); err != nil {
					t.Logf("Warning: failed to remove temp dir: %v", err)
				}
			}()

			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original dir: %v", err)
				}
			}()

			if err := os.Chdir(testDir); err != nil {
				t.Fatalf("Failed to change to test directory: %v", err)
			}

			tt.setupDir(t, testDir)

			manager := NewManager()
			err = manager.IsGitRepository()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Helper functions for tests

func createTempDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "git-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return dir
}

func writeFile(t *testing.T, path, content string) {
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write file %s: %v", path, err)
	}
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Git command failed: git %s\nError: %v\nOutput: %s",
			strings.Join(args, " "), err, string(output))
	}
}
