# Feature Ideas

Potential features to expand the Temporal TUI.

## Workflow Operations

- [ ] **Query workflows** - Execute queries and display results
- [ ] **Batch operations** - Cancel/terminate multiple workflows with multi-select
- [ ] **Workflow diff** - Compare two workflow executions side-by-side
- [ ] **Update workflow** - Send workflow updates (newer Temporal feature)
- [ ] **Retry failed activities** - Reset to last failed activity directly

## Search & Filtering

- [ ] **Advanced query builder** - Build Temporal visibility queries with UI assistance
- [ ] **Saved filters/queries** - Store and recall frequently used search criteria
- [ ] **Search history** - Navigate through previous searches
- [ ] **Date range picker** - Filter workflows by start/end time ranges

## Monitoring & Observability

- [ ] **Worker health view** - Monitor worker processes, versions, sticky cache stats
- [ ] **Metrics dashboard** - Workflow throughput, latency percentiles, failure rates
- [ ] **Pending activities view** - Activities stuck in retry or heartbeat timeout
- [ ] **Search attributes view** - List and inspect custom search attributes

## Event History Enhancements

- [x] **Event graph/tree view** - Collapsible tree of workflow execution
- [x] **Timeline/Gantt view** - Visual timeline showing duration and parallelism
- [ ] **Event filtering** - Filter events by type (e.g., only ActivityTask events)
- [ ] **Event diff** - Compare events between two runs
- [ ] **JSON viewer with folding** - Collapsible JSON for large payloads
- [ ] **Payload decoding** - Base64/protobuf decode and display

## Schedule Features

- [ ] **Create schedule** - Form to create new schedules
- [ ] **Edit schedule** - Modify existing schedule specs
- [ ] **Schedule calendar view** - Visual calendar showing upcoming runs
- [ ] **Schedule run history** - View past runs from a schedule

## Namespace Management

- [ ] **Namespace details view** - Full namespace config (archival, clusters, etc.)
- [ ] **Namespace usage stats** - Workflow counts, retention info

## Quality of Life

- [ ] **Bookmarks** - Save specific workflows for quick access
- [ ] **Export to JSON/YAML** - Export workflow details or event history
- [ ] **Tail/stream mode** - Live-follow workflow events as they happen
- [ ] **Workflow templates** - Quick-start new workflows from templates
- [ ] **Multi-cluster support** - Connect to multiple Temporal clusters
- [ ] **Session persistence** - Remember last viewed namespace/workflow on restart
- [ ] **Keyboard macro recording** - Record and replay navigation sequences

## Configuration

- [ ] **Connection profiles** - Save multiple Temporal server configs
- [ ] **Namespace favorites** - Pin frequently used namespaces
- [ ] **Custom keybindings** - User-configurable key mappings
- [ ] **Column customization** - Choose which columns to display in tables

## Developer Experience

- [ ] **Code snippets** - Generate SDK code to replicate a workflow/signal
- [ ] **Workflow replay debugger** - Step through events with state inspection
- [ ] **tctl/temporal CLI integration** - Show equivalent CLI commands (partially exists)
- [ ] **Log viewer** - Correlate structured logs with workflow execution
