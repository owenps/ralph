# Ralph TUI - Task Management Feature Specification

## Overview

Ralph is a TUI application for managing backlog tasks associated with a codebase. Tasks are stored in YAML format and managed through an interactive terminal interface built with Bubble Tea following the ELM architecture.

---

## Data Model

### Task Structure

```yaml
- id: "20250105143022"          # Timestamp-based ID (YYYYMMDDHHmmss)
  category: feature             # One of: bug, feature, refactor
  description: "High level summary of the feature"
  steps:                        # List of steps to complete/validate
    - "Step one"
    - "Step two"
  done: false                   # Boolean completion status
```

### File Format

- **Single file**: All tasks stored in one YAML file as a list
- **No file metadata**: Just the task list, no version headers or file-level timestamps
- **Task IDs**: Timestamp-based format (`YYYYMMDDHHmmss`) - unique, sortable, human-readable

### Categories

Predefined, constrained set:
- `bug` - Bug fixes
- `feature` - New features
- `refactor` - Code refactoring

---

## Configuration

### Config File

- **Location**: `.ralph.yaml` in project root
- **Format**:
  ```yaml
  tasks_file: .ralph/tasks.yaml
  ```
- **Scope**: Path setting only (expandable later)

### Behavior Without Config

- On launch without `.ralph.yaml`: **Launch Setup Wizard** (see Feature: First-Time Setup)
- Setup wizard guides user through configuration
- **Default tasks path**: `.ralph/tasks.yaml`

### Missing Tasks File

- If tasks file doesn't exist: Prompt user "No tasks file found. Create one?"
- If user accepts: Create directory structure and empty tasks file

---

## Feature: First-Time Setup

### Trigger

Setup screen appears automatically when **no `.ralph.yaml` exists** in the current directory. This is the only trigger - no separate state tracking needed.

### Re-running Setup

- **Command**: `ralph init` subcommand
- Running `ralph` without arguments launches the TUI normally
- Running `ralph init` forces the setup flow regardless of existing config
- **If config exists**: Show current values and prompt "Overwrite existing config?" before proceeding

### Setup Flow

**Wizard with breadcrumb progress indicator**: `Welcome > Config > Done`

#### Step 1: Welcome

```
┌─────────────────────────────────────────────────────┐
│              [Ralph ASCII Banner]                   │
├─────────────────────────────────────────────────────┤
│  Welcome > Config > Done                            │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Let's set up Ralph for this project.               │
│                                                     │
│  Press enter to continue.                           │
│                                                     │
├─────────────────────────────────────────────────────┤
│ enter:continue  q:quit                              │
└─────────────────────────────────────────────────────┘
```

- Brief welcome message with Ralph banner
- Single line explanation: "Let's set up Ralph for this project."
- Press enter to proceed

#### Step 2: Config (Tasks Path)

```
┌─────────────────────────────────────────────────────┐
│              [Ralph ASCII Banner]                   │
├─────────────────────────────────────────────────────┤
│  Welcome > Config > Done                            │
├─────────────────────────────────────────────────────┤
│                                                     │
│  Where should Ralph store tasks?                    │
│                                                     │
│  Path: .ralph/tasks.yaml                            │
│        ^^^^^^^^^^^^^^^^^^                           │
│                                                     │
│  Press enter to accept or type a custom path.       │
│                                                     │
├─────────────────────────────────────────────────────┤
│ enter:confirm  esc:back  q:quit                     │
└─────────────────────────────────────────────────────┘
```

- **Input behavior**: Text field pre-populated with default `.ralph/tasks.yaml`
- User can press enter to accept default, or clear and type custom path
- **No validation**: Accept any path - directories created as needed later
- Esc returns to Welcome step

#### Step 3: Done

```
┌─────────────────────────────────────────────────────┐
│              [Ralph ASCII Banner]                   │
├─────────────────────────────────────────────────────┤
│  Welcome > Config > Done                            │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ✓ Created .ralph.yaml                              │
│  ✓ Ready to use                                     │
│                                                     │
│  Press enter to continue to Ralph.                  │
│                                                     │
├─────────────────────────────────────────────────────┤
│ enter:continue                                      │
└─────────────────────────────────────────────────────┘
```

