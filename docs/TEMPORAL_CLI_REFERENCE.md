# Temporal CLI Write Operations Reference

Complete reference for Temporal CLI write/mutation operations relevant to TUI implementation.

## Workflow Operations

### Cancel Workflow

Gracefully cancels a workflow, allowing it to perform cleanup.

```bash
temporal workflow cancel \
  --workflow-id <id> \
  [--run-id <run-id>] \
  [--namespace <namespace>] \
  [--reason <reason>]
```

**Behavior:**
- Records `WorkflowExecutionCancelRequested` event
- Workflow can handle cancellation and perform cleanup
- Does not force-stop the workflow

**SDK Method:** `client.CancelWorkflow(ctx, workflowID, runID)`

---

### Terminate Workflow

Forcefully terminates a workflow immediately.

```bash
temporal workflow terminate \
  --workflow-id <id> \
  [--run-id <run-id>] \
  [--namespace <namespace>] \
  [--reason <reason>]
```

**Behavior:**
- Records `WorkflowExecutionTerminated` as final event
- No cleanup opportunity
- Immediate stop

**SDK Method:** `client.TerminateWorkflow(ctx, workflowID, runID, reason)`

---

### Signal Workflow

Sends a signal to a running workflow.

```bash
temporal workflow signal \
  --workflow-id <id> \
  --name <signal-name> \
  [--run-id <run-id>] \
  [--namespace <namespace>] \
  [--input '<json>']
```

**Parameters:**
- `--name` - Signal handler name (required)
- `--input` - JSON payload for signal

**SDK Method:** `client.SignalWorkflow(ctx, workflowID, runID, signalName, input)`

---

### Delete Workflow

Permanently deletes a workflow and its history.

```bash
temporal workflow delete \
  --workflow-id <id> \
  [--run-id <run-id>] \
  [--namespace <namespace>]
```

**Behavior:**
- Asynchronously removes workflow and event history
- Cannot be undone
- Works on completed/failed workflows

**SDK Method:** `client.WorkflowService().DeleteWorkflowExecution(ctx, req)`

---

### Start Workflow

Starts a new workflow execution.

```bash
temporal workflow start \
  --type <workflow-type> \
  --task-queue <queue> \
  [--workflow-id <id>] \
  [--namespace <namespace>] \
  [--input '<json>'] \
  [--execution-timeout <duration>] \
  [--run-timeout <duration>] \
  [--memo <key=value>] \
  [--search-attribute <key=value>]
```

**Parameters:**
- `--type` - Workflow type name (required)
- `--task-queue` - Task queue name (required)
- `--workflow-id` - Custom ID (auto-generated if not provided)
- `--input` - JSON input data
- `--execution-timeout` - Max total duration (e.g., "3600s")
- `--run-timeout` - Max single run duration

**SDK Method:** `client.ExecuteWorkflow(ctx, options, workflowType, args...)`

---

### Reset Workflow

Resets a workflow to a previous point in history.

```bash
temporal workflow reset \
  --workflow-id <id> \
  (--event-id <id> | --type <reset-type>) \
  [--run-id <run-id>] \
  [--namespace <namespace>] \
  [--reason <reason>]
```

**Reset Types:**
- `FirstWorkflowTask` - Reset to first workflow task
- `LastWorkflowTask` - Reset to last workflow task
- `LastContinuedAsNew` - Reset to last continue-as-new point

**Behavior:**
- Creates new run from reset point
- Preserves history up to reset point
- Returns new run ID

**SDK Method:** `client.ResetWorkflowExecution(ctx, req)`

---

### Update Workflow

Sends a synchronous update to a running workflow.

```bash
temporal workflow update \
  --workflow-id <id> \
  --update-name <name> \
  [--run-id <run-id>] \
  [--namespace <namespace>] \
  [--input '<json>']
```

**Behavior:**
- Calls update handler in workflow
- Waits for completion
- Returns update result

**SDK Method:** `client.UpdateWorkflow(ctx, options)`

---

