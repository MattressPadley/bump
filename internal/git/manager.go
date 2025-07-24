package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (g *Manager) IsGitRepository() error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

func (g *Manager) CommitVersionBump(version string) error {
	// Add all changes
	if err := g.runGitCommand("add", "."); err != nil {
		return fmt.Errorf("failed to stage changes: %v", err)
	}

	// Create commit
	message := fmt.Sprintf("chore(release): bump version to %s", version)
	if err := g.runGitCommand("commit", "-m", message); err != nil {
		return fmt.Errorf("failed to create commit: %v", err)
	}

	return nil
}

func (g *Manager) CreateTag(version string) error {
	tagName := fmt.Sprintf("v%s", version)
	message := fmt.Sprintf("Release version %s", version)

	if err := g.runGitCommand("tag", "-a", tagName, "-m", message); err != nil {
		return fmt.Errorf("failed to create tag: %v", err)
	}

	return nil
}

func (g *Manager) PushChanges() error {
	// Push commits first
	if err := g.runGitCommand("push", "origin", "HEAD"); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}
	return nil
}

func (g *Manager) PushTag(version string) error {
	tagName := fmt.Sprintf("v%s", version)
	// Push tag separately to ensure workflow triggers
	if err := g.runGitCommand("push", "origin", tagName); err != nil {
		return fmt.Errorf("failed to push tag: %v", err)
	}
	return nil
}



func (g *Manager) GetCommitsSince(fromVersion string) ([]Commit, error) {
	var args []string
	if fromVersion != "" {
		tagName := fmt.Sprintf("v%s", fromVersion)
		// First check if the tag exists
		checkCmd := exec.Command("git", "rev-parse", "--verify", tagName)
		if err := checkCmd.Run(); err != nil {
			// Tag doesn't exist, get all commits instead
			args = []string{"log", "--oneline", "--no-merges", "-10"} // Limit to last 10 commits
		} else {
			args = []string{"log", "--oneline", "--no-merges", fmt.Sprintf("%s..HEAD", tagName)}
		}
	} else {
		args = []string{"log", "--oneline", "--no-merges", "-10"} // Limit to last 10 commits
	}

	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If git log fails, return empty commits instead of error
		return []Commit{}, nil
	}

	var commits []Commit
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []Commit{}, nil
	}

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
		})
	}

	return commits, nil
}

func (g *Manager) GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %v", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Manager) HasUncommittedChanges() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check git status: %v", err)
	}

	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

func (g *Manager) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %v\nError: %s", strings.Join(args, " "), err, stderr.String())
	}

	return nil
}

type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// ValidationStep represents a step in the git validation process
type ValidationStep struct {
	Name        string
	Description string
	Index       int
	Total       int
}

// ValidationResult represents the result of a validation step
type ValidationResult struct {
	Step     ValidationStep
	Success  bool
	Warnings []string
	Errors   []string
}

// ValidationSummary contains the overall validation results
type ValidationSummary struct {
	Results      []ValidationResult
	HasErrors    bool
	HasWarnings  bool
	CanProceed   bool
}

// ProgressCallback is called during validation to report progress
type ProgressCallback func(ValidationResult)

// ValidateRepositoryStatus performs comprehensive git repository validation
func (g *Manager) ValidateRepositoryStatus(progressCallback ProgressCallback) (*ValidationSummary, error) {
	steps := []ValidationStep{
		{Name: "repository", Description: "Checking repository status...", Index: 1, Total: 6},
		{Name: "working_dir", Description: "Validating working directory...", Index: 2, Total: 6},
		{Name: "branch", Description: "Checking branch status...", Index: 3, Total: 6},
		{Name: "submodules_scan", Description: "Scanning for submodules...", Index: 4, Total: 6},
		{Name: "submodules_status", Description: "Validating submodule states...", Index: 5, Total: 6},
		{Name: "final", Description: "Final validation checks...", Index: 6, Total: 6},
	}

	var results []ValidationResult
	hasErrors := false
	hasWarnings := false

	// Step 1: Check repository status
	result := g.validateRepositoryStatus(steps[0])
	results = append(results, result)
	if progressCallback != nil {
		progressCallback(result)
	}
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	// Step 2: Validate working directory
	result = g.validateWorkingDirectory(steps[1])
	results = append(results, result)
	if progressCallback != nil {
		progressCallback(result)
	}
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	// Step 3: Check branch status
	result = g.validateBranchStatus(steps[2])
	results = append(results, result)
	if progressCallback != nil {
		progressCallback(result)
	}
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	// Step 4: Scan for submodules
	submodules, result := g.scanSubmodules(steps[3])
	results = append(results, result)
	if progressCallback != nil {
		progressCallback(result)
	}
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	// Step 5: Validate submodules (if any exist)
	if len(submodules) > 0 {
		result = g.validateSubmodules(steps[4], submodules)
		results = append(results, result)
		if progressCallback != nil {
			progressCallback(result)
		}
		if !result.Success {
			hasErrors = true
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
		}
	} else {
		// Skip submodule validation if no submodules
		result = ValidationResult{
			Step:     steps[4],
			Success:  true,
			Warnings: nil,
			Errors:   nil,
		}
		results = append(results, result)
		if progressCallback != nil {
			progressCallback(result)
		}
	}

	// Step 6: Final validation
	result = g.performFinalValidation(steps[5])
	results = append(results, result)
	if progressCallback != nil {
		progressCallback(result)
	}
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	return &ValidationSummary{
		Results:     results,
		HasErrors:   hasErrors,
		HasWarnings: hasWarnings,
		CanProceed:  !hasErrors,
	}, nil
}

