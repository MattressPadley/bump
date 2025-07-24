package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BumpConfig represents the configuration from a .bump file
type BumpConfig struct {
	// Version files to manage
	Files []VersionFile
}

// VersionFile represents a single version file configuration
type VersionFile struct {
	// Path to the file relative to the repository root
	Path string
}

// LoadBumpConfig loads the .bump configuration file from the project root
func LoadBumpConfig(projectRoot string) (*BumpConfig, error) {
	configPath := filepath.Join(projectRoot, ".bump")
	
	// Check if .bump file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // No config file, return nil (not an error)
	}
	
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open .bump config: %v", err)
	}
	defer file.Close()
	
	var config BumpConfig
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		config.Files = append(config.Files, VersionFile{Path: line})
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse .bump config: %v", err)
	}
	
	// Validate configuration
	if err := config.Validate(projectRoot); err != nil {
		return nil, fmt.Errorf("invalid .bump config: %v", err)
	}
	
	return &config, nil
}

// Validate checks if the configuration is valid
func (c *BumpConfig) Validate(projectRoot string) error {
	if len(c.Files) == 0 {
		return fmt.Errorf("no files specified in configuration")
	}
	
	seenPaths := make(map[string]bool)
	for i, file := range c.Files {
		if file.Path == "" {
			return fmt.Errorf("file %d: path cannot be empty", i)
		}
		
		// Check for duplicate paths
		if seenPaths[file.Path] {
			return fmt.Errorf("duplicate file path: %s", file.Path)
		}
		seenPaths[file.Path] = true
		
		// Validate file exists
		fullPath := filepath.Join(projectRoot, file.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", file.Path)
		}
	}
	
	return nil
}

// GetAbsolutePaths returns the absolute paths of all configured files
func (c *BumpConfig) GetAbsolutePaths(projectRoot string) []string {
	paths := make([]string, len(c.Files))
	for i, file := range c.Files {
		paths[i] = filepath.Join(projectRoot, file.Path)
	}
	return paths
}