## Schedule Operations

### Create Schedule

```bash
temporal schedule create \
  --schedule-id <id> \
  --workflow-type <type> \
  --task-queue <queue> \
  (--calendar '<json>' | --interval <duration> | --cron '<cron>') \
  [--namespace <namespace>] \
  [--workflow-id <base-id>] \
  [--input '<json>'] \
  [--overlap-policy <policy>] \
  [--paused]
```

**Overlap Policies:**
- `Skip` - Skip if previous still running (default)
- `BufferOne` - Buffer one execution
- `BufferAll` - Buffer all
- `CancelOther` - Cancel previous
- `TerminateOther` - Terminate previous
- `AllowAll` - Allow concurrent

---

### Toggle Schedule (Pause/Unpause)

```bash
temporal schedule toggle \
  --schedule-id <id> \
  (--pause | --unpause) \
  [--namespace <namespace>] \
  [--reason <reason>]
```

---

### Trigger Schedule

```bash
temporal schedule trigger \
  --schedule-id <id> \
  [--namespace <namespace>] \
  [--overlap-policy <policy>]
```

**Behavior:**
- Immediately executes scheduled workflow
- Ignores timing rules
- Respects overlap policy

---

### Delete Schedule

```bash
temporal schedule delete \
  --schedule-id <id> \
  [--namespace <namespace>]
```

**Behavior:**
- Removes schedule
- Does not affect running workflows

---

## Batch Operations

### Batch Cancel

```bash
temporal workflow cancel \
  --query '<visibility-query>' \
  [--namespace <namespace>] \
  [--reason <reason>] \
  [--rps <rate>]
```

### Batch Terminate

```bash
temporal workflow terminate \
  --query '<visibility-query>' \
  [--namespace <namespace>] \
  [--reason <reason>] \
  [--rps <rate>]
```

### Batch Signal

```bash
temporal workflow signal \
  --query '<visibility-query>' \
  --name <signal-name> \
  [--namespace <namespace>] \
  [--input '<json>'] \
  [--reason <reason>] \
  [--rps <rate>]
```

### Batch Reset

```bash
temporal workflow reset-batch \
  --query '<visibility-query>' \
  --type <reset-type> \
  [--namespace <namespace>] \
  [--reason <reason>] \
  [--rps <rate>] \
  [--dry-run]
```

---

## Namespace Operations

### Create Namespace

```bash
temporal operator namespace create \
  --namespace <name> \
  [--description <description>] \
  [--email <owner-email>] \
  [--retention <duration>] \
  [--history-archival-state <enabled|disabled>] \
  [--visibility-archival-state <enabled|disabled>]
```

### Update Namespace

```bash
temporal operator namespace update \
  --namespace <name> \
  [--description <description>] \
  [--email <owner-email>] \
  [--retention <duration>]
```

### Delete Namespace

```bash
temporal operator namespace delete \
  --namespace <name> \
  [--yes]
```

---

## Common Parameters

### Connection
- `--address` - Server address (default: localhost:7233)
- `--namespace` - Target namespace (default: default)
- `--api-key` - API key for auth

### TLS
- `--tls` - Enable TLS
- `--tls-cert-path` - Client certificate path
- `--tls-key-path` - Client key path
- `--tls-ca-path` - CA certificate path
- `--tls-server-name` - Server name override
- `--tls-disable-host-verification` - Skip verification

### Output
- `--output` - Format: table, json, card
- `--fields` - Custom fields
- `--time-format` - Time display format

---

## Visibility Query Examples

Used for batch operations and filtering:

```sql
-- By workflow type
WorkflowType = 'OrderWorkflow'

-- By status
ExecutionStatus = 'Running'
ExecutionStatus = 'Failed'

-- By time
StartTime > '2024-01-01T00:00:00Z'
CloseTime < '2024-01-15T00:00:00Z'

-- Combined
WorkflowType = 'OrderWorkflow' AND ExecutionStatus = 'Running'

-- By custom search attribute
CustomAttribute = 'value'
```
