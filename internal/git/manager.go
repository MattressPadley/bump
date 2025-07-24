package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// GitCommandTimeout is the default timeout for git operations
	GitCommandTimeout = 30 * time.Second
	// CommitHashLength is the expected length of a git commit hash
	CommitHashLength = 40
	// MaxCommitsToAnalyze is the maximum number of commits to analyze when no previous tag exists
	MaxCommitsToAnalyze = 10
	// ValidationStepCount is the total number of validation steps performed
	ValidationStepCount = 6
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

// validateSubmodulePath validates that a submodule path is safe and within repository bounds
func (g *Manager) validateSubmodulePath(path string) error {
	// Reject empty paths
	if path == "" {
		return fmt.Errorf("submodule path cannot be empty")
	}

	// Clean the path to normalize it and resolve any path traversal attempts
	cleanPath := filepath.Clean(path)
	
	// Reject absolute paths (check both original and cleaned paths)
	if filepath.IsAbs(path) || filepath.IsAbs(cleanPath) {
		return fmt.Errorf("submodule path cannot be absolute: %s", path)
	}

	// Reject paths with path traversal attempts (check both original and cleaned)
	if strings.Contains(path, "..") || strings.Contains(cleanPath, "..") {
		return fmt.Errorf("submodule path contains path traversal: %s", path)
	}

	// Reject paths with Windows drive letters
	if len(path) >= 2 && path[1] == ':' {
		return fmt.Errorf("submodule path cannot contain drive letters: %s", path)
	}

	// Reject paths starting with ~ (home directory)
	if strings.HasPrefix(path, "~") || strings.HasPrefix(cleanPath, "~") {
		return fmt.Errorf("submodule path cannot start with ~: %s", path)
	}

	// Additional boundary checks on the cleaned path
	// Ensure the cleaned path doesn't try to escape the current directory
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return fmt.Errorf("submodule path tries to escape repository bounds: %s (resolved to: %s)", path, cleanPath)
	}

	// Reject paths that resolve to the root directory
	if cleanPath == "." || cleanPath == "/" {
		return fmt.Errorf("submodule path cannot resolve to root directory: %s", path)
	}

	// Check for suspicious path elements that could indicate malicious intent
	pathElements := strings.Split(cleanPath, string(filepath.Separator))
	for _, element := range pathElements {
		if element == "" {
			continue // Skip empty elements (can occur with consecutive separators)
		}
		// Reject paths with null bytes or other control characters
		if strings.ContainsAny(element, "\x00\n\r\t") {
			return fmt.Errorf("submodule path contains invalid characters: %s", path)
		}
	}

	return nil
}


