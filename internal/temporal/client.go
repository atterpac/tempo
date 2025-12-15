package temporal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Client implements the Provider interface using the Temporal SDK.
type Client struct {
	client    client.Client
	config    ConnectionConfig
	connected bool
	mu        sync.RWMutex
}

// NewClient creates a new Temporal SDK client with the given configuration.
func NewClient(ctx context.Context, config ConnectionConfig) (*Client, error) {
	opts := client.Options{
		HostPort:  config.Address,
		Namespace: config.Namespace,
	}

	// Configure TLS if any TLS options are provided
	if config.TLSCertPath != "" || config.TLSCAPath != "" || config.TLSSkipVerify {
		tlsConfig, err := buildTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts.ConnectionOptions.TLS = tlsConfig
	}

	c, err := client.DialContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Temporal server: %w", err)
	}

	return &Client{
		client:    c,
		config:    config,
		connected: true,
	}, nil
}

// buildTLSConfig creates a TLS configuration from the connection config.
func buildTLSConfig(config ConnectionConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSSkipVerify,
	}

	if config.TLSServerName != "" {
		tlsConfig.ServerName = config.TLSServerName
	}

	// Load client certificate if provided
	if config.TLSCertPath != "" && config.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCertPath, config.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if config.TLSCAPath != "" {
		caCert, err := os.ReadFile(config.TLSCAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// Close releases the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	if c.client != nil {
		c.client.Close()
	}
	return nil
}

// IsConnected returns true if the client has an active connection.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// CheckConnection verifies the connection is still alive by making a lightweight API call.
func (c *Client) CheckConnection(ctx context.Context) error {
	c.mu.RLock()
	cl := c.client
	c.mu.RUnlock()

	if cl == nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("client is nil")
	}

	// Make a lightweight API call to check connection
	// ListNamespaces with PageSize 1 is a good health check
	_, err := cl.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{
		PageSize: 1,
	})
	if err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("connection check failed: %w", err)
	}

	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

// Reconnect attempts to re-establish a connection to the Temporal server.
func (c *Client) Reconnect(ctx context.Context) error {
	c.mu.Lock()
	// Close existing client if any
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
	c.connected = false
	config := c.config
	c.mu.Unlock()

	opts := client.Options{
		HostPort:  config.Address,
		Namespace: config.Namespace,
	}

	// Configure TLS if any TLS options are provided
	if config.TLSCertPath != "" || config.TLSCAPath != "" || config.TLSSkipVerify {
		tlsConfig, err := buildTLSConfig(config)
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts.ConnectionOptions.TLS = tlsConfig
	}

	newClient, err := client.DialContext(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	c.mu.Lock()
	c.client = newClient
	c.connected = true
	c.mu.Unlock()

	return nil
}

// Config returns the connection configuration used by this client.
func (c *Client) Config() ConnectionConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// ListNamespaces returns all namespaces visible to the client.
func (c *Client) ListNamespaces(ctx context.Context) ([]Namespace, error) {
	var namespaces []Namespace
	var nextPageToken []byte

	for {
		resp, err := c.client.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{
			PageSize:      100,
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}

		for _, ns := range resp.GetNamespaces() {
			info := ns.GetNamespaceInfo()
			config := ns.GetConfig()

			retention := "N/A"
			if config.GetWorkflowExecutionRetentionTtl() != nil {
				retention = formatDuration(config.GetWorkflowExecutionRetentionTtl())
			}

			namespaces = append(namespaces, Namespace{
				Name:            info.GetName(),
				State:           MapNamespaceState(info.GetState()),
				RetentionPeriod: retention,
				Description:     info.GetDescription(),
				OwnerEmail:      info.GetOwnerEmail(),
			})
		}

		nextPageToken = resp.GetNextPageToken()
		if len(nextPageToken) == 0 {
			break
		}
	}

	return namespaces, nil
}

// ListWorkflows returns workflows for a namespace with optional filtering.
func (c *Client) ListWorkflows(ctx context.Context, namespace string, opts ListOptions) ([]Workflow, string, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	req := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     namespace,
		PageSize:      int32(pageSize),
		NextPageToken: []byte(opts.PageToken),
	}

	if opts.Query != "" {
		req.Query = opts.Query
	}

	resp, err := c.client.WorkflowService().ListWorkflowExecutions(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list workflows: %w", err)
	}

	var workflows []Workflow
	for _, exec := range resp.GetExecutions() {
		wf := Workflow{
			ID:        exec.GetExecution().GetWorkflowId(),
			RunID:     exec.GetExecution().GetRunId(),
			Type:      exec.GetType().GetName(),
			Status:    MapWorkflowStatus(exec.GetStatus()),
			Namespace: namespace,
			TaskQueue: exec.GetTaskQueue(),
			StartTime: exec.GetStartTime().AsTime(),
		}

		if exec.GetCloseTime() != nil && !exec.GetCloseTime().AsTime().IsZero() {
			t := exec.GetCloseTime().AsTime()
			wf.EndTime = &t
		}

		if exec.GetParentExecution() != nil && exec.GetParentExecution().GetWorkflowId() != "" {
			parentID := exec.GetParentExecution().GetWorkflowId()
			wf.ParentID = &parentID
		}

		// Extract memo if present
		if exec.GetMemo() != nil && exec.GetMemo().GetFields() != nil {
			wf.Memo = make(map[string]string)
			for k, v := range exec.GetMemo().GetFields() {
				// Try to extract string value from payload
				if v != nil && v.GetData() != nil {
					var strVal string
					if err := json.Unmarshal(v.GetData(), &strVal); err == nil {
						wf.Memo[k] = strVal
					} else {
						wf.Memo[k] = string(v.GetData())
					}
				}
			}
		}

		workflows = append(workflows, wf)
	}

	return workflows, string(resp.GetNextPageToken()), nil
}

