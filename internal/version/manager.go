package version

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"bump-tui/internal/config"
	"github.com/Masterminds/semver/v3"
	"github.com/pelletier/go-toml/v2"
)

type ProjectType string

const (
	Rust       ProjectType = "rust"
	Python     ProjectType = "python"
	Cpp        ProjectType = "cpp"
	PlatformIO ProjectType = "platformio"
	Go         ProjectType = "go"
)

type ProjectFile struct {
	Path        string      `json:"path"`
	Type        ProjectType `json:"type"`
	Description string      `json:"description"`
}

type Manager struct {
	CurrentVersion *semver.Version    `json:"current_version"`
	ProjectFiles   []ProjectFile      `json:"project_files"`
	BumpConfig     *config.BumpConfig `json:"bump_config,omitempty"`
}

func NewManager() *Manager {
	return &Manager{
		CurrentVersion: semver.MustParse("0.1.0"), // Default version
		ProjectFiles:   []ProjectFile{},
	}
}

func (m *Manager) DetectVersionFiles(projectRoot string) error {
	// First, try to load .bump configuration
	bumpConfig, err := config.LoadBumpConfig(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load .bump config: %v", err)
	}

	if bumpConfig != nil {
		m.BumpConfig = bumpConfig
		return m.detectVersionFilesFromConfig(projectRoot)
	}

	// Fall back to automatic detection
	return m.detectVersionFilesAutomatically(projectRoot)
}

func (m *Manager) detectVersionFilesFromConfig(projectRoot string) error {
	var versions []*semver.Version

	for _, configFile := range m.BumpConfig.Files {
		fullPath := filepath.Join(projectRoot, configFile.Path)

		// Auto-detect project type based on file name/extension
		projectType := m.detectProjectTypeFromPath(configFile.Path)
		if projectType == "" {
			return fmt.Errorf("unable to determine project type for file: %s", configFile.Path)
		}

		projectFile := ProjectFile{
			Path:        fullPath,
			Type:        projectType,
			Description: m.getDefaultDescription(projectType),
		}

		// Extract version from this file
		version, err := m.extractVersionFromFile(fullPath, projectType)
		if err != nil {
			return fmt.Errorf("failed to extract version from %s: %v", configFile.Path, err)
		}

		if version != nil {
			versions = append(versions, version)
			// Use the first valid version as current version
			if m.CurrentVersion == nil || m.CurrentVersion.String() == "0.1.0" {
				m.CurrentVersion = version
			}
		}

		m.ProjectFiles = append(m.ProjectFiles, projectFile)
	}

	// Always check version sync when using .bump config
	if err := m.checkVersionSync(versions); err != nil {
		return err
	}

	return nil
}

func (m *Manager) detectVersionFilesAutomatically(projectRoot string) error {
	files := []struct {
		path        string
		projectType ProjectType
		description string
	}{
		{"go.mod", Go, "Go module file"},
		{"Cargo.toml", Rust, "Rust package manifest"},
		{"pyproject.toml", Python, "Python project configuration"},
		{"CMakeLists.txt", Cpp, "CMake build configuration"},
		{"platformio.ini", PlatformIO, "PlatformIO project configuration"},
		{"library.json", PlatformIO, "PlatformIO library manifest"},
		{"library.properties", PlatformIO, "Arduino library properties"},
	}

	for _, file := range files {
		fullPath := filepath.Join(projectRoot, file.path)
		if _, err := os.Stat(fullPath); err == nil {
			projectFile := ProjectFile{
				Path:        fullPath,
				Type:        file.projectType,
				Description: file.description,
			}

			// Try to extract version from this file
			if version, err := m.extractVersionFromFile(fullPath, file.projectType); err == nil && version != nil {
				m.CurrentVersion = version
			}

			m.ProjectFiles = append(m.ProjectFiles, projectFile)
		}
	}

	return nil
}

// detectProjectTypeFromPath determines the project type based on file path
func (m *Manager) detectProjectTypeFromPath(filePath string) ProjectType {
	fileName := filepath.Base(filePath)

	switch fileName {
	case "go.mod":
		return Go
	case "Cargo.toml":
		return Rust
	case "pyproject.toml":
		return Python
	case "CMakeLists.txt":
		return Cpp
	case "platformio.ini", "library.json", "library.properties":
		return PlatformIO
	default:
		return "" // Unknown type
	}
}

// getDefaultDescription returns a default description for a project type
func (m *Manager) getDefaultDescription(projectType ProjectType) string {
	switch projectType {
	case Go:
		return "Go module file"
	case Rust:
		return "Rust package manifest"
	case Python:
		return "Python project configuration"
	case Cpp:
		return "CMake build configuration"
	case PlatformIO:
		return "PlatformIO project configuration"
	default:
		return "Project configuration file"
	}
}