- Show confirmation of what was created
- Press enter to proceed to main menu

### Post-Setup Behavior

**Navigate to main menu** - User lands at normal main menu after setup completes.

### Cancel Behavior

**Confirm then exit** - If user presses `q` or `Ctrl+C` during setup:
1. Show prompt: "Exit without completing setup?"
2. Yes: Exit application entirely
3. No: Return to current setup step

Cannot use Ralph without completing setup (no "skip" option).

### Data Collected

Only **tasks file path** - minimal setup. Additional settings may be added to setup flow in future versions.

---

## Architecture

### Menu System

Implement **Menu Item Interface** pattern for extensibility:

```go
type MenuItem interface {
    Name() string
    Description() string
    Init() tea.Cmd
    Update(msg tea.Msg) (tea.Model, tea.Cmd)
    View() string
}
```

Each feature (Create Task, View Tasks, future features) implements this interface and registers with the main menu. This allows clean separation and easy addition of new menu items.

### Application States

1. **Setup Wizard** - First-time configuration flow (when no `.ralph.yaml`)
2. **Main Menu** - Entry point with menu item selection
3. **Create Task Wizard** - Multi-step task creation flow
4. **View Tasks** - Task list with side panel detail view

---

## User Interface

### Branding & Theme

- **Header**: Styled ASCII art banner with "Ralph" branding
- **Color Palette**: Custom branded - yellow and blue (Ralph Wiggum inspired)
  - Primary: Yellow for highlights, selection, active elements
  - Secondary: Blue for headers, borders, accents
  - Apply consistently via lipgloss throughout all views

### Navigation

Support **both** navigation styles:
- **Vim-style**: `j`/`k` for up/down, `h`/`l` where applicable
- **Arrow keys**: Standard arrow key navigation
- **Enter**: Select/confirm
- **Esc**: Back/cancel
- **q** or **Ctrl+C**: Quit (keyboard only, not a menu option)

### Help Footer

**Always visible** context-sensitive help bar at bottom of screen showing available keyboard shortcuts for current view.

Example: `j/k:navigate  enter:select  q:quit`

### Terminal Size Handling

**Truncate content** - Cut off content that doesn't fit, rely on natural scrolling behavior. No minimum size warnings.

---

## Feature: Main Menu

### Layout

```
┌─────────────────────────────────────┐
│         [Ralph ASCII Banner]        │
├─────────────────────────────────────┤
│                                     │
│   > Create Task                     │
│     View Tasks                      │
│                                     │
├─────────────────────────────────────┤
│ j/k:navigate  enter:select  q:quit  │
└─────────────────────────────────────┘
```

### Menu Items

1. **Create Task** - Launch task creation wizard
2. **View Tasks** - Open task list view

(Quit is keyboard-only via `q` or `Ctrl+C`)

---

## Feature: Create Task

### Flow Type

**Wizard with preview** - Sequential screens with final preview before saving.

### Wizard Steps

1. **Category Selection**
   - Show three options: bug, feature, refactor
   - Navigate with j/k or arrows, enter to select

2. **Description Input**
   - Text input field for high-level summary
   - **Validation**: Non-empty only (minimal validation)
   - Enter to proceed

3. **Steps Input**
   - **Multi-line textarea** for entering steps
   - One step per line
   - **Parsing**: Trim whitespace and strip empty lines
   - Enter on empty line or keyboard shortcut to proceed

4. **Preview Screen**
   - Display formatted preview of complete task
   - **Available actions**:
     - Confirm (save task)
     - Edit (go back to modify - can return to any previous step)
     - Cancel (discard with confirmation)

### Post-Creation Behavior

**Return to main menu** after successful save.

### Cancel Behavior

**Confirm discard** - Prompt "Discard task? (y/n)" before abandoning wizard and returning to menu.

---

## Feature: View Tasks

### Layout

**Side panel split view**:

