package temporal

import (
	"context"
	"time"
)

// Provider defines the interface for Temporal data access.
// This abstraction allows for different implementations (real SDK, mock, etc.)
type Provider interface {
	// ListNamespaces returns all namespaces visible to the client.
	ListNamespaces(ctx context.Context) ([]Namespace, error)

	// ListWorkflows returns workflows for a namespace with optional filtering.
	ListWorkflows(ctx context.Context, namespace string, opts ListOptions) ([]Workflow, string, error)

	// GetWorkflow returns details for a specific workflow execution.
	GetWorkflow(ctx context.Context, namespace, workflowID, runID string) (*Workflow, error)

	// GetWorkflowHistory returns the event history for a workflow execution.
	GetWorkflowHistory(ctx context.Context, namespace, workflowID, runID string) ([]HistoryEvent, error)

	// DescribeTaskQueue returns task queue info and active pollers.
	DescribeTaskQueue(ctx context.Context, namespace, taskQueue string) (*TaskQueueInfo, []Poller, error)

	// Close releases any resources held by the provider.
	Close() error

	// IsConnected returns true if the provider has an active connection.
	IsConnected() bool

	// CheckConnection verifies the connection is still alive by making a lightweight API call.
	CheckConnection(ctx context.Context) error

	// Reconnect attempts to re-establish a connection to the Temporal server.
	// Returns an error if reconnection fails.
	Reconnect(ctx context.Context) error

	// Config returns the connection configuration used by this provider.
	Config() ConnectionConfig
}

// ListOptions configures workflow list queries.
type ListOptions struct {
	PageSize  int
	PageToken string
	Query     string // Visibility query (e.g., "WorkflowType='OrderWorkflow'")
}

// Namespace represents a Temporal namespace.
type Namespace struct {
	Name            string
	State           string
	RetentionPeriod string
	Description     string
	OwnerEmail      string
}

// Workflow represents a workflow execution.
type Workflow struct {
	ID        string
	RunID     string
	Type      string
	Status    string // "Running", "Completed", "Failed", "Canceled", "Terminated", "TimedOut"
	Namespace string
	TaskQueue string
	StartTime time.Time
	EndTime   *time.Time
	ParentID  *string
	Memo      map[string]string
}

// HistoryEvent represents a workflow history event.
type HistoryEvent struct {
	ID      int64
	Type    string
	Time    time.Time
	Details string
}

// TaskQueueInfo represents task queue status information.
type TaskQueueInfo struct {
	Name        string
	Type        string // "Workflow" or "Activity"
	PollerCount int
	Backlog     int
}

// Poller represents a worker polling a task queue.
type Poller struct {
	Identity       string
	LastAccessTime time.Time
	TaskQueueType  string // "Workflow" or "Activity"
	RatePerSecond  float64
}

// ConnectionConfig holds Temporal server connection settings.
type ConnectionConfig struct {
	Address       string
	Namespace     string
	TLSCertPath   string
	TLSKeyPath    string
	TLSCAPath     string
	TLSServerName string
	TLSSkipVerify bool
}

// DefaultConnectionConfig returns default connection settings.
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		Address:   "localhost:7233",
		Namespace: "default",
	}
}
