# Temporal TUI Mutations - Technical Specification

## 1. Provider Interface Extensions

Add these methods to `internal/temporal/provider.go`:

```go
type Provider interface {
    // ... existing read methods ...

    // Workflow Mutations
    CancelWorkflow(ctx context.Context, namespace, workflowID, runID, reason string) error
    TerminateWorkflow(ctx context.Context, namespace, workflowID, runID, reason string) error
    SignalWorkflow(ctx context.Context, namespace, workflowID, runID, signalName string, input []byte) error
    DeleteWorkflow(ctx context.Context, namespace, workflowID, runID string) error
    StartWorkflow(ctx context.Context, opts StartWorkflowOptions) (*StartWorkflowResult, error)
    ResetWorkflow(ctx context.Context, opts ResetWorkflowOptions) (*ResetWorkflowResult, error)

    // Schedule Mutations (Phase 4)
    ToggleSchedule(ctx context.Context, namespace, scheduleID string, pause bool, reason string) error
    TriggerSchedule(ctx context.Context, namespace, scheduleID string) error
    DeleteSchedule(ctx context.Context, namespace, scheduleID string) error
}

// StartWorkflowOptions contains parameters for starting a workflow
type StartWorkflowOptions struct {
    Namespace        string
    WorkflowID       string            // Optional, auto-generated if empty
    WorkflowType     string
    TaskQueue        string
    Input            []byte            // JSON input
    ExecutionTimeout time.Duration
    RunTimeout       time.Duration
    Memo             map[string]string
    SearchAttributes map[string]interface{}
}

// StartWorkflowResult contains the result of starting a workflow
type StartWorkflowResult struct {
    WorkflowID string
    RunID      string
}

// ResetWorkflowOptions contains parameters for resetting a workflow
type ResetWorkflowOptions struct {
    Namespace   string
    WorkflowID  string
    RunID       string
    EventID     int64  // Reset to specific event
    ResetType   string // "FirstWorkflowTask", "LastWorkflowTask", "LastContinuedAsNew"
    Reason      string
}

// ResetWorkflowResult contains the result of resetting a workflow
type ResetWorkflowResult struct {
    NewRunID string
}
```

## 2. Client Implementation

Add to `internal/temporal/client.go` or create `internal/temporal/mutations.go`:

```go
// CancelWorkflow requests cancellation of a workflow execution.
func (c *Client) CancelWorkflow(ctx context.Context, namespace, workflowID, runID, reason string) error {
    return c.client.CancelWorkflow(ctx, workflowID, runID)
}

// TerminateWorkflow forcefully terminates a workflow execution.
func (c *Client) TerminateWorkflow(ctx context.Context, namespace, workflowID, runID, reason string) error {
    return c.client.TerminateWorkflow(ctx, workflowID, runID, reason)
}

// SignalWorkflow sends a signal to a workflow execution.
func (c *Client) SignalWorkflow(ctx context.Context, namespace, workflowID, runID, signalName string, input []byte) error {
    return c.client.SignalWorkflow(ctx, workflowID, runID, signalName, input)
}

// DeleteWorkflow deletes a workflow execution.
// Note: Uses WorkflowService directly as SDK client doesn't expose this.
func (c *Client) DeleteWorkflow(ctx context.Context, namespace, workflowID, runID string) error {
    _, err := c.client.WorkflowService().DeleteWorkflowExecution(ctx,
        &workflowservice.DeleteWorkflowExecutionRequest{
            Namespace: namespace,
            WorkflowExecution: &commonpb.WorkflowExecution{
                WorkflowId: workflowID,
                RunId:      runID,
            },
        })
    return err
}

// StartWorkflow starts a new workflow execution.
func (c *Client) StartWorkflow(ctx context.Context, opts StartWorkflowOptions) (*StartWorkflowResult, error) {
    workflowOpts := client.StartWorkflowOptions{
        ID:                       opts.WorkflowID,
        TaskQueue:                opts.TaskQueue,
        WorkflowExecutionTimeout: opts.ExecutionTimeout,
        WorkflowRunTimeout:       opts.RunTimeout,
    }

    run, err := c.client.ExecuteWorkflow(ctx, workflowOpts, opts.WorkflowType, opts.Input)
    if err != nil {
        return nil, err
    }

    return &StartWorkflowResult{
        WorkflowID: run.GetID(),
        RunID:      run.GetRunID(),
    }, nil
}

// ResetWorkflow resets a workflow to a previous state.
func (c *Client) ResetWorkflow(ctx context.Context, opts ResetWorkflowOptions) (*ResetWorkflowResult, error) {
    resp, err := c.client.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
        Namespace: opts.Namespace,
        WorkflowExecution: &commonpb.WorkflowExecution{
            WorkflowId: opts.WorkflowID,
            RunId:      opts.RunID,
        },
        Reason:                    opts.Reason,
        WorkflowTaskFinishEventId: opts.EventID,
    })
    if err != nil {
        return nil, err
    }

    return &ResetWorkflowResult{
        NewRunID: resp.GetRunId(),
    }, nil
}
```

