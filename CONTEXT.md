# Temporal TUI - Implementation Context

> **Last Updated:** Phase 5 - Testing & Polish Complete
> **Status:** All phases complete - Ready for production use

## Project Overview

Implementing real Temporal SDK integration for `loom`, a terminal UI for Temporal workflow visualization. The UI layer is complete with mock data.

## User Requirements

- **Startup:** Retry with UI - Show "Connecting..." with retry attempts, allow quit
- **Refresh:** Per-view toggle - Each view has 'a' key to toggle auto-refresh
- **Actions:** Read-only for v1 - No cancel/terminate/signal
- **TLS:** Full options - `--tls-cert`, `--tls-key`, `--tls-ca`, `--tls-server-name`, `--tls-skip-verify`

## Implementation Phases

### Phase 1: Core Infrastructure ✅
- [x] Create `internal/temporal/provider.go` (interface + models)
- [x] Create `internal/temporal/status.go` (status mapping)
- [x] Create `internal/temporal/client.go` (SDK implementation)

### Phase 2: CLI and Entry Point ✅
- [x] Update `cmd/main.go` with CLI flags
- [x] Add TLS configuration support
- [x] Implement retry-with-UI startup flow

### Phase 3: View Layer Updates ✅
- [x] Update `internal/view/app.go` (provider injection)
- [x] Update `internal/view/namespace_list.go` (async + auto-refresh)
- [x] Update `internal/view/workflow_list.go` (async + auto-refresh)
- [x] Update `internal/view/workflow_detail.go` (async + refresh)
- [x] Update `internal/view/event_history.go` (async + refresh)
- [x] Update `internal/view/task_queue.go` (async + refresh)

### Phase 4: Connection Management ✅
- [x] Implement startup connection flow with retries
- [x] Add runtime reconnection handling
- [x] Update header connection status dynamically

### Phase 5: Testing & Polish ✅
- [x] Test against local Temporal server
- [x] Verify all navigation still works
- [x] Ensure graceful error handling
- [x] Fix startup deadlock issue
- [x] Fix event history details extraction
- [x] Fix event type formatting
- [x] Fix workflow detail tab navigation
- [x] Fix task queue refresh loop
- [x] Update UI to Charm-style (borderless, minimal)
- [x] Standardize icons to Font Awesome
- [x] Create testdata workflow generator

---

## File Inventory

### Files to Create
| File | Purpose | Status |
|------|---------|--------|
| `internal/temporal/provider.go` | Interface + data models | ✅ |
| `internal/temporal/status.go` | Status enum mapping | ✅ |
| `internal/temporal/client.go` | SDK client implementation | ✅ |

### Files to Modify
| File | Changes | Status |
|------|---------|--------|
| `cmd/main.go` | CLI flags, provider init, retry-with-UI startup | ✅ |
| `internal/view/app.go` | Provider field, injection, runID support | ✅ |
| `internal/view/namespace_list.go` | Async loading, auto-refresh | ✅ |
| `internal/view/workflow_list.go` | Async loading, auto-refresh | ✅ |
| `internal/view/workflow_detail.go` | Async loading, refresh | ✅ |
| `internal/view/event_history.go` | Async loading, refresh | ✅ |
| `internal/view/task_queue.go` | Async loading, refresh | ✅ |
| `go.mod` | Add temporal SDK deps | ✅ |

### Files NOT to Modify
- `internal/ui/*.go` - UI primitives are complete

---

## Technical Specifications

### Data Models (match existing mock structures)

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
    Status    string    // "Running", "Completed", "Failed", "Canceled", "Terminated", "TimedOut"
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

type TaskQueueInfo struct {
    Name        string
    Type        string    // "Workflow" or "Activity"
    PollerCount int
    Backlog     int
}

type Poller struct {
    Identity       string
    LastAccessTime time.Time
    TaskQueueType  string
}
```

### Provider Interface

```go
type Provider interface {
    ListNamespaces(ctx context.Context) ([]Namespace, error)
    ListWorkflows(ctx context.Context, namespace string, opts ListOptions) ([]Workflow, error)
    GetWorkflow(ctx context.Context, namespace, workflowID, runID string) (*Workflow, error)
    GetWorkflowHistory(ctx context.Context, namespace, workflowID, runID string) ([]HistoryEvent, error)
    DescribeTaskQueue(ctx context.Context, namespace, taskQueue string) (*TaskQueueInfo, []Poller, error)
    Close() error
    IsConnected() bool
}