// validateRepositoryStatus checks basic git repository status
func (g *Manager) validateRepositoryStatus(step ValidationStep) ValidationResult {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check if we're in a git repository
	if err := g.IsGitRepository(); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, "Not in a git repository")
		return result
	}

	return result
}

// validateWorkingDirectory checks the working directory status
func (g *Manager) validateWorkingDirectory(step ValidationStep) ValidationResult {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check for uncommitted changes
	hasChanges, err := g.HasUncommittedChanges()
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to check working directory status: %v", err))
		return result
	}

	if hasChanges {
		result.Success = false
		result.Errors = append(result.Errors, "Working directory has uncommitted changes")
	}

	// Check for untracked files
	untracked, err := g.getUntrackedFiles()
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Could not check for untracked files: %v", err))
	} else if len(untracked) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Found %d untracked files", len(untracked)))
	}

	return result
}

// validateBranchStatus checks the current branch status
func (g *Manager) validateBranchStatus(step ValidationStep) ValidationResult {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Get current branch
	branch, err := g.GetCurrentBranch()
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get current branch: %v", err))
		return result
	}

	if branch == "" {
		result.Warnings = append(result.Warnings, "In detached HEAD state")
	}

	// Check if branch is up to date with remote
	if err := g.checkRemoteStatus(branch); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Branch status: %v", err))
	}

	return result
}

// scanSubmodules scans for git submodules in the repository
func (g *Manager) scanSubmodules(step ValidationStep) ([]Submodule, ValidationResult) {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	submodules, err := g.getSubmodules()
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to scan submodules: %v", err))
		return nil, result
	}

	// Don't add a warning just for finding submodules - this is informational only
	// Warnings will be added later if submodules don't point to tags

	return submodules, result
}

// validateSubmodules validates the status of git submodules
func (g *Manager) validateSubmodules(step ValidationStep, submodules []Submodule) ValidationResult {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	tagsFound := 0
	for _, submodule := range submodules {
		// Check if submodule points to a tag
		isTag, _, err := g.isSubmodulePointingToTag(submodule.Path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to check submodule %s: %v", submodule.Name, err))
			result.Success = false
			continue
		}

		if !isTag {
			// Only warn when submodule is NOT pointing to a tag
			result.Warnings = append(result.Warnings, fmt.Sprintf("Submodule '%s' is not pointing to a release tag", submodule.Name))
		} else {
			// Success case - submodule points to a tag (no warning needed)
			tagsFound++
		}

		// Check if submodule has uncommitted changes
		if hasChanges, err := g.submoduleHasChanges(submodule.Path); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not check submodule %s status: %v", submodule.Name, err))
		} else if hasChanges {
			result.Errors = append(result.Errors, fmt.Sprintf("Submodule '%s' has uncommitted changes", submodule.Name))
			result.Success = false
		}
	}

	// Don't add summary warnings - let individual submodule warnings speak for themselves
	// The validation view will show the step as successful if no warnings/errors were added

	return result
}

// performFinalValidation performs final validation checks
func (g *Manager) performFinalValidation(step ValidationStep) ValidationResult {
	result := ValidationResult{
		Step:     step,
		Success:  true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check git connectivity
	if err := g.checkGitConnectivity(); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Git connectivity check: %v", err))
	}

	return result
}

// Submodule represents a git submodule
type Submodule struct {
	Name   string
	Path   string
	URL    string
	Commit string
}

// Helper methods for submodule validation

// getUntrackedFiles returns a list of untracked files
func (g *Manager) getUntrackedFiles() ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []string{}, nil
	}

	return strings.Split(output, "\n"), nil
}