## 3. Confirmation Modal Component

Create `internal/ui/confirm.go`:

```go
package ui

import (
    "github.com/gdamore/tcell/v2"
    "github.com/rivo/tview"
)

// ConfirmModal displays a confirmation dialog with command preview.
type ConfirmModal struct {
    *tview.Flex
    title     string
    message   string
    command   string   // CLI equivalent to display
    warning   string   // Optional warning for destructive ops
    onConfirm func()
    onCancel  func()
}

// NewConfirmModal creates a confirmation modal.
func NewConfirmModal(title, message, command string) *ConfirmModal {
    cm := &ConfirmModal{
        Flex:    tview.NewFlex().SetDirection(tview.FlexRow),
        title:   title,
        message: message,
        command: command,
    }
    cm.setup()
    return cm
}

// SetWarning adds a warning message for destructive operations.
func (cm *ConfirmModal) SetWarning(warning string) *ConfirmModal {
    cm.warning = warning
    cm.setup() // Rebuild UI
    return cm
}

// SetOnConfirm sets the confirmation callback.
func (cm *ConfirmModal) SetOnConfirm(fn func()) *ConfirmModal {
    cm.onConfirm = fn
    return cm
}

// SetOnCancel sets the cancel callback.
func (cm *ConfirmModal) SetOnCancel(fn func()) *ConfirmModal {
    cm.onCancel = fn
    return cm
}

func (cm *ConfirmModal) setup() {
    // Build modal content with:
    // - Title
    // - Message
    // - Warning (if set)
    // - Command preview
    // - Keybind hints
}

// InputHandler handles keyboard input.
func (cm *ConfirmModal) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
    return cm.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
        switch event.Key() {
        case tcell.KeyEnter:
            if cm.onConfirm != nil {
                cm.onConfirm()
            }
        case tcell.KeyEscape:
            if cm.onCancel != nil {
                cm.onCancel()
            }
        }
    })
}
```

## 4. Input Modal Component

Create `internal/ui/input_modal.go`:

```go
package ui

// InputModal displays a modal with one or more input fields.
type InputModal struct {
    *tview.Flex
    title    string
    fields   []InputField
    onSubmit func(values map[string]string)
    onCancel func()
}

type InputField struct {
    Name        string
    Label       string
    Placeholder string
    Required    bool
    Validator   func(string) error
}

// NewInputModal creates an input modal with the specified fields.
func NewInputModal(title string, fields []InputField) *InputModal {
    // Implementation
}
```

## 5. View Integration

### Workflow Detail View (`internal/view/workflow_detail.go`)

Add keybinds in `Start()` method:

```go
func (wd *WorkflowDetail) Start() {
    wd.eventTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
        switch event.Rune() {
        case 'r':
            wd.loadData()
            return nil
        case 'c':
            wd.showCancelConfirm()
            return nil
        case 'X':
            wd.showTerminateConfirm()
            return nil
        case 's':
            wd.showSignalInput()
            return nil
        case 'D':
            wd.showDeleteConfirm()
            return nil
        case 'R':
            wd.showResetSelector()
            return nil
        }
        return event
    })
    wd.loadData()
}

func (wd *WorkflowDetail) showCancelConfirm() {
    command := fmt.Sprintf(`temporal workflow cancel \
  --workflow-id %s \
  --run-id %s \
  --namespace %s \
  --reason "Cancelled via TUI"`,
        wd.workflowID, wd.runID, wd.app.CurrentNamespace())

    modal := ui.NewConfirmModal(
        "Cancel Workflow",
        fmt.Sprintf("Cancel workflow %s?", wd.workflowID),
        command,
    ).SetOnConfirm(func() {
        wd.executeCancelWorkflow()
    }).SetOnCancel(func() {
        wd.closeModal()
    })

    wd.app.UI().Pages().AddPage("confirm-cancel", modal, true, true)
}

func (wd *WorkflowDetail) showTerminateConfirm() {
    command := fmt.Sprintf(`temporal workflow terminate \
  --workflow-id %s \
  --run-id %s \
  --namespace %s \
  --reason "Terminated via TUI"`,
        wd.workflowID, wd.runID, wd.app.CurrentNamespace())

    modal := ui.NewConfirmModal(
        "Terminate Workflow",
        fmt.Sprintf("Terminate workflow %s?", wd.workflowID),
        command,
    ).SetWarning("This will forcefully terminate the workflow. No cleanup code will run.").
      SetOnConfirm(func() {
        wd.executeTerminateWorkflow()
    }).SetOnCancel(func() {
        wd.closeModal()
    })

    wd.app.UI().Pages().AddPage("confirm-terminate", modal, true, true)
}

func (wd *WorkflowDetail) executeCancelWorkflow() {
    provider := wd.app.Provider()
    if provider == nil {
        wd.showError(fmt.Errorf("no provider connected"))
        return
    }

    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        err := provider.CancelWorkflow(ctx,
            wd.app.CurrentNamespace(),
            wd.workflowID,
            wd.runID,
            "Cancelled via TUI")

        wd.app.UI().QueueUpdateDraw(func() {
            wd.closeModal()
            if err != nil {
                wd.showError(err)
            } else {
                wd.loadData() // Refresh to show new status
            }
        })
    }()
}
```

