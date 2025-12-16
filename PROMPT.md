# Temporal TUI - Implementation Prompt

## Project Overview

You are continuing development on `loom`, a terminal UI for Temporal workflow visualization built with Go and tview. The UI layer is complete with mock data. Your task is to implement real Temporal SDK integration.

## Current State

- **UI Complete**: All views, navigation, styling, and keybindings work with mock data
- **Tech Stack**: Go 1.21+, github.com/rivo/tview, github.com/gdamore/tcell/v2
- **Design Pattern**: k9s-inspired architecture with Component interface, page stack navigation, action registry

## Project Structure

```
loom/
├── cmd/
│   └── main.go                    # Entry point (needs provider injection)
├── internal/
│   ├── ui/                        # UI primitives (DO NOT MODIFY)
│   │   ├── app.go                 # Application wrapper, header
│   │   ├── pages.go               # Page stack, Component interface
│   │   ├── table.go               # Generic table component
│   │   ├── menu.go                # Keybinding hints
│   │   ├── crumbs.go              # Breadcrumb navigation
│   │   ├── action.go              # Key action registry
│   │   ├── styles.go              # Theme colors, Nerd Font icons
│   │   └── INTEGRATION.md         # Integration guide (READ THIS)
│   ├── view/                      # Views (need provider injection)
│   │   ├── app.go                 # Main controller
│   │   ├── namespace_list.go      # Namespace browser
│   │   ├── workflow_list.go       # Workflow browser
│   │   ├── workflow_detail.go     # Workflow detail view
│   │   ├── event_history.go       # Event history view
│   │   └── task_queue.go          # Task queue view
│   └── mock/
│       └── data.go                # Mock data (to be replaced)
├── go.mod
└── go.sum
```

## Key Interface

```go
// internal/ui/pages.go
type Component interface {
    tview.Primitive
    Name() string
    Start()           // Called when view becomes active
    Stop()            // Called when view is deactivated
    Hints() []KeyHint // Keybindings for menu
}
```

## Implementation Tasks

### 1. Create Provider Interface

Create `internal/temporal/provider.go`:

```go
type Provider interface {
    ListNamespaces(ctx context.Context) ([]Namespace, error)
    ListWorkflows(ctx context.Context, namespace string, opts ListOptions) ([]Workflow, error)
    GetWorkflow(ctx context.Context, namespace, workflowID, runID string) (*Workflow, error)
    GetWorkflowHistory(ctx context.Context, namespace, workflowID, runID string) ([]HistoryEvent, error)
    DescribeTaskQueue(ctx context.Context, namespace, taskQueue string) (*TaskQueueInfo, error)

    // Actions
    CancelWorkflow(ctx context.Context, namespace, workflowID, runID string) error
    TerminateWorkflow(ctx context.Context, namespace, workflowID, runID, reason string) error
    SignalWorkflow(ctx context.Context, namespace, workflowID, runID, signalName string, input interface{}) error
}
```

### 2. Implement Temporal SDK Client

Create `internal/temporal/client.go` using:
- `go.temporal.io/sdk/client`
- `go.temporal.io/api/workflowservice/v1`

### 3. Update Views for Async Data

Each view needs:
- Provider injection via constructor
- Async data loading with `go func()` + `QueueUpdateDraw()`
- Loading states
- Error handling
- Optional: periodic refresh

### 4. Update Main Entry Point

Add CLI flags:
- `--address` (default: localhost:7233)
- `--namespace` (default: default)
- `--tls-cert`, `--tls-key` (optional)

### 5. Status Mapping

Map `enums.WorkflowExecutionStatus` to UI status strings:
- `WORKFLOW_EXECUTION_STATUS_RUNNING` -> "Running"
- `WORKFLOW_EXECUTION_STATUS_COMPLETED` -> "Completed"
- `WORKFLOW_EXECUTION_STATUS_FAILED` -> "Failed"
- `WORKFLOW_EXECUTION_STATUS_CANCELED` -> "Canceled"
- `WORKFLOW_EXECUTION_STATUS_TERMINATED` -> "Terminated"
- `WORKFLOW_EXECUTION_STATUS_TIMED_OUT` -> "TimedOut"

## Data Models

Current mock models to match:

```go
type Namespace struct {
    Name            string
    State           string
    RetentionPeriod string
}

type Workflow struct {
    ID        string
    RunID     string
    Type      string
    Status    string    // "Running", "Completed", "Failed", etc.
    Namespace string
    TaskQueue string
    StartTime time.Time
    EndTime   *time.Time
    ParentID  *string
}

type HistoryEvent struct {
    ID      int64
    Type    string
    Time    time.Time
    Details string
}

type TaskQueue struct {
    Name        string
    Type        string    // "Workflow" or "Activity"
    PollerCount int
    Backlog     int
}
```

## Thread Safety

UI updates must use:
```go
app.UI().QueueUpdateDraw(func() {
    // Update UI here
})
```

## Dependencies to Add

```bash
go get go.temporal.io/sdk
go get go.temporal.io/api
```

## Reference Files

1. **Read first**: `internal/ui/INTEGRATION.md` - Full integration guide
2. **Mock data reference**: `internal/mock/data.go` - Data structures and examples
3. **View patterns**: `internal/view/workflow_list.go` - Example view implementation

## Success Criteria

1. App connects to Temporal server via CLI flags
2. Namespaces load from real server
3. Workflows list with real data, status colors work
4. Workflow detail shows real workflow info
5. Event history displays real events
6. Task queue shows real pollers/backlog
7. Navigation and keybindings still work
8. Graceful error handling (connection failures, timeouts)
9. Loading indicators during data fetch

## Do Not Modify

- `internal/ui/*.go` - UI primitives are complete
- Styling, colors, icons - Already configured

## Optional Enhancements (After Core)

- Workflow cancellation/termination
- Signal sending
- Query execution
- Search/filter with Temporal visibility queries
- Auto-refresh toggle
- Connection status indicator (real)
