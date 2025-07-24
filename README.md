# Bump TUI

An interactive Terminal User Interface (TUI) for managing project versions, git tags, and changelogs using conventional commits.

## Features

- 🖥️ **Interactive TUI** - Beautiful terminal interface with real-time feedback
- 🔢 **Semantic version bumping** - Major, minor, and patch version increments
- 📦 **Multi-project support** - Rust, Python, C++, and PlatformIO projects
- 📝 **Automatic changelog generation** - From conventional commits with emoji categorization
- 🏷️ **Git tag creation** - Automatic tagging with proper commit messages
- 🚀 **GitHub release creation** - Integrated with `gh` CLI
- ⚡ **Fast and responsive** - Built with Go for performance

## Installation

### Prerequisites

- Go 1.21 or higher
- Git
- GitHub CLI (`gh`) for release creation
- [Just](https://github.com/casey/just) for building (optional)

### Build from source

```bash
git clone https://github.com/yourusername/bump.git
cd bump
just build
# or
go build -o bump-tui .
```

## Usage

Simply run the interactive TUI:

```bash
./build/bump-tui
```

### Command-line options

```bash
./build/bump-tui -help     # Show help
./build/bump-tui -version  # Show version info
```

### Environment variables

```bash
DEBUG=1 ./build/bump-tui   # Enable debug logging
```

## Supported Project Types

- **Go** - `go.mod` (uses git tags for versioning)
- **Rust** - `Cargo.toml`
- **Python** - `pyproject.toml` (Poetry)
- **C++** - `CMakeLists.txt`
- **PlatformIO** - `platformio.ini`, `library.json`, `library.properties`

## .bump Configuration File

You can create a `.bump` file in your repository root to specify multiple version files to manage and keep in sync. The format is simple and familiar, like a `.gitignore` file.

### Example Configuration

```
# Version files to manage
Cargo.toml
pyproject.toml
platformio.ini

# Comments and empty lines are ignored
CMakeLists.txt
```

### Format

- One file path per line (relative to repository root)
- Lines starting with `#` are treated as comments
- Empty lines are ignored
- File types are automatically detected based on filename

### Behavior

- When a `.bump` file exists, it takes precedence over automatic detection
- All configured files are updated when bumping versions
- All configured files must have matching versions (automatically enforced)

## TUI Flow

1. **Welcome Screen** - Project detection and initialization
2. **Version Selection** - Choose major, minor, or patch bump
3. **Changelog Preview** - Review generated changes from commits
4. **Confirmation** - Final review before applying changes
5. **Progress** - Real-time feedback during operations
6. **Results** - Success summary

## Keyboard Navigation

- `↑/↓` or `j/k` - Navigate lists
- `←/→` or `h/l` - Navigate between screens
- `Enter` - Select/confirm
- `q` or `Ctrl+C` - Quit

## Conventional Commits

The changelog generator recognizes these commit types:

- ✨ `feat:` - New features
- 🐛 `fix:` - Bug fixes
- 📚 `docs:` - Documentation changes
- 💎 `style:` - Code style changes
- ♻️ `refactor:` - Code refactoring
- ⚡️ `perf:` - Performance improvements
- ✅ `test:` - Test updates
- 📦 `build:` - Build system changes
- 👷 `ci:` - CI configuration changes
- 🔧 `chore:` - General maintenance

## Development

### Building

```bash
just build        # Build binary
just run          # Run with build info
just dev          # Run with debug logging
just test         # Run tests
just clean        # Clean build artifacts
just build-all    # Build for multiple platforms
```

## Requirements

- Git repository (must be run from within a git repo)
- At least one supported project file
- GitHub CLI (`gh`) configured for release creation