func (g *Manager) IsGitRepository() error {
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

func (g *Manager) CommitVersionBump(version string) error {
	// Add all changes
	if err := g.runGitCommand("add", "."); err != nil {
		return fmt.Errorf("unable to stage changes for commit. Ensure you have write permissions: %v", err)
	}

	// Create commit
	message := fmt.Sprintf("chore(release): bump version to %s", version)
	if err := g.runGitCommand("commit", "-m", message); err != nil {
		return fmt.Errorf("unable to create version bump commit. Check git configuration: %v", err)
	}

	return nil
}

func (g *Manager) CreateTag(version string) error {
	tagName := fmt.Sprintf("v%s", version)
	message := fmt.Sprintf("Release version %s", version)

	if err := g.runGitCommand("tag", "-a", tagName, "-m", message); err != nil {
		return fmt.Errorf("unable to create git tag %s. Tag may already exist: %v", tagName, err)
	}

	return nil
}

func (g *Manager) PushChanges() error {
	// Push commits first
	if err := g.runGitCommand("push", "origin", "HEAD"); err != nil {
		return fmt.Errorf("unable to push commits to remote. Check network and permissions: %v", err)
	}
	return nil
}

func (g *Manager) PushTag(version string) error {
	tagName := fmt.Sprintf("v%s", version)
	// Push tag separately to ensure workflow triggers
	if err := g.runGitCommand("push", "origin", tagName); err != nil {
		return fmt.Errorf("unable to push tag %s to remote. Check network and permissions: %v", tagName, err)
	}
	return nil
}

func (g *Manager) GetCommitsSince(fromVersion string) ([]Commit, error) {
	var args []string
	if fromVersion != "" {
		tagName := fmt.Sprintf("v%s", fromVersion)
		// First check if the tag exists
		ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
		checkCmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", tagName)
		if err := checkCmd.Run(); err != nil {
			// Tag doesn't exist, get all commits instead
			args = []string{"log", "--oneline", "--no-merges", fmt.Sprintf("-%d", MaxCommitsToAnalyze)} // Limit to last N commits
		} else {
			args = []string{"log", "--oneline", "--no-merges", fmt.Sprintf("%s..HEAD", tagName)}
		}
		cancel()
	} else {
		args = []string{"log", "--oneline", "--no-merges", fmt.Sprintf("-%d", MaxCommitsToAnalyze)} // Limit to last N commits
	}

	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
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
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("unable to determine current git branch: %v", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (g *Manager) HasUncommittedChanges() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("unable to check repository status: %v", err)
	}

	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

func (g *Manager) runGitCommand(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
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
	Results     []ValidationResult
	HasErrors   bool
	HasWarnings bool
	CanProceed  bool
}

// ValidateRepositoryStatus performs comprehensive git repository validation
func (g *Manager) ValidateRepositoryStatus() (*ValidationSummary, error) {
	steps := []ValidationStep{
		{Name: "repository", Description: "Checking repository status...", Index: 1, Total: ValidationStepCount},
		{Name: "working_dir", Description: "Validating working directory...", Index: 2, Total: ValidationStepCount},
		{Name: "branch", Description: "Checking branch status...", Index: 3, Total: ValidationStepCount},
		{Name: "submodules_scan", Description: "Scanning for submodules...", Index: 4, Total: ValidationStepCount},
		{Name: "submodules_status", Description: "Validating submodule states...", Index: 5, Total: ValidationStepCount},
		{Name: "final", Description: "Final validation checks...", Index: 6, Total: ValidationStepCount},
	}

	// Run independent validations in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]ValidationResult, ValidationStepCount) // Pre-allocate for all validation steps
	hasErrors := false
	hasWarnings := false

	// Channel for collecting errors from goroutines
	errChan := make(chan error, 3)

	// Step 1: Check repository status (parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.validateRepositoryStatus(steps[0])
		mu.Lock()
		results[0] = result
		if !result.Success {
			hasErrors = true
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
		}
		mu.Unlock()
	}()

	// Step 2: Validate working directory (parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.validateWorkingDirectory(steps[1])
		mu.Lock()
		results[1] = result
		if !result.Success {
			hasErrors = true
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
		}
		mu.Unlock()
	}()

	// Step 3: Check branch status (parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := g.validateBranchStatus(steps[2])
		mu.Lock()
		results[2] = result
		if !result.Success {
			hasErrors = true
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
		}
		mu.Unlock()
	}()

	// Wait for parallel operations to complete
	wg.Wait()
	close(errChan)
	
	// Check if any errors occurred in goroutines
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	// Step 4: Scan for submodules (sequential - others depend on it)
	submodules, result := g.scanSubmodules(steps[3])
	results[3] = result
	if !result.Success {
		hasErrors = true
	}
	if len(result.Warnings) > 0 {
		hasWarnings = true
	}

	// Step 5: Validate submodules (sequential - depends on step 4)
	if len(submodules) > 0 {
		result = g.validateSubmodules(steps[4], submodules)
		results[4] = result
		if !result.Success {
			hasErrors = true
		}
		if len(result.Warnings) > 0 {
			hasWarnings = true
		}
	} else {
		// Skip submodule validation if no submodules
		results[4] = ValidationResult{
			Step:     steps[4],
			Success:  true,
			Warnings: nil,
			Errors:   nil,
		}
	}

	// Step 6: Final validation (can run independently but do it last for logical flow)
	result = g.performFinalValidation(steps[5])
	results[5] = result
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
		result.Errors = append(result.Errors, "Current directory is not a git repository. Run 'git init' or navigate to a git repository.")
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
		result.Errors = append(result.Errors, "Working directory has uncommitted changes. Commit or stash changes before proceeding.")
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
		// Validate submodule path for security
		if err := g.validateSubmodulePath(submodule.Path); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Insecure submodule path %s: %v", submodule.Name, err))
			result.Success = false
			continue
		}

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
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
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
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	if err := cmd.Run(); err != nil {
		cancel()
		return fmt.Errorf("no remote origin configured")
	}
	cancel()

	// Fetch to get latest remote refs (but don't show output)
	ctx, cancel = context.WithTimeout(context.Background(), GitCommandTimeout)
	cmd = exec.CommandContext(ctx, "git", "fetch", "--dry-run")
	var fetchErr bytes.Buffer
	cmd.Stderr = &fetchErr
	fetchResult := cmd.Run()
	cancel()

	// Analyze fetch errors for specific issues
	if fetchResult != nil {
		fetchErrMsg := strings.TrimSpace(fetchErr.String())
		
		// Classify error type based on error message patterns
		if fetchErrMsg != "" {
			errLower := strings.ToLower(fetchErrMsg)
			switch {
			case strings.Contains(errLower, "authentication failed") || 
				 strings.Contains(errLower, "permission denied") ||
				 strings.Contains(errLower, "access denied"):
				return fmt.Errorf("authentication failed - check your credentials: %v", fetchErrMsg)
			case strings.Contains(errLower, "network") || 
				 strings.Contains(errLower, "connection") ||
				 strings.Contains(errLower, "timeout") ||
				 strings.Contains(errLower, "unreachable"):
				return fmt.Errorf("network connectivity issue - check internet connection: %v", fetchErrMsg)
			case strings.Contains(errLower, "repository not found") ||
				 strings.Contains(errLower, "does not exist"):
				return fmt.Errorf("remote repository not found - check remote URL: %v", fetchErrMsg)
			default:
				return fmt.Errorf("remote connectivity issue: %v", fetchErrMsg)
			}
		}
		return fmt.Errorf("unable to fetch from remote - check network connection and credentials")
	}

	// Check ahead/behind status
	ctx, cancel = context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--count", "--left-right", fmt.Sprintf("origin/%s...HEAD", branch))
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
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	cmd := exec.CommandContext(ctx, "git", "ls-files", ".gitmodules")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil || strings.TrimSpace(stdout.String()) == "" {
		cancel()
		// No .gitmodules file, so no submodules
		return []Submodule{}, nil
	}
	cancel()

	// Get submodule status
	ctx, cancel = context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, "git", "submodule", "status")
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

		submodule, err := g.parseSubmoduleStatusLine(line)
		if err != nil {
			// Skip malformed lines but don't fail entirely
			continue
		}

		// Validate submodule path for security before processing
		if err := g.validateSubmodulePath(submodule.Path); err != nil {
			// Skip insecure submodule paths but don't fail entirely
			continue
		}

		submodules = append(submodules, submodule)
	}

	return submodules, nil
}

