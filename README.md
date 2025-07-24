# Bump TUI

An interactive Terminal User Interface (TUI) for managing project versions, git tags, and changelogs using conventional commits.

## Features

- 🖥️ **Interactive TUI** - Beautiful terminal interface with real-time feedback
- 🔢 **Semantic version bumping** - Major, minor, and patch version increments
- 📦 **Multi-project support** - Rust, Python, C++, and PlatformIO projects
- 📝 **Automatic changelog generation** - From conventional commits with emoji categorization
- 🏷️ **Git tag creation** - Automatic tagging with proper commit messages
- 🚀 **GitHub release creation** - Integrated with `gh` CLI
- 🔍 **Git repository validation** - Comprehensive checks for clean working directory, submodules, and branch status
- 📋 **Submodule tag validation** - Ensures submodules point to release tags before version bumping
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
2. **Repository Validation** - Comprehensive git status and submodule checks
3. **Version Selection** - Choose major, minor, or patch bump
4. **Changelog Preview** - Review generated changes from commits
5. **Confirmation** - Final review before applying changes
6. **Progress** - Real-time feedback during operations
7. **Results** - Success summary

## Git Repository Validation

Before allowing version bumps, the tool performs comprehensive repository validation:

### Validation Checks

**✅ Repository Status**
- Verifies you're in a git repository
- Checks git connectivity and remote configuration

**✅ Working Directory**
- **Blocks on**: Uncommitted changes
- **Warns on**: Untracked files

**✅ Branch Status**  
- **Warns on**: Detached HEAD state
- **Warns on**: Branch ahead/behind remote

**✅ Submodule Validation**
- Detects and validates git submodules
- **Blocks on**: Submodules with uncommitted changes
- **Warns on**: Submodules not pointing to release tags
- **Success**: Submodules pointing to specific version tags

### Validation Results

The validation screen shows detailed results and requires user confirmation before proceeding. You can continue with warnings but errors must be resolved first.

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

### Building and Testing

```bash
just build          # Build binary
just run            # Run with build info
just dev            # Run with debug logging
just test           # Run tests
just test-coverage  # Run tests with coverage and race detection
just lint           # Run golangci-lint
just vet            # Run go vet
just ci-test        # Run full CI-equivalent checks
just clean          # Clean build artifacts
just build-all      # Build for multiple platforms
```

### Testing

The project includes comprehensive test coverage:

- **89 test cases** covering all validation scenarios
- **Security tests** for path traversal protection
- **Integration tests** with real git repositories
- **Race condition detection** in concurrent operations
- **Coverage reporting** with detailed metrics

Run the full test suite with:

```bash
just ci-test        # Complete CI checks (tidy, vet, test-coverage, lint)
just test-coverage  # Tests with race detection and coverage
```

### Code Quality

The codebase maintains high quality standards:

- **golangci-lint** with comprehensive linters (errcheck, staticcheck, unused, unparam)
- **Context-based timeouts** for all external git commands (30s default)
- **Security validations** preventing path traversal attacks
- **Comprehensive error handling** with user-friendly messages
- **Extracted constants** and clean separation of concerns

### CI/CD Pipeline

The project includes a robust GitHub Actions workflow:

- **Automated testing** on pull requests and pushes to main
- **Consistent tooling** between local development and CI using just commands
- **Multi-job pipeline** with separate test and build validation
- **Dependency caching** for faster CI runs
- **Code coverage reporting** with Codecov integration

The CI pipeline runs:
1. **Test job**: `just ci-test` (tidy, vet, test-coverage, lint)
2. **Build job**: `just build` with binary verification

## Requirements

- Git repository (must be run from within a git repo)
- Clean working directory (no uncommitted changes)
- At least one supported project file
- GitHub CLI (`gh`) configured for release creation
- For repositories with submodules: submodules should point to release tags for best practices