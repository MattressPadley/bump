package models

import (
	"fmt"
	"strings"

	"bump-tui/internal/changelog"
	"bump-tui/internal/git"
	"bump-tui/internal/version"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
	welcomeView sessionState = iota
	validationView
	versionSelectView
	changelogGeneratingView
	changelogPreviewView
	confirmationView
	progressView
	resultsView
)

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Help  key.Binding
	Quit  key.Binding
	Enter key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Help, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}

type bumpType int

const (
	bumpMajor bumpType = iota
	bumpMinor
	bumpPatch
)

func (b bumpType) String() string {
	switch b {
	case bumpMajor:
		return "Major"
	case bumpMinor:
		return "Minor"
	case bumpPatch:
		return "Patch"
	default:
		return "Unknown"
	}
}

type versionItem struct {
	title string
	desc  string
	bump  bumpType
}

func (i versionItem) Title() string       { return i.title }
func (i versionItem) Description() string { return i.desc }
func (i versionItem) FilterValue() string { return i.title }

type MainModel struct {
	state  sessionState
	keys   keyMap
	width  int
	height int
	err    error

	// Managers
	versionManager   *version.Manager
	gitManager       *git.Manager
	changelogManager *changelog.Manager

	// UI components
	versionList   list.Model
	changelogView viewport.Model
	spinner       spinner.Model

	// State data
	selectedBump          bumpType
	generatedChanges      string
	newVersion            string
	showHelp              bool
	claudeEnabled         bool
	validationSummary *git.ValidationSummary
}

func NewMainModel() MainModel {
	// Initialize managers
	versionManager := version.NewManager()
	gitManager := git.NewManager()
	changelogManager := changelog.NewManager()

	// Create version selection items
	items := []list.Item{
		versionItem{
			title: "Major (x.0.0)",
			desc:  "Breaking changes - incompatible API changes",
			bump:  bumpMajor,
		},
		versionItem{
			title: "Minor (0.x.0)",
			desc:  "New features - backwards compatible functionality",
			bump:  bumpMinor,
		},
		versionItem{
			title: "Patch (0.0.x)",
			desc:  "Bug fixes - backwards compatible fixes",
			bump:  bumpPatch,
		},
	}

	// Create custom delegate with Catppuccin colors
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#8aadf4")).
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#8aadf4")).
		Foreground(lipgloss.Color("#6e738d")).
		Padding(0, 0, 0, 1)
	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cad3f5")).
		Padding(0, 0, 0, 1)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e738d")).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#5b6078")).
		Padding(0, 0, 0, 1)
	delegate.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#494d64")).
		Padding(0, 0, 0, 1)

	versionList := list.New(items, delegate, 0, 0)
	versionList.Title = "Select Version Bump Type"
	versionList.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true).
		Padding(0, 1)

	changelogView := viewport.New(0, 0)

	// Initialize spinner for Claude processing
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#8aadf4"))

	// Progress bar removed - using spinner for validation since it's instantaneous

	// Check if Claude is available
	claudeAvailable := changelogManager.IsClaudeAvailable()

	return MainModel{
		state:            welcomeView,
		keys:             keys,
		versionManager:   versionManager,
		gitManager:       gitManager,
		changelogManager: changelogManager,
		versionList:      versionList,
		changelogView:    changelogView,
		spinner:          s,
		claudeEnabled:    claudeAvailable,
	}
}

type initDoneMsg struct {
	projectFiles   []version.ProjectFile
	currentVersion string
	err            error
}

type changelogGeneratedMsg struct {
	changes string
	err     error
}


type validationCompleteMsg struct {
	summary *git.ValidationSummary
	err     error
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.initProject,
	)
}

func (m MainModel) initProject() tea.Msg {
	// Check if we're in a git repository
	if err := m.gitManager.IsGitRepository(); err != nil {
		return initDoneMsg{err: err}
	}

	// Detect version files
	if err := m.versionManager.DetectVersionFiles("."); err != nil {
		return initDoneMsg{err: err}
	}

	return initDoneMsg{
		projectFiles:   m.versionManager.ProjectFiles,
		currentVersion: m.versionManager.CurrentVersion.String(),
	}
}

func (m MainModel) generateChangelog() tea.Msg {
	changes, err := m.changelogManager.GenerateChanges(m.versionManager.CurrentVersion.String())
	return changelogGeneratedMsg{
		changes: changes,
		err:     err,
	}
}