// parseSubmoduleStatusLine parses a single line from 'git submodule status' output
// Format: "[status]commit path (describe)" where status is optional
func (g *Manager) parseSubmoduleStatusLine(line string) (Submodule, error) {
	// Check minimum length for a commit hash + path
	if len(line) < CommitHashLength+1 {
		return Submodule{}, fmt.Errorf("line too short: %s", line)
	}

	var statusChar byte
	var commit, path string

	// Check if line starts with a commit hash (no status character prefix)
	if len(line) >= CommitHashLength && isHexString(line[:CommitHashLength]) {
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
		return Submodule{}, fmt.Errorf("could not parse commit and path from: %s", line)
	}

	// Validate commit hash format
	if len(commit) != CommitHashLength || !isHexString(commit) {
		return Submodule{}, fmt.Errorf("invalid commit hash format: %s", commit)
	}

	// Extract name from path (use last component)
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}

	// Note: statusChar can be used for additional validation if needed
	_ = statusChar

	return Submodule{
		Name:   name,
		Path:   path,
		Commit: commit,
		URL:    "", // We'll populate this if needed
	}, nil
}

// isSubmodulePointingToTag checks if a submodule is pointing to a git tag
func (g *Manager) isSubmodulePointingToTag(submodulePath string) (bool, string, error) {
	// Check if the submodule directory exists and is initialized
	// Modern git uses .git files that point to the actual git directory
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	cmd := exec.CommandContext(ctx, "git", "-C", submodulePath, "rev-parse", "--git-dir")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		cancel()
		return false, "", fmt.Errorf("submodule %s is not initialized: %v (stderr: %s)", submodulePath, err, stderr.String())
	}
	cancel()

	// Get the commit hash that the submodule is currently pointing to
	// Use git rev-parse HEAD in the submodule directory
	ctx, cancel = context.WithTimeout(context.Background(), GitCommandTimeout)
	cmd = exec.CommandContext(ctx, "git", "-C", submodulePath, "rev-parse", "HEAD")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	stderr.Reset()
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		cancel()
		return false, "", fmt.Errorf("failed to get submodule HEAD commit for %s: %v (stderr: %s)", submodulePath, err, stderr.String())
	}
	cancel()

	currentCommit := strings.TrimSpace(stdout.String())

	// Check if this commit corresponds to any tags in the submodule
	ctx, cancel = context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, "git", "-C", submodulePath, "tag", "--points-at", currentCommit)
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
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", submodulePath, "status", "--porcelain")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to check submodule status: %v", err)
	}

	return len(strings.TrimSpace(stdout.String())) > 0, nil
}

// checkGitConnectivity checks basic git connectivity
func (g *Manager) checkGitConnectivity() error {
	ctx, cancel := context.WithTimeout(context.Background(), GitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "remote", "-v")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no git remotes configured")
	}
	return nil
}

// isHexString checks if a string contains only hexadecimal characters
func isHexString(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
