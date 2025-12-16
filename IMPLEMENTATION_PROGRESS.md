# Temporal TUI Feature Implementation Progress

## Overview
Implementing advanced features for the Temporal TUI (loom) application.

## Completed Features

### Phase 1: Query Workflows ✅

**Files Modified:**
- `internal/temporal/provider.go` - Added interface methods:
  - `QueryWorkflow(ctx, namespace, workflowID, runID, queryType string, args []byte) (*QueryResult, error)`
  - `CancelWorkflows(ctx, namespace string, workflows []WorkflowIdentifier) ([]BatchResult, error)`
  - `TerminateWorkflows(ctx, namespace string, workflows []WorkflowIdentifier, reason string) ([]BatchResult, error)`
  - `GetResetPoints(ctx, namespace, workflowID, runID string) ([]ResetPoint, error)`

- `internal/temporal/client.go` - Implemented all new provider methods (lines 1595-1767)

**Files Created:**
- `internal/ui/query_result_modal.go` - Modal for displaying query results with:
  - JSON-formatted result display
  - Error display
  - Copy to clipboard (`y` key)
  - Scrollable content

**Files Modified:**
- `internal/view/workflow_detail.go` - Added:
  - `Q` keybinding for query (line 354)
  - `showQueryInput()` method (lines 733-767)
  - `executeQuery()` method (lines 769-806)
  - `showQueryResult()` and `showQueryError()` methods (lines 808-829)

**New Types in provider.go:**
```go
type QueryResult struct {
    QueryType string
    Result    string // JSON-formatted
    Error     string
}

type WorkflowIdentifier struct {
    WorkflowID string
    RunID      string
}

type BatchResult struct {
    WorkflowID string
    RunID      string
    Success    bool
    Error      string
}

type ResetPoint struct {
    EventID     int64
    EventType   string
    Timestamp   time.Time
    Description string
    Reason      string
}
```

### Phase 2: Batch Operations with Multi-Select ✅

**Files Modified:**
- `internal/ui/table.go` - Added multi-select support (lines 193-399):
  - `EnableSelection()` / `DisableSelection()`
  - `ToggleSelection()` / `ToggleRowSelection(row int)`
  - `SelectRowMulti(row int)` / `DeselectRow(row int)`
  - `IsRowSelected(row int) bool`
  - `GetSelectedRows() []int`
  - `SelectionCount() int`
  - `SelectAll()` / `ClearSelection()`
  - `SetOnSelectionChange(fn func(selected []int))`
  - Visual checkmarks and background highlighting

- `internal/ui/commandbar.go` - Added `CommandQuery` type (line 17)

**Files Created:**
- `internal/ui/batch_confirm_modal.go` - Batch operation modal with:
  - `BatchOperation` enum (Cancel, Terminate, Delete)
  - `BatchItem` struct for tracking individual items
  - Confirmation view with warnings
  - Progress tracking with visual progress bar
  - Per-item status updates
  - `StartProgress()`, `MarkItemCompleted()`, `MarkItemFailed()` methods

**Files Modified:**
- `internal/view/workflow_list.go` - Added batch operations (lines 580-804):
  - `selectionMode` field
  - `v` key to toggle selection mode
  - `Space` to toggle row selection
  - `Ctrl+A` to select all
  - `c` for batch cancel (in selection mode)
  - `X` for batch terminate (in selection mode)
  - `toggleSelectionMode()` method
  - `updateSelectionPreview()` method
  - `showBatchCancelConfirm()` / `showBatchTerminateConfirm()` methods
  - `executeBatchCancel()` / `executeBatchTerminate()` methods
  - Updated `Hints()` to show selection mode hints

---

## Remaining Features

### Phase 3: Query Builder with Autocomplete ✅