func (m MainModel) validateRepository() tea.Cmd {
	return func() tea.Msg {
		// Provide a no-op callback since TUI doesn't need progress updates during validation
		noOpCallback := func(result git.ValidationResult) {
			// Progress updates during validation are not needed in TUI mode
			// All results are shown together after validation completes
		}

		summary, err := m.gitManager.ValidateRepositoryStatus(noOpCallback)
		if err != nil {
			return validationCompleteMsg{err: err}
		}

		return validationCompleteMsg{summary: summary}
	}
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update sub-components
		m.versionList.SetWidth(msg.Width - 4)
		m.versionList.SetHeight(msg.Height - 8)
		m.changelogView.Width = msg.Width - 12   // Account for border + padding
		m.changelogView.Height = msg.Height - 12 // Account for header, version info, footer, spacing, and borders

		return m, nil

	case initDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Project initialized successfully, move to validation
		m.state = validationView
		return m, tea.Batch(
			m.validateRepository(),
			m.spinner.Tick,
		)

	case validationCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.validationSummary = msg.summary

		// Always stay on validation view to show results
		// User must press enter to continue or see errors
		return m, nil

	case changelogGeneratedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.generatedChanges = msg.changes
		m.changelogView.SetContent(msg.changes)
		m.state = changelogPreviewView
		return m, nil

	case spinner.TickMsg:
		if m.state == validationView || m.state == changelogGeneratingView || m.state == progressView {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case string:
		if msg == "success" {
			m.state = resultsView
			return m, nil
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

		// Handle state-specific key events
		switch m.state {
		case validationView:
			return m.updateValidation(msg)
		case versionSelectView:
			return m.updateVersionSelect(msg)
		case changelogGeneratingView:
			// Only allow quit during changelog generation
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			return m, nil
		case changelogPreviewView:
			return m.updateChangelogPreview(msg)
		case confirmationView:
			return m.updateConfirmation(msg)
		case resultsView:
			return m, tea.Quit
		}

	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

func (m MainModel) updateValidation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		// If validation completed and can proceed, move to version selection
		if m.validationSummary != nil && m.validationSummary.CanProceed {
			m.state = versionSelectView
			return m, nil
		}
		// If validation failed, stay on validation view
		return m, nil
	}
	return m, nil
}

func (m MainModel) updateVersionSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		if selectedItem, ok := m.versionList.SelectedItem().(versionItem); ok {
			m.selectedBump = selectedItem.bump

			// Calculate new version
			switch m.selectedBump {
			case bumpMajor:
				m.newVersion = m.versionManager.BumpMajor().String()
			case bumpMinor:
				m.newVersion = m.versionManager.BumpMinor().String()
			case bumpPatch:
				m.newVersion = m.versionManager.BumpPatch().String()
			}

			// Show loading state if Claude is available, otherwise generate directly
			if m.claudeEnabled {
				m.state = changelogGeneratingView
				return m, tea.Batch(
					m.generateChangelog,
					m.spinner.Tick,
				)
			} else {
				// Generate changelog synchronously for non-Claude fallback
				changes, err := m.changelogManager.GenerateChanges(m.versionManager.CurrentVersion.String())
				if err != nil {
					m.err = err
					return m, nil
				}
				m.generatedChanges = changes
				m.changelogView.SetContent(changes)

				m.state = changelogPreviewView
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.versionList, cmd = m.versionList.Update(msg)
	return m, cmd
}

func (m MainModel) updateChangelogPreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		m.state = confirmationView
		return m, nil
	case key.Matches(msg, m.keys.Left):
		m.state = versionSelectView
		return m, nil
	}

	var cmd tea.Cmd
	m.changelogView, cmd = m.changelogView.Update(msg)
	return m, cmd
}

func (m MainModel) updateConfirmation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.state = progressView
		return m, tea.Batch(
			m.performVersionBump,
			m.spinner.Tick,
		)
	case "n", "N":
		m.state = versionSelectView
		return m, nil
	case "left", "h":
		m.state = changelogPreviewView
		return m, nil
	}

	return m, nil
}

func (m MainModel) performVersionBump() tea.Msg {
	// Update all version files
	if err := m.versionManager.UpdateAllVersions(m.newVersion); err != nil {
		return err
	}

	// Update changelog
	if err := m.changelogManager.UpdateChangelog(m.newVersion, m.generatedChanges); err != nil {
		return err
	}

	// Git operations
	if err := m.gitManager.CommitVersionBump(m.newVersion); err != nil {
		return err
	}

	if err := m.gitManager.CreateTag(m.newVersion); err != nil {
		return err
	}

	// Push changes and tag separately to GitHub (ensures workflow triggers)
	if err := m.gitManager.PushChanges(); err != nil {
		return err
	}

	if err := m.gitManager.PushTag(m.newVersion); err != nil {
		return err
	}

	return "success"
}

