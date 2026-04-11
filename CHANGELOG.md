# Changelog

All notable changes to ratatosk are documented here.

## [0.1.0] - 2026-04-11

First release as **ratatosk** (binary: `tosk`), renamed from `tp`.

### Added
- TUI project picker with filter, multi-select, and keyboard shortcuts
- Git preview panel showing branch, last commit, status, and remote
- Live ASCII layout diagram in the preview panel
- Configurable multi-pane tmux layouts (sequential splits)
- Per-project layout overrides
- Reusable layout presets (`layout_presets`)
- Glob patterns and tilde expansion in `source_dirs`
- Sort by recent git activity or alphabetical (toggle with Ctrl+S)
- Profile support (`tosk -p work`) with separate configs per context
- Context-aware profile auto-detection inside tmux sessions
- Save and restore project selections per profile
- `tosk list` — list current session windows
- `tosk kill` — kill session (switches to another if available)
- `tosk switch` — switch between profile sessions
- `tosk validate` — validate config file
- `tosk config` — open config in `$EDITOR`
- iTerm2 tmux integration (`--cc` flag)
- Duplicate project name disambiguation across source dirs
- Editor-centric default layout (vim + shell + lazygit)
- Goreleaser config for cross-platform builds (Linux/macOS, amd64/arm64)

[0.1.0]: https://github.com/jontk/ratatosk/releases/tag/v0.1.0