// checkRemoteStatus checks if the current branch is up to date with remote
func (g *Manager) checkRemoteStatus(branch string) error {
	if branch == "" {
		return fmt.Errorf("no branch specified")
	}

	// Check if remote exists
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no remote origin configured")
	}

	// Fetch to get latest remote refs (but don't show output)
	cmd = exec.Command("git", "fetch", "--dry-run")
	cmd.Run() // Ignore errors, this is just a connectivity check

	// Check ahead/behind status
	cmd = exec.Command("git", "rev-list", "--count", "--left-right", fmt.Sprintf("origin/%s...HEAD", branch))
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot compare with remote branch")
	}

	output := strings.TrimSpace(stdout.String())
	parts := strings.Fields(output)
	if len(parts) != 2 {
		return nil
	}

	behind, ahead := parts[0], parts[1]
	if behind != "0" && ahead != "0" {
		return fmt.Errorf("branch is %s commits behind and %s commits ahead of origin", behind, ahead)
	} else if behind != "0" {
		return fmt.Errorf("branch is %s commits behind origin", behind)
	} else if ahead != "0" {
		return fmt.Errorf("branch is %s commits ahead of origin", ahead)
	}

	return nil
}

// getSubmodules returns a list of git submodules
func (g *Manager) getSubmodules() ([]Submodule, error) {
	// First check if .gitmodules exists
	cmd := exec.Command("git", "ls-files", ".gitmodules")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	
	if err := cmd.Run(); err != nil || strings.TrimSpace(stdout.String()) == "" {
		// No .gitmodules file, so no submodules
		return []Submodule{}, nil
	}

	// Get submodule status
	cmd = exec.Command("git", "submodule", "status")
	stdout.Reset()
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return []Submodule{}, fmt.Errorf("failed to get submodule status: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return []Submodule{}, nil
	}

	var submodules []Submodule
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse submodule status line format: "[status]commit path (describe)"
		// Status chars: ' ' = current, '-' = not initialized, '+' = different commit, 'U' = merge conflicts
		// Note: Some lines may not have a leading space (different status)
		
		if len(line) < 41 { // minimum length for a commit hash (40) + path
			continue
		}

		var statusChar byte
		var commit, path string
		
		// Check if line starts with a 40-character hex string (commit hash)
		// If so, there's no status character prefix
		if len(line) >= 40 && isHexString(line[:40]) {
			// No status character - this is the commit hash directly
			statusChar = ' ' // Assume current status
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				commit = parts[0]
				path = parts[1]
			}
		} else {
			// Has status character prefix
			statusChar = line[0]
			rest := line[1:]
			parts := strings.Fields(rest)
			if len(parts) >= 2 {
				commit = parts[0]
				path = parts[1]
			}
		}
		
		if commit == "" || path == "" {
			continue
		}
		
		// Extract name from path (use last component)
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}

		submodules = append(submodules, Submodule{
			Name:   name,
			Path:   path,
			Commit: commit,
			URL:    "", // We'll populate this if needed
		})

		// Note: statusChar can be used for additional validation if needed
		_ = statusChar
	}

	return submodules, nil
}

// isSubmodulePointingToTag checks if a submodule is pointing to a git tag
func (g *Manager) isSubmodulePointingToTag(submodulePath string) (bool, string, error) {
	// Check if the submodule directory exists and is initialized
	// Modern git uses .git files that point to the actual git directory
	cmd := exec.Command("git", "-C", submodulePath, "rev-parse", "--git-dir")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, "", fmt.Errorf("submodule %s is not initialized: %v (stderr: %s)", submodulePath, err, stderr.String())
	}

	// Get the commit hash that the submodule is currently pointing to
	// Use git rev-parse HEAD in the submodule directory
	cmd = exec.Command("git", "-C", submodulePath, "rev-parse", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, "", fmt.Errorf("failed to get submodule HEAD commit for %s: %v (stderr: %s)", submodulePath, err, stderr.String())
	}

	currentCommit := strings.TrimSpace(stdout.String())

	// Check if this commit corresponds to any tags in the submodule
	cmd = exec.Command("git", "-C", submodulePath, "tag", "--points-at", currentCommit)
	stdout.Reset()
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// If tag command fails, assume no tags point to this commit
		return false, "", nil
	}

	tagOutput := strings.TrimSpace(stdout.String())
	
	if tagOutput == "" {
		return false, "", nil
	}

	// Return first tag found
	tags := strings.Split(tagOutput, "\n")
	return true, tags[0], nil
}

// submoduleHasChanges checks if a submodule has uncommitted changes
func (g *Manager) submoduleHasChanges(submodulePath string) (bool, error) {
	cmd := exec.Command("git", "-C", submodulePath, "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check submodule status: %v", err)
	}

	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

// checkGitConnectivity checks basic git connectivity
func (g *Manager) checkGitConnectivity() error {
	cmd := exec.Command("git", "remote", "-v")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no git remotes configured")
	}
	return nil
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
