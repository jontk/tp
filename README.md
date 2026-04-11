# ratatosk

A TUI project picker that creates tmux sessions with per-project windows, each with a configurable multi-pane layout.

Built for workflows where you have many projects but only work on a handful at a time. Instead of hardcoding which projects to open, `ratatosk` lets you interactively select them.

```
┌──────────────┬──────────────┐
│              │    shell     │
│    vim .     │  (selected)  │
│              ├──────────────┤
│              │   lazygit    │
└──────────────┴──────────────┘
```

## Install

### Pre-built binaries

Download from the [releases page](https://github.com/jontk/ratatosk/releases) for Linux and macOS (amd64/arm64).

### Go install

```bash
go install github.com/jontk/ratatosk/cmd/tosk@latest
```

### Build from source

```bash
git clone https://github.com/jontk/ratatosk.git
cd ratatosk
go build -o tosk ./cmd/tosk
```

## Usage

```bash
tosk              # Open picker — creates session or manages existing one
tosk list         # List current session windows
tosk kill         # Kill the current session (switches to another if available)
tosk switch       # Switch between profile sessions
tosk validate     # Validate config file
tosk config       # Open config in $EDITOR
tosk --cc         # Force iTerm2 control mode on attach
tosk -p work      # Use a named profile (work.yaml)
```

`ratatosk` is context-aware: if no session exists it creates one with your last selections pre-selected (falling back to defaults on first run). If a session already exists, it opens the picker showing what's open so you can add or remove windows. Selections are saved automatically per profile.

### Picker controls

| Key | Action |
|-----|--------|
| Type | Filter projects |
| Space / Tab | Toggle selection |
| Enter | Confirm and launch |
| Ctrl+A | Select / deselect all visible |
| Ctrl+S | Toggle sort (recent / alphabetical) |
| Esc | Cancel |

Projects already open in the session are shown with an `(open)` badge and pre-selected. Deselecting an open project will close its window — it turns red with a `(closing)` indicator before you confirm.

A preview panel on the right shows git info (branch, last commit, status, remote) for the highlighted project. Info is fetched asynchronously so the UI stays responsive.

### SSH usage

When connecting from a remote machine (e.g. macOS with iTerm2):

```bash
ssh user@host -t '~/go/bin/tosk --cc'
```

The `--cc` flag enables iTerm2 control mode (`tmux -CC`), which gives you native tabs and windows for each tmux window.

## Configuration

Config lives at `~/.config/ratatosk/config.yaml` and is created with defaults on first run.

See [config.example.yaml](config.example.yaml) for a fully commented example.

### source_dirs

Paths or glob patterns that resolve to project directories. Tilde (`~`) is expanded to your home directory:

```yaml
source_dirs:
    - ~/src/github.com/myuser/*      # every subdir is a project
    - ~/src/github.com/myuser/api-*  # only projects matching the prefix
    - ~/src/github.com/myuser/myapp  # a single project (literal path)
```

### defaults

Projects pre-selected in the picker. You can still deselect them:

```yaml
defaults:
    - project-a
    - project-b
```

### session

Tmux session name (default: `projects`). Using a single session works best with iTerm2's tmux integration.

### sort

Default sort order for the picker: `recent` (default) or `alphabetical`. Recent sorts by last git commit, showing your most active projects first. Toggle with Ctrl+S in the picker.

```yaml
sort: recent
```

### preview

Show a git info preview panel next to the project list (default: `true`). Auto-hides if the terminal is narrower than 80 columns. Set to `false` to disable:

```yaml
preview: false
```

### layout

Layout is defined as sequential tmux splits. The first entry is the initial pane. Each subsequent entry splits the last created pane:

```yaml
layout:
    - command: vim .
    - split: horizontal
      percent: 40
      command: ""
      active: true
    - split: vertical
      percent: 50
      command: lazygit
```

- `split`: `horizontal` (side-by-side) or `vertical` (top/bottom)
- `percent`: size of the new pane
- `command`: command to run (empty string = shell)
- `active`: which pane gets focus

This maps directly to how tmux splits work, so any layout is possible — multiple columns, nested splits, etc. See [config.example.yaml](config.example.yaml) for more examples.

`ratatosk` is layout- and tool-agnostic: any command works in a pane. Editors, shells, watchers (`entr`, `watchexec`), log tailers (`kubectl logs`, `docker compose logs`), TUIs (`lazygit`, `k9s`, `htop`), AI agents (`claude`, `aider`, `codex`, `gemini`) — all fair game.

### Layout presets

Define reusable layouts and assign them to projects by name:

```yaml
layout_presets:
    editor:
        - command: vim .
        - split: horizontal
          percent: 40
          command: ""
          active: true
        - split: vertical
          percent: 50
          command: lazygit

    frontend:
        - command: npm run dev
        - split: horizontal
          percent: 60
          command: vim .
        - split: vertical
          percent: 66
          command: npm test -- --watch
        - split: vertical
          percent: 50
          command: ""
          active: true

projects:
    dotfiles:
        preset: editor
    my-webapp:
        preset: frontend
```

Projects can use `preset:` to reference a named layout, or inline `layout:` for one-offs. Inline layout takes priority if both are set.

### Per-project overrides

Override the layout for specific projects using the `projects` key:

**Editor-centric** — just an editor, a shell, and lazygit:

```yaml
# ┌──────────┬──────────┐
# │          │  shell   │
# │   vim    ├──────────┤
# │          │ lazygit  │
# └──────────┴──────────┘
projects:
    dotfiles:
        layout:
            - command: vim .
            - split: horizontal
              percent: 40
              command: ""
              active: true
            - split: vertical
              percent: 50
              command: lazygit
```

**Go backend** — a test watcher alongside the editor:

```yaml
# ┌──────────┬──────────┐
# │          │   vim    │
# │ go test  ├──────────┤
# │ (watch)  │  shell   │
# └──────────┴──────────┘
projects:
    my-api:
        layout:
            - command: watchexec -e go -- go test ./...
            - split: horizontal
              percent: 60
              command: vim .
            - split: vertical
              percent: 50
              command: ""
              active: true
```

**Ops / infra** — a cluster TUI with editor and shell:

```yaml
# ┌─────────────────────┐
# │         k9s         │
# ├──────────┬──────────┤
# │   vim    │  shell   │
# └──────────┴──────────┘
projects:
    infra:
        layout:
            - command: k9s
            - split: vertical
              percent: 60
              command: vim .
            - split: horizontal
              percent: 50
              command: ""
              active: true
```

**Wide-screen 3-column** — for large monitors:

```yaml
# ┌──────────┬──────────┬──────────┐
# │          │          │  shell   │
# │ lazygit  │   vim    ├──────────┤
# │          │          │  shell   │
# └──────────┴──────────┴──────────┘
projects:
    big-project:
        layout:
            - command: lazygit
            - split: horizontal
              percent: 66
              command: vim .
            - split: horizontal
              percent: 50
              command: ""
            - split: vertical
              percent: 50
              command: ""
              active: true
```

## Profiles

Profiles let you maintain separate configurations for different contexts (e.g. personal, work, a specific project). Each profile is a separate YAML file in the config directory:

```
~/.config/ratatosk/config.yaml     # default (tosk)
~/.config/ratatosk/work.yaml       # tosk -p work
~/.config/ratatosk/ops.yaml        # tosk -p ops
```

Each profile has its own session name, source directories, defaults, and layout. Use `-p` with any command:

```bash
tosk -p work              # Launch or manage work profile
tosk -p work config       # Edit work config
```

### Context-aware auto-detection

When you run `tosk` inside a tmux session that was created by a profile, it automatically detects the profile — no need to pass `-p`. This means:

- The tmux keybinding (`prefix + A`) works correctly regardless of which profile's session you're in
- Running `tosk list` from a shell pane uses the right config automatically

This works by storing the profile name as a tmux session environment variable (`TOSK_PROFILE`) when the session is created.

## Tmux keybinding

Add a keybinding to open the picker from within an active tmux session. Add this to your `~/.tmux.conf`:

```tmux
# prefix + A opens the tosk project picker in a popup
bind A display-popup -E -w 80% -h 80% "~/go/bin/tosk"
```

This opens `tosk` in a centered popup overlay. The `-E` flag closes the popup automatically when you're done selecting.

Use the full path to the binary since tmux popups don't load your shell profile. The keybinding is automatically context-aware — it detects which profile the current session belongs to.

## How it works

1. Scans configured `source_dirs` for project directories
2. Presents a TUI picker with filter and multi-select
3. For each selected project, creates a tmux window with the configured pane layout
4. Closes windows for any deselected open projects
5. Attaches to the session (with optional iTerm2 `-CC` mode)

All projects live as windows within a single tmux session per profile, making it easy to switch between them and compatible with iTerm2's tmux integration.