func (m MainModel) View() string {
	if m.err != nil {
		return m.errorView()
	}

	switch m.state {
	case welcomeView:
		return m.welcomeView()
	case validationView:
		return m.validationView()
	case versionSelectView:
		return m.versionSelectView()
	case changelogGeneratingView:
		return m.changelogGeneratingView()
	case changelogPreviewView:
		return m.changelogPreviewView()
	case confirmationView:
		return m.confirmationView()
	case progressView:
		return m.progressView()
	case resultsView:
		return m.resultsView()
	default:
		return "Unknown view"
	}
}

func (m MainModel) errorView() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ed8796")).
		Bold(true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		errorStyle.Render("❌ Error"),
		"",
		m.err.Error(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6e738d")).Render("Press q to quit"),
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) changelogGeneratingView() string {
	header := m.headerView("Generating Changelog")

	versionInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true)

	versionInfo := versionInfoStyle.Render(
		fmt.Sprintf("%s → %s", m.versionManager.CurrentVersion.String(), m.newVersion),
	)

	// Animated spinner with text
	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true)

	statusText := "Analyzing commits and generating changelog..."
	if m.claudeEnabled {
		statusText = "Using Claude to generate changelog..."
	}

	spinner := spinnerStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), statusText))

	footer := m.footerView("q: quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		versionInfo,
		"",
		"",
		spinner,
		"",
		"",
		footer,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) versionSelectView() string {
	header := m.headerView("Select Version Bump Type")

	currentVersionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e738d"))

	currentVersion := currentVersionStyle.Render(
		fmt.Sprintf("Current version: %s", m.versionManager.CurrentVersion.String()),
	)

	projectFiles := m.projectFilesView()

	footer := m.footerView("↑/↓: navigate • enter: select • q: quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		currentVersion,
		"",
		projectFiles,
		"",
		m.versionList.View(),
		"",
		footer,
	)

	return content
}

func (m MainModel) changelogPreviewView() string {
	header := m.headerView("Changelog Preview")

	versionInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true)

	versionInfo := versionInfoStyle.Render(
		fmt.Sprintf("%s → %s", m.versionManager.CurrentVersion.String(), m.newVersion),
	)

	changelogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#494d64")).
		Padding(1).
		Width(m.changelogView.Width + 4).  // Match viewport width + border/padding
		Height(m.changelogView.Height + 2) // Match viewport height + padding

	changelog := changelogStyle.Render(m.changelogView.View())

	footer := m.footerView("↑/↓: scroll • enter: continue • ←: back • q: quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		versionInfo,
		"",
		changelog,
		"",
		footer,
	)

	return content
}

func (m MainModel) confirmationView() string {
	header := m.headerView("Confirmation")

	questionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5a97f")).
		Bold(true)

	question := questionStyle.Render("Are you sure you want to proceed?")

	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e738d"))

	var actions []string
	actions = append(actions, fmt.Sprintf("• Update version to %s", m.newVersion))
	actions = append(actions, "• Update changelog")
	actions = append(actions, "• Create git commit")
	actions = append(actions, fmt.Sprintf("• Create git tag v%s", m.newVersion))
	actions = append(actions, "• Push changes to GitHub")
	actions = append(actions, "• Push tag to trigger release workflow")

	summary := summaryStyle.Render(
		fmt.Sprintf("This will:\n%s", strings.Join(actions, "\n")),
	)

	// Workflow info
	workflowInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4"))

	workflowInfo := workflowInfoStyle.Render(
		"The GitHub Actions workflow will build binaries and update Homebrew tap",
	)

	footer := m.footerView("y: yes • n: no • ←: back • q: quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		question,
		"",
		summary,
		"",
		workflowInfo,
		"",
		footer,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) progressView() string {
	header := m.headerView("Processing")

	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4"))

	spinner := spinnerStyle.Render(fmt.Sprintf("%s Updating version files...", m.spinner.View()))

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		spinner,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) resultsView() string {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6da95")).
		Bold(true)

	var results []string
	results = append(results, successStyle.Render("✅ Success!"))
	results = append(results, "")

	// This was a version bump
	results = append(results, fmt.Sprintf("Version bumped to %s", m.newVersion))
	results = append(results, fmt.Sprintf("Created tag v%s", m.newVersion))
	results = append(results, "Updated changelog")
	results = append(results, "Pushed changes to GitHub")
	results = append(results, "Pushed tag to trigger release workflow")
	results = append(results, "")
	results = append(results, "🚀 GitHub Actions will build binaries and update Homebrew tap")

	results = append(results, "")
	results = append(results, lipgloss.NewStyle().Foreground(lipgloss.Color("#6e738d")).Render("Press q to quit"))

	content := lipgloss.JoinVertical(lipgloss.Left, results...)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) headerView(title string) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width)

	return titleStyle.Render("🚀 Bump - " + title)
}