type ListOptions struct {
    PageSize  int
    PageToken []byte
    Query     string
}
```

### CLI Flags

```
--address         Temporal server address (default: localhost:7233)
--namespace       Default namespace (default: default)
--tls-cert        Path to TLS certificate
--tls-key         Path to TLS private key
--tls-ca          Path to CA certificate
--tls-server-name Server name for TLS verification
--tls-skip-verify Skip TLS verification (insecure)
```

### Async Loading Pattern

```go
func (v *View) loadData() {
    v.setLoading(true)
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        data, err := v.provider.FetchData(ctx, ...)

        v.app.UI().QueueUpdateDraw(func() {
            v.setLoading(false)
            if err != nil {
                v.showError(err)
                return
            }
            v.populateTable(data)
        })
    }()
}
```

### Auto-Refresh Pattern

```go
type View struct {
    // ...
    autoRefresh   bool
    refreshTicker *time.Ticker
    stopRefresh   chan struct{}
}

func (v *View) Start() {
    v.setupKeybindings() // includes 'a' for toggle, 'r' for manual refresh
    v.loadData()
}

func (v *View) toggleAutoRefresh() {
    v.autoRefresh = !v.autoRefresh
    if v.autoRefresh {
        v.startAutoRefresh()
    } else {
        v.stopAutoRefresh()
    }
}

