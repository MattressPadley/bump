# Changelog

# 0.1.5 (2025-07-24)

## Bug Fixes
- Fixed error handling to resolve linting issues
- Updated golangci-lint configuration for v2.1 compatibility  
- Updated golangci-lint GitHub Action to latest version with recommended settings
- Pinned golangci-lint to v1.61.0 for config compatibility
- Added golangci-lint installation to CI workflow
- Resolved various golangci-lint issues

## Features
- Added comprehensive git repository validation with submodule tag checking
- Added PR review recommendations and comprehensive improvements

## Improvements
- Improved git validation performance, error handling, and security
- Updated README with comprehensive improvements and CI/CD details

## Other
- Added Claude Code Review workflow
- Added Claude PR Assistant workflow


# 0.1.4 (2025-07-24)

## Features
- Simplified .bump configuration format to use gitignore-style syntax

## Improvements
- Updated README documentation to reflect new .bump configuration format

## Other
- Added bump-tui binary to .gitignore
- Removed artifact files


# 0.1.3 (2025-07-22)

## Features
- Add support for .bump configuration files to manage versions across multiple files


# 0.1.2 (2025-07-22)

## Features
- Renamed command from `bump-tui` to `bump` for easier usage

## Bug Fixes
- Fixed spinner animation during command execution


# 0.1.1 (2025-07-22)

## Bug Fixes
- Improved GitHub Actions workflow reliability by modifying push strategy


# 0.1.0 (2025-07-22)

## Features
- Added animated progress indicator for changelog generation
- Applied Catppuccin Macchiato theme with blue accents
- Rewrote application in Go with interactive terminal interface
- Added support for Go projects

## Bug Fixes
- Fixed changelog viewport sizing issue that left gaps at bottom
- Fixed version display to show current version during preview
- Improved error handling for git operations during changelog generation


# 0.0.10 (2025-03-23)