func (m MainModel) footerView(help string) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e738d")).
		Align(lipgloss.Center).
		Width(m.width)

	return helpStyle.Render(help)
}

func (m MainModel) projectFilesView() string {
	if len(m.versionManager.ProjectFiles) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5a97f")).
			Render("⚠️ No project files detected")
	}

	var files []string
	for _, file := range m.versionManager.ProjectFiles {
		fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6e738d"))
		files = append(files, fileStyle.Render(fmt.Sprintf("• %s", file.Description)))
	}

	return strings.Join(files, "\n")
}

func (m MainModel) validationView() string {
	header := m.headerView("Repository Validation")

	// Current step or completion status
	var statusText string
	var statusStyle lipgloss.Style

	if m.validationSummary == nil {
		// Still validating - show spinner
		statusText = fmt.Sprintf("%s Validating repository status...", m.spinner.View())
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8aadf4")).
			Bold(true)
	} else if !m.validationSummary.CanProceed {
		// Validation failed
		statusText = "❌ Validation Failed - Repository is not ready for version bump"
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ed8796")).
			Bold(true)
	} else if m.validationSummary.HasWarnings {
		// Validation passed with warnings
		statusText = "⚠️  Validation Complete - Warnings found but can proceed"
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5a97f")).
			Bold(true)
	} else {
		// Validation passed completely
		statusText = "✅ Validation Complete - Repository is ready"
		statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6da95")).
			Bold(true)
	}

	status := statusStyle.Render(statusText)

	// Results summary - ALWAYS show detailed results when available
	var resultsContent []string
	if m.validationSummary != nil {
		resultsContent = append(resultsContent,
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8aadf4")).
				Bold(true).
				Render("📋 Validation Results:"))
		resultsContent = append(resultsContent, "")

		for _, result := range m.validationSummary.Results {
			// Step name and status
			stepIcon := "✅"
			if !result.Success {
				stepIcon = "❌"
			} else if len(result.Warnings) > 0 {
				stepIcon = "⚠️ "
			}

			stepLine := fmt.Sprintf("%s %s", stepIcon, result.Step.Description)
			resultsContent = append(resultsContent, stepLine)

			// Add errors
			for _, err := range result.Errors {
				errorLine := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#ed8796")).
					Render(fmt.Sprintf("   • %s", err))
				resultsContent = append(resultsContent, errorLine)
			}

			// Add warnings
			for _, warning := range result.Warnings {
				warningLine := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#f5a97f")).
					Render(fmt.Sprintf("   • %s", warning))
				resultsContent = append(resultsContent, warningLine)
			}

			// For submodule validation step, add success info when no warnings
			if result.Step.Name == "submodules_status" && len(result.Warnings) == 0 && len(result.Errors) == 0 && result.Success {
				successLine := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#a6da95")).
					Render("   • All submodules point to release tags")
				resultsContent = append(resultsContent, successLine)
			}
		}

		// Add summary stats
		resultsContent = append(resultsContent, "")
		summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6e738d"))

		if m.validationSummary.HasErrors {
			resultsContent = append(resultsContent,
				summaryStyle.Render("❌ Found blocking errors - cannot proceed with version bump"))
		} else if m.validationSummary.HasWarnings {
			resultsContent = append(resultsContent,
				summaryStyle.Render(fmt.Sprintf("⚠️  Found %d validation warnings - can proceed with caution",
					m.countWarnings())))
		} else {
			resultsContent = append(resultsContent,
				summaryStyle.Render("✅ All validation checks passed - repository is ready"))
		}
	}

	results := strings.Join(resultsContent, "\n")

	// Footer instructions
	var footerText string
	if m.validationSummary == nil {
		footerText = "q: quit"
	} else if m.validationSummary.CanProceed {
		footerText = "enter: continue to version selection • q: quit"
	} else {
		footerText = "Fix errors and restart • q: quit"
	}

	footer := m.footerView(footerText)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		status,
		"",
		"",
		results,
		"",
		"",
		footer,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) countWarnings() int {
	if m.validationSummary == nil {
		return 0
	}
	count := 0
	for _, result := range m.validationSummary.Results {
		count += len(result.Warnings)
	}
	return count
}

func (m MainModel) welcomeView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#8aadf4")).
		Bold(true).
		Render("🚀 Bump - Version Manager")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6e738d")).
		Render("Interactive semantic version management tool")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		subtitle,
		"",
		"Detecting project files...",
		"",
		"Press q to quit",
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}
