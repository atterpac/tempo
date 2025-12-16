# Temporal TUI Mutation Implementation Plan

This document outlines the plan for adding write/mutation operations to the Temporal TUI, transforming it from a read-only viewer to a full management interface.

## Overview

Currently the TUI is read-only. This plan adds the ability to:
- Cancel, terminate, signal, and delete workflows
- Start new workflows
- Reset workflows to previous states
- Manage schedules (pause, trigger, delete)

All mutations will require user confirmation with a preview of the equivalent CLI command.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Keybind/Command                         │
│           (e.g., 'c' for cancel, ':signal' in command bar)      │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Confirmation Modal                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │ Preview:                                                   │  │
│  │   temporal workflow cancel                                 │  │
│  │     --workflow-id order-12345                              │  │
│  │     --namespace production                                 │  │
│  │     --reason "Manual cancellation from TUI"                │  │
│  │                                                            │  │
│  │ [Enter] Confirm    [Esc] Cancel                            │  │
│  └───────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │ (on confirm)
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Provider Method                             │
│         (e.g., provider.CancelWorkflow(ctx, params))            │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│               Temporal SDK / gRPC Call                          │
│     (e.g., client.CancelWorkflowExecution(ctx, workflowID))     │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Phases

### Phase 1: Infrastructure
- Extend `Provider` interface with mutation methods
- Implement mutation methods in `Client`
- Create `ConfirmModal` UI component
- Create `InputModal` UI component for operations requiring input

### Phase 2: Single Workflow Operations
- Cancel workflow (graceful)
- Terminate workflow (forceful)
- Signal workflow (with input)
- Delete workflow

### Phase 3: List Operations & Start
- Quick actions from workflow list view
- Start new workflow via command bar
- Bulk selection (future consideration)

### Phase 4: Advanced Operations
- Reset workflow with event selector
- Schedule management (pause/unpause, trigger, delete)
- Batch operations with query builder

## Priority Operations

### Tier 1 - Essential
| Operation | CLI Equivalent | Keybind | Use Case |
|-----------|---------------|---------|----------|
| Cancel Workflow | `temporal workflow cancel` | `c` | Graceful shutdown |
| Terminate Workflow | `temporal workflow terminate` | `X` | Force stop |
| Signal Workflow | `temporal workflow signal` | `s` | Send signals |
| Delete Workflow | `temporal workflow delete` | `D` | Cleanup |

### Tier 2 - Important
| Operation | CLI Equivalent | Keybind | Use Case |
|-----------|---------------|---------|----------|
| Start Workflow | `temporal workflow start` | `n` / `:start` | Launch new |
| Reset Workflow | `temporal workflow reset` | `R` | Replay |

### Tier 3 - Schedule Operations
| Operation | CLI Equivalent | Use Case |
|-----------|---------------|----------|
| Pause/Unpause Schedule | `temporal schedule toggle` | Control |
| Trigger Schedule | `temporal schedule trigger` | Manual run |
| Delete Schedule | `temporal schedule delete` | Cleanup |

## Files to Create/Modify

### New Files
- `internal/ui/confirm.go` - Confirmation modal component
- `internal/ui/input.go` - Input modal for signal/start operations
- `internal/temporal/mutations.go` - Mutation method implementations

### Modified Files
- `internal/temporal/provider.go` - Add mutation interface methods
- `internal/temporal/client.go` - Implement mutation methods
- `internal/view/workflow_detail.go` - Add keybinds for mutations
- `internal/view/workflow_list.go` - Add keybinds for list mutations
- `internal/ui/commandbar.go` - Enhanced command parsing

## Success Criteria

1. All mutations require confirmation before execution
2. Confirmation modal shows CLI-equivalent command
3. Destructive operations (terminate, delete) use capital letter keybinds
4. Operations that need input use command bar or input modal
5. Error handling with user-friendly messages
6. Loading states during async operations