// checkVersionSync verifies that all versions are the same
func (m *Manager) checkVersionSync(versions []*semver.Version) error {
	if len(versions) <= 1 {
		return nil // Nothing to sync or only one version
	}

	firstVersion := versions[0]
	for i, version := range versions[1:] {
		if !version.Equal(firstVersion) {
			return fmt.Errorf("version mismatch: file %d has version %s, but file 0 has version %s",
				i+1, version.String(), firstVersion.String())
		}
	}

	return nil
}

// CheckAllVersionsInSync checks if all configured files have the same version
func (m *Manager) CheckAllVersionsInSync() error {
	if m.BumpConfig == nil {
		return nil // No config, nothing to check
	}

	var versions []*semver.Version
	var filePaths []string

	for _, projectFile := range m.ProjectFiles {
		version, err := m.extractVersionFromFile(projectFile.Path, projectFile.Type)
		if err != nil {
			return fmt.Errorf("failed to extract version from %s: %v", projectFile.Path, err)
		}

		if version != nil {
			versions = append(versions, version)
			filePaths = append(filePaths, projectFile.Path)
		}
	}

	// Check if all versions are the same
	if len(versions) <= 1 {
		return nil
	}

	firstVersion := versions[0]
	for i, version := range versions[1:] {
		if !version.Equal(firstVersion) {
			return fmt.Errorf("version mismatch: %s has version %s, but %s has version %s",
				filePaths[i+1], version.String(), filePaths[0], firstVersion.String())
		}
	}

	return nil
}

func (m *Manager) extractVersionFromFile(filePath string, projectType ProjectType) (*semver.Version, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	contentStr := string(content)

	switch projectType {
	case Go:
		return m.extractGoVersion()
	case Rust:
		return m.extractCargoVersion(contentStr)
	case Python:
		return m.extractPyprojectVersion(contentStr)
	case Cpp:
		return m.extractCMakeVersion(contentStr)
	case PlatformIO:
		if strings.HasSuffix(filePath, ".ini") {
			return m.extractPlatformIOIniVersion(contentStr)
		} else if strings.HasSuffix(filePath, ".json") {
			return m.extractLibraryJsonVersion(contentStr)
		} else if strings.HasSuffix(filePath, ".properties") {
			return m.extractLibraryPropertiesVersion(contentStr)
		}
	}

	return nil, fmt.Errorf("unsupported project type: %s", projectType)
}

func (m *Manager) extractGoVersion() (*semver.Version, error) {
	// For Go projects, get version from latest git tag
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err != nil {
		// If no tags exist, default to v0.1.0
		return semver.NewVersion("0.1.0")
	}

	tagStr := strings.TrimSpace(string(output))
	// Remove 'v' prefix if present
	if strings.HasPrefix(tagStr, "v") {
		tagStr = tagStr[1:]
	}

	return semver.NewVersion(tagStr)
}

func (m *Manager) extractCargoVersion(content string) (*semver.Version, error) {
	var config struct {
		Package struct {
			Version string `toml:"version"`
		} `toml:"package"`
	}

	err := toml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, err
	}

	if config.Package.Version == "" {
		return nil, fmt.Errorf("no version found in Cargo.toml")
	}

	return semver.NewVersion(config.Package.Version)
}