**Files Created:**
- `internal/ui/autocomplete.go` - Autocomplete input and query template components:
  - `Suggestion` struct with Text, InsertText, Description, Category
  - `QueryTemplate` struct with Name, Description, Query
  - `TemporalVisibilityFields` - WorkflowId, WorkflowType, ExecutionStatus, StartTime, CloseTime, etc.
  - `TemporalOperators` - =, !=, >, <, BETWEEN, AND, OR, ORDER BY
  - `TemporalStatusValues` - Running, Completed, Failed, Canceled, Terminated, TimedOut
  - `TemporalTimeExpressions` - now(), now()-1h, now()-24h, now()-7d, now()-30d
  - `DefaultQueryTemplates` - 6 pre-defined query templates
  - `AutocompleteInput` struct:
    - Shows context-aware suggestions as user types
    - Keyboard navigation (up/down to select, tab/enter to accept)
    - Custom suggestion provider support
  - `QueryTemplateSelector` struct:
    - Quick select for predefined query templates
    - Number key shortcuts (1-9) for quick selection

**Files Modified:**
- `internal/ui/commandbar.go` - Added `CommandVisibility` type (line 18)

- `internal/view/workflow_list.go` - Added visibility query support:
  - `visibilityQuery` field (line 30)
  - `F` keybinding for visibility query with autocomplete (line 395-398)
  - `f` keybinding for query templates (line 399-402)
  - `C` keybinding to clear visibility query (lines 437-442)
  - `showVisibilityQuery()` method (lines 820-854)
  - `showQueryTemplates()` method (lines 861-897)
  - `showTemplateInput()` method (lines 904-934) - for templates with placeholders
  - `updatePanelTitle()` method (lines 941-954) - shows active query in panel title
  - `clearVisibilityQuery()` method (lines 956-960)
  - Updated `loadData()` to use `visibilityQuery` (line 200)
  - Updated `Hints()` to show "Clear Query" when query is active (lines 490-492)

- `internal/view/app.go` - Added modal page handling for visibility-query, query-templates, template-input (lines 94-96)

**Query Templates Available:**
| Name | Query |
|------|-------|
| Running | `ExecutionStatus='Running'` |
| Failed (24h) | `ExecutionStatus='Failed' AND CloseTime > now()-24h` |
| Timed Out | `ExecutionStatus='TimedOut'` |
| Long Running | `ExecutionStatus='Running' AND StartTime < now()-1h` |
| Recently Completed | `ExecutionStatus='Completed' AND CloseTime > now()-1h` |
| By Type | `WorkflowType='${type}'` (prompts for value) |

**Keybindings (Workflow List):**
| Key | Action |
|-----|--------|
| `F` | Open visibility query with autocomplete |
| `f` | Show query templates |
| `C` | Clear active visibility query |

### Phase 4: Date Range Picker ✅

**Files Created:**
- `internal/ui/date_range.go` - Date range picker component:
  - `DateRangePreset` struct with Name, Description, Duration, Query
  - `DefaultDatePresets` - 1h, 24h, 7d, 30d, All
  - `DateRangePicker` struct:
    - Preset selection (1-5 number keys for quick select)
    - Custom duration input (e.g., "3d", "2w", "4h", "30m")
    - Tab to switch between preset and custom mode
    - Supports both StartTime and CloseTime filtering
  - `parseCustomDuration()` - Parses duration strings
  - `formatDurationForQuery()` - Converts to Temporal query format

**Files Modified:**
- `internal/view/workflow_list.go` - Added date range picker support:
  - `D` keybinding to open date range picker (line 403-406)
  - `showDateRangePicker()` method (lines 981-1020)
  - `closeDateRangePicker()` method (lines 1022-1025)
  - `clearDateFromQuery()` method (lines 1027-1039) - Removes date conditions from query
  - Updated `Hints()` to include date range hint (line 493)

- `internal/view/app.go` - Added modal page handling for date-range (line 97)

