package models

import (
	"fmt"
	"strings"

	"bump-tui/internal/changelog"
	"bump-tui/internal/git"
	"bump-tui/internal/version"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sessionState int

const (
	welcomeView sessionState = iota
	versionSelectView
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
	state           sessionState
	keys            keyMap
	width           int
	height          int
	err             error
	
	// Managers
	versionManager   *version.Manager
	gitManager       *git.Manager
	changelogManager *changelog.Manager
	
	// UI components
	versionList    list.Model
	changelogView  viewport.Model
	
	// State data
	selectedBump    bumpType
	generatedChanges string
	newVersion      string
	showHelp        bool
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
	
	versionList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	versionList.Title = "Select Version Bump Type"
	
	changelogView := viewport.New(0, 0)
	
	return MainModel{
		state:            welcomeView,
		keys:             keys,
		versionManager:   versionManager,
		gitManager:       gitManager,
		changelogManager: changelogManager,
		versionList:      versionList,
		changelogView:    changelogView,
	}
}

type initDoneMsg struct {
	projectFiles []version.ProjectFile
	currentVersion string
	err error
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
		projectFiles: m.versionManager.ProjectFiles,
		currentVersion: m.versionManager.CurrentVersion.String(),
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
		m.changelogView.Width = msg.Width - 4
		m.changelogView.Height = msg.Height - 12
		
		return m, nil
		
	case initDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		
		// Project initialized successfully, move to version selection
		m.state = versionSelectView
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
		case versionSelectView:
			return m.updateVersionSelect(msg)
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
			
			// Generate changelog preview
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
		return m, m.performVersionBump
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
	
	return "success"
}

func (m MainModel) View() string {
	if m.err != nil {
		return m.errorView()
	}

	switch m.state {
	case welcomeView:
		return m.welcomeView()
	case versionSelectView:
		return m.versionSelectView()
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
		Foreground(lipgloss.Color("#FF6B6B")).
		Bold(true)
	
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		errorStyle.Render("❌ Error"),
		"",
		m.err.Error(),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press q to quit"),
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
		Foreground(lipgloss.Color("#626262"))
	
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
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)
	
	versionInfo := versionInfoStyle.Render(
		fmt.Sprintf("%s → %s", m.versionManager.CurrentVersion.String(), m.newVersion),
	)
	
	changelogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#626262")).
		Padding(1).
		Width(m.width - 8)
	
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
		Foreground(lipgloss.Color("#FFA500")).
		Bold(true)
	
	question := questionStyle.Render("Are you sure you want to proceed?")
	
	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))
	
	summary := summaryStyle.Render(
		fmt.Sprintf("This will:\n• Update version to %s\n• Update changelog\n• Create git commit\n• Create git tag v%s",
			m.newVersion, m.newVersion),
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
		Foreground(lipgloss.Color("#04B575"))
	
	spinner := spinnerStyle.Render("⠋ Updating version files...")
	
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
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)
	
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		successStyle.Render("✅ Success!"),
		"",
		fmt.Sprintf("Version bumped to %s", m.newVersion),
		fmt.Sprintf("Created tag v%s", m.newVersion),
		"Updated changelog",
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press q to quit"),
	)
	
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (m MainModel) headerView(title string) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width)
	
	return titleStyle.Render("🚀 Bump - " + title)
}

func (m MainModel) footerView(help string) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Align(lipgloss.Center).
		Width(m.width)
	
	return helpStyle.Render(help)
}

func (m MainModel) projectFilesView() string {
	if len(m.versionManager.ProjectFiles) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Render("⚠️ No project files detected")
	}
	
	var files []string
	for _, file := range m.versionManager.ProjectFiles {
		fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
		files = append(files, fileStyle.Render(fmt.Sprintf("• %s", file.Description)))
	}
	
	return strings.Join(files, "\n")
}

func (m MainModel) welcomeView() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		Render("🚀 Bump - Version Manager")
	
	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
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