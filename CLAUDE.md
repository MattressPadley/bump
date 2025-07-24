# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Bump is an interactive Terminal User Interface (TUI) tool for semantic version management built in Go. It automates version bumping, changelog generation from conventional commits, git tag creation, and GitHub release management.

## Architecture

The application follows a modular architecture with these key components:

- **main.go**: Entry point that initializes the Bubble Tea TUI
- **internal/models/main.go**: Core TUI model implementing the Bubble Tea pattern with session states (welcome → version selection → changelog generation → preview → confirmation → progress → results)
- **internal/version/manager.go**: Handles version detection, parsing, and updating across multiple project types (Go, Rust, Python, C++, PlatformIO)
- **internal/changelog/manager.go**: Generates changelogs from conventional commits with Claude AI integration and regex fallback
- **internal/git/manager.go**: Git operations (commits, tags, pushing)
- **internal/config/bump_config.go**: Configuration file parsing for `.bump` TOML files

### Key Architecture Patterns

- **Manager Pattern**: Each domain (version, git, changelog) has its own manager with encapsulated functionality
- **State Machine**: TUI uses sessionState enum to manage user flow through different screens
- **Command Pattern**: Bubble Tea commands handle async operations (changelog generation, version updates)
- **Configuration Override**: `.bump` files take precedence over automatic project file detection

## Development Commands

### Building and Running
```bash
just build          # Build binary to build/bump-tui
just run             # Run with build info (version/commit/date)
just dev             # Run with debug logging (DEBUG=1)
go run .             # Basic run without build info
```

### Testing and Maintenance
```bash
just test            # Run all tests
just tidy            # Tidy go modules
just clean           # Remove build artifacts
just install         # Install to GOPATH/bin
```

### Multi-platform Building
```bash
just build-all       # Build for Linux, macOS (Intel/ARM), Windows
```

### Debug Mode
```bash
DEBUG=1 ./build/bump-tui    # Enables debug logging to debug.log
```

## Supported Project Types

The version manager detects and updates these project files:
- **Go**: Uses git tags for versioning (no file modification)
- **Rust**: `Cargo.toml` 
- **Python**: `pyproject.toml` (Poetry projects)
- **C++**: `CMakeLists.txt`
- **PlatformIO**: `platformio.ini`, `library.json`, `library.properties`

## Configuration System

### .bump Configuration Files
Projects can define a `.bump` TOML file to specify multiple version files:

```toml
[[files]]
path = "Cargo.toml"
description = "Rust package manifest"

[[files]]  
path = "pyproject.toml"
description = "Python project configuration"
```

When `.bump` exists, it overrides automatic detection and all specified files are updated synchronously.

## Changelog Generation

The changelog system has two modes:
1. **Claude AI**: Uses Claude Code CLI for intelligent commit analysis
2. **Regex Fallback**: Pattern-based conventional commit parsing

Conventional commit types are mapped to emojis (feat→✨, fix→🐛, docs→📚, etc.).

## TUI Flow Implementation

The application implements a linear state machine:
1. **welcomeView**: Project detection and initialization
2. **versionSelectView**: Major/minor/patch selection with current version display
3. **changelogGeneratingView**: Async changelog generation with spinner
4. **changelogPreviewView**: Scrollable changelog review
5. **confirmationView**: Final confirmation with action summary
6. **progressView**: Version update operations with spinner
7. **resultsView**: Success summary and next steps

## Testing Notes

- Run tests with `just test` or `go test -v ./...`
- No specific test framework configuration required - uses standard Go testing
- Tests are located alongside source files following Go conventions

## Git Workflow Integration

The tool integrates with GitHub workflows:
- Creates conventional commit messages (`chore(release): bump version to X.Y.Z`)
- Creates annotated git tags (`vX.Y.Z`)
- Pushes changes and tags separately to trigger CI/CD
- Mentions GitHub Actions will build binaries and update Homebrew tap