### Update Hints

```go
func (wd *WorkflowDetail) Hints() []ui.KeyHint {
    hints := []ui.KeyHint{
        {Key: "r", Description: "Refresh"},
        {Key: "j/k", Description: "Navigate"},
    }

    // Only show mutation hints if workflow is running
    if wd.workflow != nil && wd.workflow.Status == "Running" {
        hints = append(hints,
            ui.KeyHint{Key: "c", Description: "Cancel"},
            ui.KeyHint{Key: "X", Description: "Terminate"},
            ui.KeyHint{Key: "s", Description: "Signal"},
        )
    }

    hints = append(hints,
        ui.KeyHint{Key: "D", Description: "Delete"},
        ui.KeyHint{Key: "T", Description: "Theme"},
        ui.KeyHint{Key: "esc", Description: "Back"},
    )

    return hints
}
```

## 6. Command Bar Enhancement

Extend command bar to handle action commands:

```go
// In commandbar.go or a new command parser

type ActionCommand struct {
    Name   string
    Target string
    Args   map[string]string
}

// ParseActionCommand parses `:command target --arg value` syntax
func ParseActionCommand(input string) (*ActionCommand, error) {
    // Parse commands like:
    // :cancel order-12345
    // :signal order-12345 my-signal {"data": "value"}
    // :terminate order-12345 --reason "manual stop"
}
```

## 7. Temporal CLI Command Reference

For each operation, the equivalent CLI command to show in confirmation:

### Cancel Workflow
```bash
temporal workflow cancel \
  --workflow-id <id> \
  --run-id <run-id> \
  --namespace <namespace> \
  --reason "<reason>"
```

### Terminate Workflow
```bash
temporal workflow terminate \
  --workflow-id <id> \
  --run-id <run-id> \
  --namespace <namespace> \
  --reason "<reason>"
```

### Signal Workflow
```bash
temporal workflow signal \
  --workflow-id <id> \
  --run-id <run-id> \
  --namespace <namespace> \
  --name <signal-name> \
  --input '<json-input>'
```

### Delete Workflow
```bash
temporal workflow delete \
  --workflow-id <id> \
  --run-id <run-id> \
  --namespace <namespace>
```

### Start Workflow
```bash
temporal workflow start \
  --workflow-id <id> \
  --type <workflow-type> \
  --task-queue <queue> \
  --namespace <namespace> \
  --input '<json-input>'
```

### Reset Workflow
```bash
temporal workflow reset \
  --workflow-id <id> \
  --run-id <run-id> \
  --namespace <namespace> \
  --event-id <event-id> \
  --reason "<reason>"
```

## 8. Error Handling

All mutation operations should:

1. Show loading state during execution
2. Handle timeouts gracefully
3. Display user-friendly error messages
4. Allow retry on transient failures

```go
type MutationError struct {
    Operation string
    Target    string
    Err       error
}

func (e *MutationError) Error() string {
    return fmt.Sprintf("failed to %s %s: %v", e.Operation, e.Target, e.Err)
}
```

## 9. Testing Considerations

- Mock provider should implement all mutation methods
- Unit tests for confirmation modal
- Unit tests for command parsing
- Integration tests with local Temporal server (testdata/workflows.go)
