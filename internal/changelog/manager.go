package changelog

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"bump-tui/internal/git"
)

type Manager struct {
	gitManager *git.Manager
}

type ChangeEntry struct {
	Type        string `json:"type"`
	Scope       string `json:"scope,omitempty"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
}

func NewManager() *Manager {
	return &Manager{
		gitManager: git.NewManager(),
	}
}

func (c *Manager) GenerateChanges(fromVersion string) (string, error) {
	commits, err := c.gitManager.GetCommitsSince(fromVersion)
	if err != nil {
		// If we can't get commits, return a default message
		return "- Minor updates and improvements", nil
	}

	// Try Claude first if available
	if c.isClaudeAvailable() {
		if changelog, err := c.generateWithClaude(commits); err == nil {
			return changelog, nil
		}
		// If Claude fails, continue to fallback
	}

	// Fallback to existing regex-based system
	return c.generateWithRegex(commits), nil
}

func (c *Manager) generateWithRegex(commits []git.Commit) string {
	var changes []string
	for _, commit := range commits {
		// Skip version bump commits
		if strings.Contains(commit.Message, "bump version") ||
			strings.Contains(commit.Message, "release") ||
			strings.Contains(commit.Message, "chore(release)") {
			continue
		}

		if formatted := c.formatCommitMessage(commit.Message); formatted != "" {
			changes = append(changes, formatted)
		}
	}

	if len(changes) == 0 {
		return "- Minor updates and improvements"
	}

	return strings.Join(changes, "\n")
}

func (c *Manager) formatCommitMessage(message string) string {
	if message == "" {
		return ""
	}

	// Extract first line only
	firstLine := strings.Split(message, "\n")[0]
	firstLine = strings.TrimSpace(firstLine)

	// Parse conventional commit format: type(scope): description
	conventionalRe := regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?: (.+)$`)
	matches := conventionalRe.FindStringSubmatch(firstLine)

	if len(matches) >= 4 {
		commitType := matches[1]
		scope := matches[2]
		description := matches[3]

		emoji := c.getEmojiForType(commitType)
		
		if scope != "" {
			return fmt.Sprintf("- %s **%s:** %s", emoji, scope, description)
		}
		return fmt.Sprintf("- %s %s", emoji, description)
	}

	// Non-conventional commit, just add a generic emoji
	return fmt.Sprintf("- 🔧 %s", firstLine)
}

func (c *Manager) getEmojiForType(commitType string) string {
	emojiMap := map[string]string{
		"feat":     "✨",
		"fix":      "🐛",
		"docs":     "📚",
		"style":    "💎",
		"refactor": "♻️",
		"perf":     "⚡️",
		"test":     "✅",
		"build":    "📦",
		"ci":       "👷",
		"chore":    "🔧",
		"revert":   "⏪",
		"merge":    "🔀",
	}

	if emoji, exists := emojiMap[commitType]; exists {
		return emoji
	}

	return "🔧" // Default emoji
}

func (c *Manager) UpdateChangelog(version, changes string) error {
	changelogDir := "docs"
	changelogPath := filepath.Join(changelogDir, "CHANGELOG.md")

	// Create docs directory if it doesn't exist
	if err := os.MkdirAll(changelogDir, 0755); err != nil {
		return fmt.Errorf("failed to create docs directory: %v", err)
	}

	// Generate new content
	date := time.Now().Format("2006-01-02")
	newContent := fmt.Sprintf("# %s (%s)\n\n%s\n\n", version, date, changes)

	// Read existing content
	existingContent := ""
	if content, err := os.ReadFile(changelogPath); err == nil {
		existingContent = string(content)
	}

	// Combine content
	var finalContent string
	if existingContent == "" {
		finalContent = "# Changelog\n\n" + newContent
	} else {
		// Find position after "# Changelog" header
		if pos := strings.Index(existingContent, "# Changelog"); pos >= 0 {
			headerEnd := pos + len("# Changelog")
			// Skip to end of line
			if newlinePos := strings.Index(existingContent[headerEnd:], "\n"); newlinePos >= 0 {
				headerEnd += newlinePos + 1
			}
			
			finalContent = existingContent[:headerEnd] + "\n" + newContent + existingContent[headerEnd:]
		} else {
			// No header found, prepend everything
			finalContent = "# Changelog\n\n" + newContent + existingContent
		}
	}

	// Write updated content
	if err := os.WriteFile(changelogPath, []byte(finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write changelog: %v", err)
	}

	return nil
}

func (c *Manager) PreviewChanges(fromVersion string) (string, error) {
	return c.GenerateChanges(fromVersion)
}

func (c *Manager) IsClaudeAvailable() bool {
	// Try common Claude locations
	claudePaths := []string{
		"claude",                                    // In PATH
		"/Users/" + os.Getenv("USER") + "/.claude/local/claude", // Common install location
		"/opt/homebrew/bin/claude",                 // Homebrew
		"/usr/local/bin/claude",                    // System install
	}
	
	for _, claudePath := range claudePaths {
		cmd := exec.Command(claudePath, "--version")
		cmd.Stdout = nil // Suppress output
		cmd.Stderr = nil // Suppress errors
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	
	return false
}

func (c *Manager) isClaudeAvailable() bool {
	return c.IsClaudeAvailable()
}

func (c *Manager) formatCommitsForClaude(commits []git.Commit) string {
	var commitText strings.Builder
	for _, commit := range commits {
		// Skip version bump commits
		if strings.Contains(commit.Message, "bump version") ||
			strings.Contains(commit.Message, "release") ||
			strings.Contains(commit.Message, "chore(release)") {
			continue
		}
		commitText.WriteString(fmt.Sprintf("- %s\n", commit.Message))
	}
	return commitText.String()
}

func (c *Manager) buildSimplePrompt(commits []git.Commit) string {
	commitMessages := c.formatCommitsForClaude(commits)
	
	return fmt.Sprintf(`Please format these git commit messages into a clean changelog:

%s

Requirements:
- Use markdown bullet points (-)
- Group changes by category (Features, Bug Fixes, Improvements, Other)
- Rewrite commit messages to be user-friendly
- Focus on what changed, not technical details
- Skip merge commits and version bumps

Output format:
## Features
- New feature description

## Bug Fixes  
- Fixed issue description

## Improvements
- Enhancement description

## Other
- Misc changes
`, commitMessages)
}

func (c *Manager) getClaudePath() string {
	// Try common Claude locations
	claudePaths := []string{
		"claude",                                    // In PATH
		"/Users/" + os.Getenv("USER") + "/.claude/local/claude", // Common install location
		"/opt/homebrew/bin/claude",                 // Homebrew
		"/usr/local/bin/claude",                    // System install
	}
	
	for _, claudePath := range claudePaths {
		cmd := exec.Command(claudePath, "--version")
		cmd.Stdout = nil // Suppress output
		cmd.Stderr = nil // Suppress errors
		if err := cmd.Run(); err == nil {
			return claudePath
		}
	}
	
	return "" // Not found
}

func (c *Manager) generateWithClaude(commits []git.Commit) (string, error) {
	if len(commits) == 0 {
		return "- Minor updates and improvements", nil
	}

	claudePath := c.getClaudePath()
	if claudePath == "" {
		return "", fmt.Errorf("claude not found")
	}

	prompt := c.buildSimplePrompt(commits)
	
	cmd := exec.Command(claudePath, "-p", prompt)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude command failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("claude returned empty output")
	}

	return output, nil
}