**Presets Available:**
| Key | Name | Query Fragment |
|-----|------|----------------|
| 1 | 1h | `StartTime > now()-1h` |
| 2 | 24h | `StartTime > now()-24h` |
| 3 | 7d | `StartTime > now()-7d` |
| 4 | 30d | `StartTime > now()-30d` |
| 5 | All | (no filter) |

**Custom Input:**
- Supports formats: `30m`, `4h`, `3d`, `2w`
- Press Tab to switch to custom input mode
- Validates input in real-time

### Phase 5: Saved Filters ✅

**Files Modified:**
- `internal/config/config.go` - Added saved filter support:
  - `SavedFilter` struct with Name, Query, IsDefault (lines 50-55)
  - Added `SavedFilters []SavedFilter` to Config struct (line 62)
  - `GetSavedFilters()` method (lines 287-290)
  - `GetSavedFilter(name)` method (lines 292-300)
  - `SaveFilter(filter)` method (lines 302-313)
  - `DeleteFilter(name)` method (lines 315-324)
  - `GetDefaultFilter()` method (lines 326-334)
  - `SetDefaultFilter(name)` method (lines 336-351)
  - `ClearDefaultFilter()` method (lines 353-358)

**Files Created:**
- `internal/ui/filter_picker.go` - Filter picker modal with:
  - `FilterPicker` struct - Full filter management UI
  - `filterPickerModeSelect` / `filterPickerModeSave` modes
  - Select, save, delete, and set default operations
  - Number key shortcuts (1-9) for quick selection
  - `QuickSaveDialog` struct - Quick save dialog for current query

**Files Modified:**
- `internal/view/workflow_list.go` - Added saved filter integration:
  - `L` keybinding to load saved filters (lines 449-452)
  - `S` keybinding to save current filter (lines 453-458)
  - `showSavedFilters()` method (lines 1059-1119)
  - `closeSavedFilters()` method (lines 1121-1124)
  - `showSaveFilter()` method (lines 1126-1152)
  - `closeSaveFilter()` method (lines 1154-1157)
  - Updated `Hints()` to show save filter hint when query active (lines 508-511)

- `internal/view/app.go` - Added modal page handling for saved-filters, save-filter (lines 98-99)

**Keybindings (Workflow List):**
| Key | Action |
|-----|--------|
| `L` | Load saved filter |
| `S` | Save current filter (when query active) |

**Filter Picker Features:**
- List all saved filters with queries
- Mark default filter with checkmark
- `d` to delete selected filter
- `*` to set selected as default
- `s` to save new filter from current query

### Phase 6: Search History ✅

**Files Modified:**
- `internal/view/workflow_list.go` - Added search history:
  - Added `searchHistory []string` field (line 37)
  - Added `historyIndex int` field (line 38)
  - Added `maxHistorySize int` field (line 39)
  - Initialized in NewWorkflowList (lines 53-55)
  - `addToHistory(query)` method (lines 1167-1196)
  - `historyPrevious()` method (lines 1198-1212)
  - `historyNext()` method (lines 1214-1228)
  - `getHistoryStatus()` method (lines 1230-1238)
  - Updated `showVisibilityQuery()` to set up history provider (lines 867-873)
  - Updated `showVisibilityQuery()` to add to history on submit (line 878)

- `internal/ui/autocomplete.go` - Added history navigation support:
  - Added `historyFn func(direction int) string` field (line 165)
  - `SetHistoryProvider()` method (lines 224-227)
  - Updated `InputHandler()` for up/down arrow history navigation (lines 644-670)

**Features:**
- Keeps up to 50 search queries in memory
- No duplicate entries (moves duplicates to end)
- Arrow keys navigate history when suggestions not visible:
  - `↑` - Previous history entry
  - `↓` - Next history entry
- History resets on cancel

### Phase 7: Workflow Diff ✅

