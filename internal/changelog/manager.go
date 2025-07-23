package changelog

import (
	"fmt"
	"os"
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
		return "- 🔧 Minor updates and improvements", nil
	}

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
		return "- 🔧 Minor updates and improvements", nil
	}

	return strings.Join(changes, "\n"), nil
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