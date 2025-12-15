package temporal

import (
	"go.temporal.io/api/enums/v1"
)

// WorkflowStatus constants match the UI display strings.
const (
	StatusRunning    = "Running"
	StatusCompleted  = "Completed"
	StatusFailed     = "Failed"
	StatusCanceled   = "Canceled"
	StatusTerminated = "Terminated"
	StatusTimedOut   = "TimedOut"
	StatusUnknown    = "Unknown"
)

// MapWorkflowStatus converts a Temporal SDK workflow execution status to a UI-friendly string.
func MapWorkflowStatus(status enums.WorkflowExecutionStatus) string {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return StatusRunning
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return StatusCompleted
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return StatusFailed
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return StatusCanceled
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return StatusTerminated
	case enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return StatusTimedOut
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return StatusCompleted // Treat ContinuedAsNew as completed for display
	default:
		return StatusUnknown
	}
}

// NamespaceState constants.
const (
	NamespaceStateActive     = "Active"
	NamespaceStateDeprecated = "Deprecated"
	NamespaceStateDeleted    = "Deleted"
	NamespaceStateUnknown    = "Unknown"
)

// MapNamespaceState converts a Temporal SDK namespace state to a UI-friendly string.
func MapNamespaceState(state enums.NamespaceState) string {
	switch state {
	case enums.NAMESPACE_STATE_REGISTERED:
		return NamespaceStateActive
	case enums.NAMESPACE_STATE_DEPRECATED:
		return NamespaceStateDeprecated
	case enums.NAMESPACE_STATE_DELETED:
		return NamespaceStateDeleted
	default:
		return NamespaceStateUnknown
	}
}

// TaskQueueType constants.
const (
	TaskQueueTypeWorkflow = "Workflow"
	TaskQueueTypeActivity = "Activity"
)

// MapTaskQueueType converts a Temporal SDK task queue type to a UI-friendly string.
func MapTaskQueueType(tqType enums.TaskQueueType) string {
	switch tqType {
	case enums.TASK_QUEUE_TYPE_WORKFLOW:
		return TaskQueueTypeWorkflow
	case enums.TASK_QUEUE_TYPE_ACTIVITY:
		return TaskQueueTypeActivity
	default:
		return "Unknown"
	}
}
