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
	if err := g.runGitCommand("push", "--follow-tags", "origin", "HEAD"); err != nil {
		return fmt.Errorf("failed to push changes: %v", err)
	}
	return nil
}

func (g *Manager) CreateGitHubRelease(version, changelog string) error {
	tagName := fmt.Sprintf("v%s", version)
	title := fmt.Sprintf("Release %s", tagName)

	args := []string{
		"release",
		"create",
		tagName,
		"--title", title,
		"--notes", changelog,
		"--verify-tag",
		"--latest",
	}

	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create GitHub release: %v\nError: %s", err, stderr.String())
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