// GetWorkflow returns details for a specific workflow execution.
func (c *Client) GetWorkflow(ctx context.Context, namespace, workflowID, runID string) (*Workflow, error) {
	resp, err := c.client.WorkflowService().DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
		Namespace: namespace,
		Execution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe workflow: %w", err)
	}

	info := resp.GetWorkflowExecutionInfo()
	wf := &Workflow{
		ID:        info.GetExecution().GetWorkflowId(),
		RunID:     info.GetExecution().GetRunId(),
		Type:      info.GetType().GetName(),
		Status:    MapWorkflowStatus(info.GetStatus()),
		Namespace: namespace,
		TaskQueue: info.GetTaskQueue(),
		StartTime: info.GetStartTime().AsTime(),
	}

	if info.GetCloseTime() != nil && !info.GetCloseTime().AsTime().IsZero() {
		t := info.GetCloseTime().AsTime()
		wf.EndTime = &t
	}

	if info.GetParentExecution() != nil && info.GetParentExecution().GetWorkflowId() != "" {
		parentID := info.GetParentExecution().GetWorkflowId()
		wf.ParentID = &parentID
	}

	return wf, nil
}

// GetWorkflowHistory returns the event history for a workflow execution.
func (c *Client) GetWorkflowHistory(ctx context.Context, namespace, workflowID, runID string) ([]HistoryEvent, error) {
	var events []HistoryEvent
	var nextPageToken []byte

	for {
		resp, err := c.client.WorkflowService().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
			Namespace: namespace,
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: workflowID,
				RunId:      runID,
			},
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow history: %w", err)
		}

		for _, event := range resp.GetHistory().GetEvents() {
			he := HistoryEvent{
				ID:      event.GetEventId(),
				Type:    formatEventType(event.GetEventType().String()),
				Time:    event.GetEventTime().AsTime(),
				Details: extractEventDetails(event),
			}
			events = append(events, he)
		}

		nextPageToken = resp.GetNextPageToken()
		if len(nextPageToken) == 0 {
			break
		}
	}

	return events, nil
}

// formatEventType cleans up the event type string for display
func formatEventType(eventType string) string {
	// Remove EVENT_TYPE_ prefix if present (older protobuf format)
	eventType = strings.TrimPrefix(eventType, "EVENT_TYPE_")

	// If it contains underscores, convert from SCREAMING_SNAKE_CASE to PascalCase
	if strings.Contains(eventType, "_") {
		parts := strings.Split(strings.ToLower(eventType), "_")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, "")
	}

	// Otherwise it's already in a readable format (e.g., WorkflowExecutionStarted)
	return eventType
}