func (m *Manager) extractPyprojectVersion(content string) (*semver.Version, error) {
	var config struct {
		Tool struct {
			Poetry struct {
				Version string `toml:"version"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}

	err := toml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, err
	}

	if config.Tool.Poetry.Version == "" {
		return nil, fmt.Errorf("no version found in pyproject.toml")
	}

	return semver.NewVersion(config.Tool.Poetry.Version)
}

func (m *Manager) extractCMakeVersion(content string) (*semver.Version, error) {
	// Try project() version first - support variables like ${PROJECT_NAME}
	projectRe := regexp.MustCompile(`project\s*\(\s*[^)]+\s+VERSION\s+(\d+)\.(\d+)\.(\d+)`)
	if matches := projectRe.FindStringSubmatch(content); len(matches) >= 4 {
		versionStr := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
		return semver.NewVersion(versionStr)
	}

	// Try set(PROJECT_VERSION) format
	setRe := regexp.MustCompile(`set\s*\(\s*(?:PROJECT|CMAKE_PROJECT)_VERSION\s+(\d+)\.(\d+)\.(\d+)`)
	if matches := setRe.FindStringSubmatch(content); len(matches) >= 4 {
		versionStr := fmt.Sprintf("%s.%s.%s", matches[1], matches[2], matches[3])
		return semver.NewVersion(versionStr)
	}

	return nil, fmt.Errorf("no version found in CMakeLists.txt")
}

func (m *Manager) extractPlatformIOIniVersion(content string) (*semver.Version, error) {
	re := regexp.MustCompile(`version\s*=\s*["']?([\d\.]+)["']?`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no version found in platformio.ini")
	}

	return semver.NewVersion(matches[1])
}

func (m *Manager) extractLibraryJsonVersion(content string) (*semver.Version, error) {
	var config struct {
		Version string `json:"version"`
	}

	err := json.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, err
	}

	if config.Version == "" {
		return nil, fmt.Errorf("no version found in library.json")
	}

	return semver.NewVersion(config.Version)
}

func (m *Manager) extractLibraryPropertiesVersion(content string) (*semver.Version, error) {
	re := regexp.MustCompile(`version\s*=\s*([\d\.]+)`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no version found in library.properties")
	}

	return semver.NewVersion(matches[1])
}

func (m *Manager) BumpMajor() *semver.Version {
	newVersion := m.CurrentVersion.IncMajor()
	return &newVersion
}

func (m *Manager) BumpMinor() *semver.Version {
	newVersion := m.CurrentVersion.IncMinor()
	return &newVersion
}

func (m *Manager) BumpPatch() *semver.Version {
	newVersion := m.CurrentVersion.IncPatch()
	return &newVersion
}

func (m *Manager) UpdateAllVersions(newVersion string) error {
	for _, projectFile := range m.ProjectFiles {
		if err := m.updateVersionInFile(projectFile, newVersion); err != nil {
			return fmt.Errorf("failed to update %s: %v", projectFile.Path, err)
		}
	}
	return nil
}

func (m *Manager) updateVersionInFile(projectFile ProjectFile, newVersion string) error {
	content, err := os.ReadFile(projectFile.Path)
	if err != nil {
		return err
	}

	var updatedContent string

	switch projectFile.Type {
	case Go:
		return m.updateGoVersion(newVersion)
	case Rust:
		updatedContent, err = m.updateCargoVersion(string(content), newVersion)
	case Python:
		updatedContent, err = m.updatePyprojectVersion(string(content), newVersion)
	case Cpp:
		updatedContent, err = m.updateCMakeVersion(string(content), newVersion)
	case PlatformIO:
		if strings.HasSuffix(projectFile.Path, ".ini") {
			updatedContent, err = m.updatePlatformIOIniVersion(string(content), newVersion)
		} else if strings.HasSuffix(projectFile.Path, ".json") {
			updatedContent, err = m.updateLibraryJsonVersion(string(content), newVersion)
		} else if strings.HasSuffix(projectFile.Path, ".properties") {
			updatedContent, err = m.updateLibraryPropertiesVersion(string(content), newVersion)
		}
	default:
		return fmt.Errorf("unsupported project type: %s", projectFile.Type)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(projectFile.Path, []byte(updatedContent), 0644)
}

func (m *Manager) updateGoVersion(newVersion string) error {
	// For Go projects, we create a git tag instead of modifying files
	// The go.mod file doesn't contain version information
	// The actual git tag creation is handled by the git manager
	return nil
}

func (m *Manager) updateCargoVersion(content, newVersion string) (string, error) {
	re := regexp.MustCompile(`(\[package\][\s\S]*?version\s*=\s*")([^"]+)(")`)
	return re.ReplaceAllString(content, "${1}"+newVersion+"${3}"), nil
}

func (m *Manager) updatePyprojectVersion(content, newVersion string) (string, error) {
	re := regexp.MustCompile(`(\[tool\.poetry\][\s\S]*?version\s*=\s*")([^"]+)(")`)
	return re.ReplaceAllString(content, "${1}"+newVersion+"${3}"), nil
}

func (m *Manager) updateCMakeVersion(content, newVersion string) (string, error) {
	parts := strings.Split(newVersion, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", newVersion)
	}

	// Update project() version - support variables like ${PROJECT_NAME}
	projectRe := regexp.MustCompile(`(project\s*\(\s*[^)]+\s+VERSION\s+)(\d+\.\d+\.\d+)`)
	content = projectRe.ReplaceAllString(content, "${1}"+newVersion)

	// Update set(PROJECT_VERSION) format
	setRe := regexp.MustCompile(`(set\s*\(\s*(?:PROJECT|CMAKE_PROJECT)_VERSION\s+)(\d+\.\d+\.\d+)`)
	content = setRe.ReplaceAllString(content, "${1}"+newVersion)

	return content, nil
}

func (m *Manager) updatePlatformIOIniVersion(content, newVersion string) (string, error) {
	re := regexp.MustCompile(`(version\s*=\s*)([^\r\n]+)`)
	return re.ReplaceAllString(content, "${1}\""+newVersion+"\""), nil
}

func (m *Manager) updateLibraryJsonVersion(content, newVersion string) (string, error) {
	var config map[string]interface{}
	err := json.Unmarshal([]byte(content), &config)
	if err != nil {
		return "", err
	}

	config["version"] = newVersion

	updatedBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	return string(updatedBytes), nil
}

func (m *Manager) updateLibraryPropertiesVersion(content, newVersion string) (string, error) {
	re := regexp.MustCompile(`(version\s*=\s*)([^\r\n]+)`)
	return re.ReplaceAllString(content, "${1}"+newVersion), nil
}
