# Version Manager

A command-line tool for managing project versions, git tags, and changelogs using conventional commits.

## Features

- 🔢 Semantic version bumping (major, minor, patch)
- 📦 Automatic version updates in Cargo.toml, pyproject.toml, CMakeLists.txt, and meson.build
- 📝 Automatic changelog generation from conventional commits
- 🏷️ Git tag creation and management
- 🚀 GitHub release creation
- 🔄 Automated git operations

## Installation

Install using Homebrew:
```bash
brew tap mattresspadley/tap
brew install bump
```

Or, build from source:
Clone this repository
```bash
git clone https://github.com/mattresspadley/bump.git
```
Install the binary:
```bash
cargo install --path .
```


## How It Works

1. **Version Detection**: Automatically detects and reads version information from:
   - Cargo.toml (Rust projects)
   - pyproject.toml (Python projects)
   - CMakeLists.txt (C++ projects)
   - meson.build (Meson projects)

2. **Version Bumping**: Updates version numbers according to semantic versioning rules:
   - Major: Breaking changes (x.0.0)
   - Minor: New features (0.x.0)
   - Patch: Bug fixes (0.0.x)

3. **Changelog Generation**: 
   - Automatically generates changelog entries from git commits
   - Uses conventional commit format for smart categorization
   - Updates CHANGELOG.md in the docs directory
   - Adds emojis for better readability

4. **Git Integration**:
   - Creates version bump commits
   - Creates git tags
   - Optionally pushes changes to remote
   - Creates GitHub releases (with --release flag)

## Conventional Commits

The changelog generator recognizes the following conventional commit types:

- ✨ feat: New features
- 🐛 fix: Bug fixes
- 📚 docs: Documentation changes
- 💎 style: Code style changes
- ♻️ refactor: Code refactoring
- ⚡️ perf: Performance improvements
- ✅ test: Test updates
- 📦 build: Build system changes
- 👷 ci: CI configuration changes
- 🔧 chore: General maintenance

## Requirements

- Git
- GitHub CLI (gh) for release creation

