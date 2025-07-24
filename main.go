package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bump-tui/internal/models"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("bump-tui %s (commit: %s, date: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *showHelp {
		fmt.Println("Bump - Interactive Version Management Tool")
		fmt.Println("")
		fmt.Println("A TUI tool for semantic versioning, changelog generation, and git operations.")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  bump-tui [flags]")
		fmt.Println("")
		fmt.Println("Flags:")
		fmt.Println("  -version    Show version information")
		fmt.Println("  -help       Show this help message")
		fmt.Println("")
		fmt.Println("Supported project types:")
		fmt.Println("  • Rust (Cargo.toml)")
		fmt.Println("  • Python (pyproject.toml)")
		fmt.Println("  • C++ (CMakeLists.txt)")
		fmt.Println("  • PlatformIO (platformio.ini, library.json, library.properties)")
		fmt.Println("")
		fmt.Println("Requirements:")
		fmt.Println("  • Git repository")
		fmt.Println("  • gh CLI (for GitHub releases)")
		os.Exit(0)
	}

	// Enable debug logging if DEBUG env var is set
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Printf("Warning: failed to close debug log: %v\n", err)
			}
		}()
	}

	// Start the TUI
	p := tea.NewProgram(
		models.NewMainModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