// extractEventDetails extracts a verbose summary string from a history event.
func extractEventDetails(event *historypb.HistoryEvent) string {
	var details []string

	switch event.GetEventType() {
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		attrs := event.GetWorkflowExecutionStartedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowType() != nil {
				details = append(details, fmt.Sprintf("WorkflowType: %s", attrs.GetWorkflowType().GetName()))
			}
			if attrs.GetTaskQueue() != nil {
				details = append(details, fmt.Sprintf("TaskQueue: %s", attrs.GetTaskQueue().GetName()))
			}
			if attrs.GetInput() != nil {
				details = append(details, fmt.Sprintf("Input: %s", formatPayloads(attrs.GetInput())))
			}
			if attrs.GetWorkflowExecutionTimeout() != nil {
				details = append(details, fmt.Sprintf("ExecutionTimeout: %s", attrs.GetWorkflowExecutionTimeout().AsDuration()))
			}
			if attrs.GetWorkflowRunTimeout() != nil {
				details = append(details, fmt.Sprintf("RunTimeout: %s", attrs.GetWorkflowRunTimeout().AsDuration()))
			}
			if attrs.GetWorkflowTaskTimeout() != nil {
				details = append(details, fmt.Sprintf("TaskTimeout: %s", attrs.GetWorkflowTaskTimeout().AsDuration()))
			}
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
			if attrs.GetAttempt() > 1 {
				details = append(details, fmt.Sprintf("Attempt: %d", attrs.GetAttempt()))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		attrs := event.GetWorkflowExecutionCompletedEventAttributes()
		if attrs != nil {
			if attrs.GetResult() != nil {
				details = append(details, fmt.Sprintf("Result: %s", formatPayloads(attrs.GetResult())))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetWorkflowExecutionFailedEventAttributes()
		if attrs != nil {
			if attrs.GetFailure() != nil {
				details = append(details, fmt.Sprintf("Failure: %s", attrs.GetFailure().GetMessage()))
				if attrs.GetFailure().GetStackTrace() != "" {
					// Truncate stack trace for display
					trace := attrs.GetFailure().GetStackTrace()
					if len(trace) > 200 {
						trace = trace[:200] + "..."
					}
					details = append(details, fmt.Sprintf("StackTrace: %s", trace))
				}
			}
			details = append(details, fmt.Sprintf("RetryState: %s", attrs.GetRetryState().String()))
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		attrs := event.GetWorkflowExecutionTimedOutEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("RetryState: %s", attrs.GetRetryState().String()))
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		attrs := event.GetWorkflowExecutionCanceledEventAttributes()
		if attrs != nil {
			if attrs.GetDetails() != nil {
				details = append(details, fmt.Sprintf("Details: %s", formatPayloads(attrs.GetDetails())))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		attrs := event.GetWorkflowExecutionTerminatedEventAttributes()
		if attrs != nil {
			if attrs.GetReason() != "" {
				details = append(details, fmt.Sprintf("Reason: %s", attrs.GetReason()))
			}
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED:
		attrs := event.GetWorkflowTaskScheduledEventAttributes()
		if attrs != nil {
			if attrs.GetTaskQueue() != nil {
				details = append(details, fmt.Sprintf("TaskQueue: %s", attrs.GetTaskQueue().GetName()))
			}
			if attrs.GetStartToCloseTimeout() != nil {
				details = append(details, fmt.Sprintf("StartToCloseTimeout: %s", attrs.GetStartToCloseTimeout().AsDuration()))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_TASK_STARTED:
		attrs := event.GetWorkflowTaskStartedEventAttributes()
		if attrs != nil {
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
		}

	case enums.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
		attrs := event.GetWorkflowTaskCompletedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_TASK_TIMED_OUT:
		attrs := event.GetWorkflowTaskTimedOutEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			details = append(details, fmt.Sprintf("TimeoutType: %s", attrs.GetTimeoutType().String()))
		}

	case enums.EVENT_TYPE_WORKFLOW_TASK_FAILED:
		attrs := event.GetWorkflowTaskFailedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("Cause: %s", attrs.GetCause().String()))
			if attrs.GetFailure() != nil {
				details = append(details, fmt.Sprintf("Failure: %s", attrs.GetFailure().GetMessage()))
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		attrs := event.GetActivityTaskScheduledEventAttributes()
		if attrs != nil {
			if attrs.GetActivityType() != nil {
				details = append(details, fmt.Sprintf("ActivityType: %s", attrs.GetActivityType().GetName()))
			}
			if attrs.GetActivityId() != "" {
				details = append(details, fmt.Sprintf("ActivityId: %s", attrs.GetActivityId()))
			}
			if attrs.GetTaskQueue() != nil {
				details = append(details, fmt.Sprintf("TaskQueue: %s", attrs.GetTaskQueue().GetName()))
			}
			if attrs.GetInput() != nil {
				details = append(details, fmt.Sprintf("Input: %s", formatPayloads(attrs.GetInput())))
			}
			if attrs.GetScheduleToCloseTimeout() != nil {
				details = append(details, fmt.Sprintf("ScheduleToCloseTimeout: %s", attrs.GetScheduleToCloseTimeout().AsDuration()))
			}
			if attrs.GetScheduleToStartTimeout() != nil {
				details = append(details, fmt.Sprintf("ScheduleToStartTimeout: %s", attrs.GetScheduleToStartTimeout().AsDuration()))
			}
			if attrs.GetStartToCloseTimeout() != nil {
				details = append(details, fmt.Sprintf("StartToCloseTimeout: %s", attrs.GetStartToCloseTimeout().AsDuration()))
			}
			if attrs.GetRetryPolicy() != nil {
				rp := attrs.GetRetryPolicy()
				details = append(details, fmt.Sprintf("RetryPolicy: MaxAttempts=%d", rp.GetMaximumAttempts()))
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		attrs := event.GetActivityTaskStartedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("Attempt: %d", attrs.GetAttempt()))
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		attrs := event.GetActivityTaskCompletedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			if attrs.GetResult() != nil {
				details = append(details, fmt.Sprintf("Result: %s", formatPayloads(attrs.GetResult())))
			}
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		attrs := event.GetActivityTaskFailedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			if attrs.GetFailure() != nil {
				details = append(details, fmt.Sprintf("Failure: %s", attrs.GetFailure().GetMessage()))
			}
			details = append(details, fmt.Sprintf("RetryState: %s", attrs.GetRetryState().String()))
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		attrs := event.GetActivityTaskTimedOutEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			if attrs.GetFailure() != nil {
				details = append(details, fmt.Sprintf("TimeoutType: %s", attrs.GetFailure().GetMessage()))
			}
			details = append(details, fmt.Sprintf("RetryState: %s", attrs.GetRetryState().String()))
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED:
		attrs := event.GetActivityTaskCancelRequestedEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		attrs := event.GetActivityTaskCanceledEventAttributes()
		if attrs != nil {
			details = append(details, fmt.Sprintf("ScheduledEventId: %d", attrs.GetScheduledEventId()))
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
			if attrs.GetDetails() != nil {
				details = append(details, fmt.Sprintf("Details: %s", formatPayloads(attrs.GetDetails())))
			}
		}

	case enums.EVENT_TYPE_TIMER_STARTED:
		attrs := event.GetTimerStartedEventAttributes()
		if attrs != nil {
			if attrs.GetTimerId() != "" {
				details = append(details, fmt.Sprintf("TimerId: %s", attrs.GetTimerId()))
			}
			if attrs.GetStartToFireTimeout() != nil {
				details = append(details, fmt.Sprintf("StartToFireTimeout: %s", attrs.GetStartToFireTimeout().AsDuration()))
			}
		}

	case enums.EVENT_TYPE_TIMER_FIRED:
		attrs := event.GetTimerFiredEventAttributes()
		if attrs != nil {
			if attrs.GetTimerId() != "" {
				details = append(details, fmt.Sprintf("TimerId: %s", attrs.GetTimerId()))
			}
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
		}

	case enums.EVENT_TYPE_TIMER_CANCELED:
		attrs := event.GetTimerCanceledEventAttributes()
		if attrs != nil {
			if attrs.GetTimerId() != "" {
				details = append(details, fmt.Sprintf("TimerId: %s", attrs.GetTimerId()))
			}
			details = append(details, fmt.Sprintf("StartedEventId: %d", attrs.GetStartedEventId()))
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED:
		attrs := event.GetWorkflowExecutionSignaledEventAttributes()
		if attrs != nil {
			if attrs.GetSignalName() != "" {
				details = append(details, fmt.Sprintf("SignalName: %s", attrs.GetSignalName()))
			}
			if attrs.GetInput() != nil {
				details = append(details, fmt.Sprintf("Input: %s", formatPayloads(attrs.GetInput())))
			}
			if attrs.GetIdentity() != "" {
				details = append(details, fmt.Sprintf("Identity: %s", attrs.GetIdentity()))
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED:
		attrs := event.GetWorkflowExecutionUpdateAcceptedEventAttributes()
		if attrs != nil {
			if attrs.GetAcceptedRequest() != nil {
				if attrs.GetAcceptedRequest().GetMeta() != nil {
					details = append(details, fmt.Sprintf("UpdateId: %s", attrs.GetAcceptedRequest().GetMeta().GetUpdateId()))
				}
			}
		}

	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_COMPLETED:
		attrs := event.GetWorkflowExecutionUpdateCompletedEventAttributes()
		if attrs != nil {
			if attrs.GetMeta() != nil {
				details = append(details, fmt.Sprintf("UpdateId: %s", attrs.GetMeta().GetUpdateId()))
			}
		}

	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED:
		attrs := event.GetStartChildWorkflowExecutionInitiatedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowType() != nil {
				details = append(details, fmt.Sprintf("WorkflowType: %s", attrs.GetWorkflowType().GetName()))
			}
			if attrs.GetWorkflowId() != "" {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowId()))
			}
			if attrs.GetTaskQueue() != nil {
				details = append(details, fmt.Sprintf("TaskQueue: %s", attrs.GetTaskQueue().GetName()))
			}
			if attrs.GetInput() != nil {
				details = append(details, fmt.Sprintf("Input: %s", formatPayloads(attrs.GetInput())))
			}
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED:
		attrs := event.GetChildWorkflowExecutionStartedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowType() != nil {
				details = append(details, fmt.Sprintf("WorkflowType: %s", attrs.GetWorkflowType().GetName()))
			}
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
				details = append(details, fmt.Sprintf("RunId: %s", attrs.GetWorkflowExecution().GetRunId()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED:
		attrs := event.GetChildWorkflowExecutionCompletedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			if attrs.GetResult() != nil {
				details = append(details, fmt.Sprintf("Result: %s", formatPayloads(attrs.GetResult())))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetChildWorkflowExecutionFailedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			if attrs.GetFailure() != nil {
				details = append(details, fmt.Sprintf("Failure: %s", attrs.GetFailure().GetMessage()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:
		attrs := event.GetChildWorkflowExecutionCanceledEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT:
		attrs := event.GetChildWorkflowExecutionTimedOutEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED:
		attrs := event.GetChildWorkflowExecutionTerminatedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_MARKER_RECORDED:
		attrs := event.GetMarkerRecordedEventAttributes()
		if attrs != nil {
			if attrs.GetMarkerName() != "" {
				details = append(details, fmt.Sprintf("MarkerName: %s", attrs.GetMarkerName()))
			}
		}

	case enums.EVENT_TYPE_EXTERNAL_WORKFLOW_EXECUTION_SIGNALED:
		attrs := event.GetExternalWorkflowExecutionSignaledEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			details = append(details, fmt.Sprintf("InitiatedEventId: %d", attrs.GetInitiatedEventId()))
		}

	case enums.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED:
		attrs := event.GetSignalExternalWorkflowExecutionInitiatedEventAttributes()
		if attrs != nil {
			if attrs.GetWorkflowExecution() != nil {
				details = append(details, fmt.Sprintf("WorkflowId: %s", attrs.GetWorkflowExecution().GetWorkflowId()))
			}
			if attrs.GetSignalName() != "" {
				details = append(details, fmt.Sprintf("SignalName: %s", attrs.GetSignalName()))
			}
			if attrs.GetInput() != nil {
				details = append(details, fmt.Sprintf("Input: %s", formatPayloads(attrs.GetInput())))
			}
		}

	default:
		// For unhandled event types, return event type name
		details = append(details, fmt.Sprintf("EventType: %s", event.GetEventType().String()))
	}

	return strings.Join(details, ", ")
}

// formatPayloads formats payloads for display
func formatPayloads(payloads *commonpb.Payloads) string {
	if payloads == nil {
		return ""
	}

	var results []string
	for _, p := range payloads.GetPayloads() {
		if p == nil {
			continue
		}
		data := p.GetData()
		if len(data) == 0 {
			continue
		}

		// Try to parse as JSON for nicer display
		var jsonVal interface{}
		if err := json.Unmarshal(data, &jsonVal); err == nil {
			// Format as compact JSON
			if b, err := json.Marshal(jsonVal); err == nil {
				results = append(results, string(b))
				continue
			}
		}

		// Fall back to raw string (truncated)
		s := string(data)
		if len(s) > 100 {
			s = s[:100] + "..."
		}
		results = append(results, s)
	}

	return strings.Join(results, ", ")
}

// DescribeTaskQueue returns task queue info and active pollers.
func (c *Client) DescribeTaskQueue(ctx context.Context, namespace, taskQueue string) (*TaskQueueInfo, []Poller, error) {
	// Query workflow task queue
	wfResp, err := c.client.WorkflowService().DescribeTaskQueue(ctx, &workflowservice.DescribeTaskQueueRequest{
		Namespace: namespace,
		TaskQueue: &taskqueue.TaskQueue{
			Name: taskQueue,
			Kind: enums.TASK_QUEUE_KIND_NORMAL,
		},
		TaskQueueType: enums.TASK_QUEUE_TYPE_WORKFLOW,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe workflow task queue: %w", err)
	}

	// Query activity task queue
	actResp, err := c.client.WorkflowService().DescribeTaskQueue(ctx, &workflowservice.DescribeTaskQueueRequest{
		Namespace: namespace,
		TaskQueue: &taskqueue.TaskQueue{
			Name: taskQueue,
			Kind: enums.TASK_QUEUE_KIND_NORMAL,
		},
		TaskQueueType: enums.TASK_QUEUE_TYPE_ACTIVITY,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to describe activity task queue: %w", err)
	}

	// Combine poller info
	var pollers []Poller

	for _, p := range wfResp.GetPollers() {
		pollers = append(pollers, Poller{
			Identity:       p.GetIdentity(),
			LastAccessTime: p.GetLastAccessTime().AsTime(),
			TaskQueueType:  TaskQueueTypeWorkflow,
			RatePerSecond:  p.GetRatePerSecond(),
		})
	}

	for _, p := range actResp.GetPollers() {
		pollers = append(pollers, Poller{
			Identity:       p.GetIdentity(),
			LastAccessTime: p.GetLastAccessTime().AsTime(),
			TaskQueueType:  TaskQueueTypeActivity,
			RatePerSecond:  p.GetRatePerSecond(),
		})
	}

	info := &TaskQueueInfo{
		Name:        taskQueue,
		Type:        "Combined",
		PollerCount: len(pollers),
		Backlog:     0, // Backlog info requires enhanced visibility or approximation
	}

	return info, pollers, nil
}

// formatDuration formats a protobuf duration as a human-readable string.
func formatDuration(d *durationpb.Duration) string {
	if d == nil {
		return "N/A"
	}

	dur := d.AsDuration()

	if dur < time.Hour {
		return fmt.Sprintf("%d minutes", int(dur.Minutes()))
	}
	if dur < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(dur.Hours()))
	}

	days := int(dur.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// Ensure Client implements Provider
var _ Provider = (*Client)(nil)