```
┌─────────────────────────────────────────────────────────┐
│                   [Ralph Banner]                        │
├─────────────────────────────────────────────────────────┤
│ [All] [bug] [feature] [refactor]                        │
├──────────────────────┬──────────────────────────────────┤
│ Task List            │ Task Details                     │
│                      │                                  │
│ > [bug] Fix login    │ Category: bug                    │
│   [feature] Add API  │ Description: Fix login issue...  │
│   [refactor] Clean   │                                  │
│                      │ Steps:                           │
│                      │ 1. Identify root cause           │
│                      │ 2. Write failing test            │
│                      │ 3. Implement fix                 │
│                      │                                  │
│                      │ Status: Incomplete               │
├──────────────────────┴──────────────────────────────────┤
│ j/k:navigate  tab:filter  esc:menu  q:quit              │
└─────────────────────────────────────────────────────────┘
```

### Task List Panel

- Shows category tag and truncated description
- Selected task highlighted
- **Sort order**: Creation order (oldest first, newest at bottom)

### Detail Panel

- Shows full task information for selected item
- **Steps display**: Numbered list format
  ```
  Steps:
  1. First step
  2. Second step
  3. Third step
  ```
- Shows completion status (Done: Yes/No)

### Category Filter

**Tab/header bar** style:
- Tabs always visible: `[All] [bug] [feature] [refactor]`
- Tab key or click to switch between filters
- Active filter highlighted
- "All" shows all tasks regardless of category

### Empty State

**Simple message**: "No tasks yet. Press enter to create one." with hint about keyboard shortcut.

### Mode

**Read-only** - No editing, deleting, or status toggling in this version. Future enhancement.

---

## Error Handling

### YAML Parse Errors

**Show error and exit** - Display detailed parse error message with line number if available, then exit. User must fix file manually before relaunching.

Example:
```
Error: Failed to parse tasks file
  .ralph/tasks.yaml:15: mapping values not allowed here

Please fix the YAML syntax and restart Ralph.
```

### File System Errors

- Permission denied: Show error message, exit
- Disk full: Show error message, exit
- Directory creation failure: Show error with path, exit

---

## YAML File Examples

### Empty Tasks File

```yaml
[]
```

### Tasks File with Data

```yaml
- id: "20250105091530"
  category: feature
  description: Add user authentication to the API
  steps:
    - Design auth schema
    - Implement JWT middleware
    - Add login endpoint
    - Write integration tests
  done: false

- id: "20250105142200"
  category: bug
  description: Fix memory leak in websocket handler
  steps:
    - Profile memory usage
    - Identify leak source
    - Implement fix
    - Verify with load test
  done: true

- id: "20250105143022"
  category: refactor
  description: Extract database queries into repository pattern
  steps:
    - Create repository interfaces
    - Implement concrete repositories
    - Update services to use repositories
    - Remove direct DB access from handlers
  done: false
```

### Config File

```yaml
tasks_file: .ralph/tasks.yaml
```

---

## Implementation Notes

### Bubble Tea Structure

Follow ELM architecture with flattened package structure:

```
/
├── main.go              # Entry point, program initialization
├── internal/
│   ├── config.go        # Config loading and defaults
│   ├── task.go          # Task model and YAML operations
│   └── ui/
│       ├── app.go       # Root model coordinating views
│       ├── setup.go     # First-time setup wizard
│       ├── menu.go      # Main menu component
│       ├── create.go    # Create task wizard
│       ├── view.go      # View tasks with side panel
│       ├── theme.go     # Color palette and styling
│       ├── help.go      # Help bar rendering
│       └── confirm.go   # Confirmation dialog component
└── .ralph.yaml          # Example config
```

**Package design**: Only 2 packages under internal (`internal` and `internal/ui`). Avoid single-file subpackages - group related code in the same package.

### Key Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Text input, viewport, etc.
- `gopkg.in/yaml.v3` - YAML parsing

### Reference Examples

Consult Bubble Tea examples for implementation patterns:
- https://github.com/charmbracelet/bubbletea/tree/main/examples

---

## Future Considerations (Out of Scope)

The following are explicitly **not** in this version but noted for future:

- Task editing and deletion
- Toggle done status from view
- Search/text filtering
- Custom categories
- Theme customization in config
- Task due dates or priorities
- Export functionality
- Multi-project support