**Files Created:**
- `internal/view/workflow_diff.go` - Side-by-side workflow comparison view:
  - `WorkflowDiff` struct with dual panels for workflows A and B
  - `NewWorkflowDiff()` and `NewWorkflowDiffWithWorkflows()` constructors
  - Tabbed panel focus switching
  - `promptWorkflowInput()` for entering workflow IDs
  - `loadWorkflow()` async loading of workflow data and events
  - `formatWorkflowInfo()` displays workflow metadata
  - Event tables for both sides

**Files Modified:**
- `internal/view/app.go`:
  - `NavigateToWorkflowDiff()` method (lines 243-247)
  - `NavigateToWorkflowDiffEmpty()` method (lines 249-253)
  - Added diff-input modal handling (line 100)
  - Added workflow-diff to crumbs (lines 182-183)

- `internal/view/workflow_list.go`:
  - `d` keybinding to start diff (lines 465-468)
  - `startDiff()` method (lines 1258-1268)
  - Added diff hint (line 525)

**Keybindings (Workflow Diff):**
| Key | Action |
|-----|--------|
| `Tab` | Switch between left/right panels |
| `a` | Set left workflow |
| `b` | Set right workflow |
| `r` | Refresh both workflows |

**Features:**
- Side-by-side comparison with info summary
- Event history tables for both workflows
- Dynamic workflow loading by ID
- Visual focus indicator for active panel

### Phase 8: Enhanced Reset Picker ✅

**Files Created:**
- `internal/ui/reset_picker.go` - Enhanced reset picker with:
  - `ResetPicker` struct - Full reset point picker UI
  - `QuickResetModal` struct - Quick reset confirmation for detected failures
  - Visual highlighting of failure events (red)
  - Selection marker and keyboard navigation (j/k, up/down arrows)
  - Number key shortcuts (1-9) for quick selection
  - `GetFirstFailurePoint()` method to detect failure points
  - `isFailureEvent()` helper for failure type detection

**Files Modified:**
- `internal/view/workflow_detail.go` - Replaced EventSelectorModal with enhanced reset picker:
  - `showResetSelector()` now calls `GetResetPoints()` API (lines 634-675)
  - `showQuickResetModal()` for automatic failure detection (lines 678-696)
  - `showResetPicker()` for full reset point selection (lines 699-726)
  - Auto-detects failures and shows quick reset confirmation
  - Press 'a' in quick reset to access all reset points

**Keybindings (Workflow Detail):**
| Key | Action |
|-----|--------|
| `R` | Reset workflow (shows quick reset if failure detected) |
| `Enter` | Confirm reset to selected point |
| `a` | Advanced mode (show all reset points from quick reset) |
| `1-9` | Quick select reset point by number |
| `j/k` | Navigate reset points |
| `Esc` | Cancel |

**Features:**
- Automatic failure detection (ActivityTaskFailed, WorkflowExecutionFailed, etc.)
- Quick reset mode shows last failure point for one-click reset
- Advanced mode lists all valid reset points with descriptions
- Failure events highlighted in red for visibility
- Uses `GetResetPoints()` API for accurate reset point discovery

---

## Architecture Notes

- **Provider Pattern**: All Temporal operations go through `temporal.Provider` interface
- **Theme System**: Use `ui.ColorXxx()` and `ui.TagXxx()` functions for colors
- **Modal Pattern**: Extend `ui.Modal` base, use `ui.Pages().AddPage()` to show
- **Async Updates**: Always use `app.UI().QueueUpdateDraw()` from goroutines
- **Key Hints**: Implement `Hints() []ui.KeyHint` on views, update with `app.UI().Menu().SetHints()`

## Build Command
```bash
pushd /Users/atterpac/projects/temportui && go build ./...
```

## Key Files Reference
- `internal/temporal/provider.go` - Provider interface and types
- `internal/temporal/client.go` - SDK implementation
- `internal/ui/` - UI components
- `internal/view/` - Application views
- `internal/config/config.go` - Configuration structs