func (v *View) Stop() {
    v.stopAutoRefresh()
}
```

---

## Dependencies to Add

```bash
go get go.temporal.io/sdk
go get go.temporal.io/api
```

---

## Progress Log

### Phase 5 - Testing & Polish (Complete)
- **Startup deadlock fix** - `cmd/main.go`:
  - Separated `setStatusText` for direct text setting before app runs
  - Created `updateStatus` wrapper using `QueueUpdateDraw` for after app is running
  - Added `appRunning` channel with `sync.Once` for proper synchronization
- **Event history details extraction** - `internal/temporal/client.go`:
  - Implemented comprehensive `extractEventDetails` function (~400 lines)
  - Handles all major Temporal event types with verbose details
  - Fixed `formatEventType` to preserve PascalCase for already-formatted event types
- **Tab navigation fix** - `internal/view/workflow_detail.go`:
  - Fixed Tab key to navigate directly to Events view
  - Simplified tab bar to show info/events navigation hints
- **Task queue refresh loop fix** - `internal/view/task_queue.go`:
  - Added `suppressSelect` flag to prevent recursive selection handling
  - Protected `SelectRow` calls in `populateQueueTable` and `updateQueueInfo`
  - Task queues now discovered from workflows (Temporal has no ListTaskQueues API)
- **UI style update** - Changed from k9s-style to Charm-style:
  - Updated `internal/ui/styles.go` with Catppuccin Mocha color palette
  - Standardized all icons to Font Awesome for better Nerd Font compatibility
  - Removed borders from tables and panels throughout all views
  - Simplified headers, menus, and breadcrumbs for minimal aesthetic
- **Test data generator** - Created `testdata/main.go` and `testdata/workflows.go`:
  - 9 different workflow types (Quick, Slow, Failing, Activity, Timer, Signal, Child, LongRunning, Retry)
  - 4 activity types (Process, Heartbeat, Flakey, Slow)
  - Weighted random workflow starter for realistic test data
- Verified code compiles successfully

### Phase 4 - Connection Management (Complete)
- Updated `internal/temporal/provider.go`:
  - Added `CheckConnection(ctx context.Context) error` method to verify connection is alive
  - Added `Reconnect(ctx context.Context) error` method for automatic reconnection
  - Added `Config() ConnectionConfig` method to access connection settings
- Updated `internal/temporal/client.go`:
  - Implemented `CheckConnection()` using lightweight ListNamespaces API call
  - Implemented `Reconnect()` with proper cleanup of old client and TLS reconfiguration
  - Implemented `Config()` getter for connection configuration
- Updated `internal/view/app.go`:
  - Added connection monitor constants (10s check interval, 2-30s reconnect backoff)
  - Added `stopMonitor` channel and `reconnecting` flag to App struct
  - Added `connectionMonitor()` goroutine that periodically checks connection health
  - Added `attemptReconnect()` method with exponential backoff for reconnection
  - Added `Stop()` method to cleanly stop connection monitor and application
  - Updated global quit handler to use new `Stop()` method
  - Connection monitor updates header status dynamically (Connected/Disconnected)
- Startup connection flow with UI was already implemented in Phase 2 (cmd/main.go)
- Verified code compiles successfully

### Phase 3 - View Layer Updates (Complete)
- Updated `internal/view/namespace_list.go`:
  - Changed from local namespace struct to `temporal.Namespace` type
  - Added async data loading via `loadData()` with `QueueUpdateDraw()`
  - Added loading indicator in title bar
  - Added auto-refresh support with 'a' key toggle (5 second interval)
  - Added manual refresh with 'r' key
  - Mock data fallback when no provider is configured
- Updated `internal/view/workflow_list.go`:
  - Changed from `mock.Workflow` to `temporal.Workflow` type
  - Added async data loading with provider.ListWorkflows()
  - Added auto-refresh support with 'a' key toggle
  - Added manual refresh with 'r' key
  - Updated to pass RunID to NavigateToWorkflowDetail
  - Added filter support with visibility query
- Updated `internal/view/app.go`:
  - Modified `NavigateToWorkflowDetail(workflowID, runID)` to accept RunID
  - Modified `NavigateToEvents(workflowID, runID)` to accept RunID
- Updated `internal/view/workflow_detail.go`:
  - Changed from `*mock.Workflow` to `*temporal.Workflow` type
  - Added runID field and tracking
  - Added async data loading via provider.GetWorkflow()
  - Added manual refresh with 'r' key
  - Updated NavigateToEvents call to include runID
- Updated `internal/view/event_history.go`:
  - Changed from `[]mock.HistoryEvent` to `[]temporal.HistoryEvent` type
  - Added runID field and tracking
  - Added async data loading via provider.GetWorkflowHistory()
  - Added manual refresh with 'r' key (30 second timeout for large histories)
- Updated `internal/view/task_queue.go`:
  - Created local `taskQueueEntry` struct for queue list
  - Changed pollers to `[]temporal.Poller` type
  - Shows mock queue list (provider doesn't have ListTaskQueues)
  - When queue selected, fetches real pollers via provider.DescribeTaskQueue()
  - Added manual refresh with 'r' key for selected queue
- All views now:
  - Get provider from `app.Provider()`
  - Show loading indicators while fetching
  - Handle errors gracefully with error display
  - Support refresh functionality
- Verified code compiles successfully

### Phase 2 - CLI and Entry Point (Complete)
- Updated `cmd/main.go` with full CLI flag support:
  - `--address`: Temporal server address (default: localhost:7233)
  - `--namespace`: Default namespace (default: default)
  - `--tls-cert`, `--tls-key`, `--tls-ca`, `--tls-server-name`, `--tls-skip-verify`: TLS options
- Implemented retry-with-UI startup flow:
  - Shows "Connecting..." with retry count and address
  - Exponential backoff (1s, 2s, 4s, 8s, 10s max)
  - Max 5 retries before giving up
  - User can quit with 'q' at any time
  - Brief "Connected!" message before launching main app
- Updated `internal/view/app.go`:
  - Added `provider temporal.Provider` field
  - Added `NewAppWithProvider(provider, namespace)` constructor
  - Added `Provider()` getter method
  - Sets initial connection status in header
- Verified code compiles successfully

### Phase 1 - Core Infrastructure (Complete)
- Created `internal/temporal/provider.go` with Provider interface and data models (Namespace, Workflow, HistoryEvent, TaskQueueInfo, Poller, ConnectionConfig)
- Created `internal/temporal/status.go` with status mapping functions (MapWorkflowStatus, MapNamespaceState, MapTaskQueueType)
- Created `internal/temporal/client.go` with full SDK client implementation:
  - NewClient with TLS configuration support
  - ListNamespaces, ListWorkflows, GetWorkflow, GetWorkflowHistory, DescribeTaskQueue
  - Connection state management
- Added Temporal SDK dependencies (go.temporal.io/sdk v1.38.0, go.temporal.io/api v1.59.0)
- Verified code compiles successfully

### Phase 0 - Planning (Complete)
- Analyzed codebase structure
- Gathered user requirements via questions
- Created implementation plan
- Created CONTEXT.md

---

## Next Phase Instructions

When starting a new agent session, use the prompt in the "Agent Handoff Prompt" section below.

---

## Agent Handoff Prompt

All phases are complete. The application is ready for production use.

```
loom - Temporal Workflow Visualization TUI

All implementation phases complete:
- Phase 1: Core Infrastructure ✅
- Phase 2: CLI and Entry Point ✅
- Phase 3: View Layer Updates ✅
- Phase 4: Connection Management ✅
- Phase 5: Testing & Polish ✅

To run the application:
  go run ./cmd/main.go --address localhost:7233 --namespace default

To generate test data:
  pushd testdata && go run . -mode both -count 20; popd

Key features:
- Real Temporal SDK integration with all views
- Charm-style minimal UI with Catppuccin Mocha colors
- Connection retry with UI feedback
- Auto-reconnection on connection loss
- Auto-refresh toggle ('a') and manual refresh ('r')
- Navigation: workflows, details, events, task queues

Key files:
- cmd/main.go (entry point with CLI flags)
- internal/view/*.go (all view implementations)
- internal/temporal/client.go (SDK client)
- internal/ui/styles.go (colors and icons)
- testdata/ (workflow generator for testing)
```
