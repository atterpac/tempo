# Implementation Prompt for Temporal TUI Mutations

Copy and use this prompt to start a new LLM conversation for implementing mutations.

---

## Prompt

```
I'm building a Temporal TUI (terminal user interface) in Go using tview/tcell. The TUI is currently read-only and I want to add write/mutation operations.

## Project Context

- **Language:** Go
- **UI Framework:** tview (github.com/rivo/tview) with tcell
- **Temporal SDK:** go.temporal.io/sdk/client
- **Project Structure:**
  - `internal/temporal/provider.go` - Provider interface (abstraction over Temporal client)
  - `internal/temporal/client.go` - SDK client implementation
  - `internal/ui/` - UI components (commandbar.go, table.go, panel.go, etc.)
  - `internal/view/` - Views (workflow_list.go, workflow_detail.go, etc.)
  - `cmd/main.go` - Entry point

## Documentation

Read these files for the implementation plan:
- `docs/MUTATION_IMPLEMENTATION_PLAN.md` - High-level plan and phases
- `docs/MUTATION_TECHNICAL_SPEC.md` - Detailed technical specification with code examples
- `docs/TEMPORAL_CLI_REFERENCE.md` - CLI command reference for all operations

## Current State

The TUI already has:
1. A `CommandBar` component (`internal/ui/commandbar.go`) with `CommandAction` type ready for `:` commands
2. A `Provider` interface that the views use for data access
3. Theme selector modal as reference for modal implementation
4. Key hint system in the menu bar

## Requirements

1. **All mutations must require confirmation** - Show a modal with:
   - Operation description
   - Target workflow/schedule info
   - Equivalent CLI command (for learning)
   - Warning for destructive operations
   - [Enter] Confirm / [Esc] Cancel

2. **Keybinds for common operations:**
   - `c` - Cancel workflow (graceful)
   - `X` - Terminate workflow (destructive, capital letter)
   - `s` - Signal workflow (opens input for signal name/data)
   - `D` - Delete workflow (destructive, capital letter)
   - `R` - Reset workflow (opens event selector)

3. **Command bar for complex operations:**
   - `:signal <workflow-id> <signal-name> [json-input]`
   - `:start <workflow-type> --task-queue <queue>`

## Implementation Order

Start with Phase 1:
1. Add mutation methods to Provider interface
2. Implement methods in Client (use Temporal SDK)
3. Create ConfirmModal component
4. Add cancel/terminate to workflow_detail.go

## Key Files to Read First

1. `internal/temporal/provider.go` - Current interface
2. `internal/temporal/client.go` - Current SDK usage
3. `internal/view/workflow_detail.go` - View where mutations will be added
4. `internal/ui/commandbar.go` - Existing command bar implementation
5. `internal/view/app.go` - See showThemeSelector() for modal pattern

## Style Guidelines

- Match existing code style
- Use the existing theme/color system (ui.ColorBg(), ui.TagFg(), etc.)
- Keep modals consistent with theme selector style
- Use existing Panel component for consistent borders

Please start by reading the documentation files and key source files, then begin implementing Phase 1 starting with the Provider interface extensions.
```

---

## Alternative Shorter Prompt

```
I'm adding write operations to a Temporal TUI built with Go/tview.

Read these docs first:
- docs/MUTATION_IMPLEMENTATION_PLAN.md
- docs/MUTATION_TECHNICAL_SPEC.md
- docs/TEMPORAL_CLI_REFERENCE.md

Then read:
- internal/temporal/provider.go (interface to extend)
- internal/temporal/client.go (SDK implementation)
- internal/view/workflow_detail.go (add keybinds here)
- internal/view/app.go (modal pattern in showThemeSelector)

Start with Phase 1:
1. Extend Provider interface with CancelWorkflow, TerminateWorkflow, SignalWorkflow, DeleteWorkflow
2. Implement in Client using Temporal SDK
3. Create ConfirmModal component (similar style to theme selector)
4. Add 'c' keybind to workflow_detail.go for cancel with confirmation

All mutations need confirmation modal showing the equivalent CLI command.
```

---

## Files to Reference

When starting the implementation, the LLM should read these files:

### Documentation (created above)
- `docs/MUTATION_IMPLEMENTATION_PLAN.md`
- `docs/MUTATION_TECHNICAL_SPEC.md`
- `docs/TEMPORAL_CLI_REFERENCE.md`

### Source Files
- `internal/temporal/provider.go`
- `internal/temporal/client.go`
- `internal/ui/commandbar.go`
- `internal/ui/styles.go`
- `internal/view/app.go`
- `internal/view/workflow_detail.go`
- `internal/view/workflow_list.go`

### For Reference (modal patterns)
- `internal/view/app.go` - `showThemeSelector()` method shows modal pattern
