# Raft Control Plane

Go-based control plane for multi-cloud ML orchestration using Raft consensus.

## Quick Start

```bash
# Install dependencies
make deps

# Generate protobuf code
make proto

# Verify compilation
make verify

# Format code
make fmt
```

## Project Structure

```
control-plane/
├── cmd/raft-node/          # Main entry point
├── internal/
│   ├── raft/              # Raft cluster management
│   ├── api/               # gRPC API server
│   └── models/            # Data models (TaskManifest)
├── pkg/proto/             # Protocol buffer definitions
└── bin/                   # Build output
```

## Development Status

**Story 1.2 Part 1**: ✅ COMPLETE
- Project structure initialized
- Protocol buffers defined and generated
- Task Manifest data model implemented
- Build automation configured

**Story 1.2 Part 2**: ⏳ NEXT
- Raft FSM implementation
- Raft cluster initialization
- gRPC server implementation
- Main entry point

## Available Make Targets

| Target | Description |
|--------|-------------|
| `make all` | Generate protobuf and build binary |
| `make proto` | Generate protobuf code |
| `make build` | Build the raft-node binary |
| `make test` | Run tests |
| `make verify` | Verify code compiles |
| `make clean` | Clean build artifacts |
| `make deps` | Install dependencies |
| `make help` | Display all targets |

## Protocol Buffers

### Services

**TaskService** - Task management
- `SubmitTask` - Submit new task
- `GetTask` - Query task status
- `ListTasks` - List all tasks

**NodeService** - Node management
- `Heartbeat` - Node health check
- `PollTask` - Request task assignment
- `ReportTaskResult` - Report completion

### Message Types

- `Task` - Computational task definition
- `Node` - Worker node metadata
- `TaskStatus` - PENDING, ASSIGNED, RUNNING, COMPLETED, FAILED
- `NodeStatus` - HEALTHY, UNHEALTHY, UNKNOWN

## Task Manifest API

```go
manifest := models.NewTaskManifest()

// Task operations
manifest.AddTask(task)
manifest.GetTask(taskID)
manifest.AssignTask(taskID, nodeID)
manifest.CompleteTask(taskID, result)

// Node operations
manifest.UpdateNodeHeartbeat(nodeID, cpu, mem, tasks)
manifest.GetHealthyNodes()
manifest.SelectLeastLoadedNode()
```

## Dependencies

### Core
- `google.golang.org/grpc` - gRPC framework
- `google.golang.org/protobuf` - Protocol buffers

### To Be Added (Part 2)
- `github.com/hashicorp/raft` - Raft consensus
- `github.com/hashicorp/raft-boltdb` - BoltDB backend
- `github.com/google/uuid` - UUID generation
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration

## Testing

```bash
# Run all tests
make test

# Run with race detector
make test-race

# Run specific package
go test -v ./internal/models
```

## Documentation

- [Story 1.2 Part 1 Completion Report](../docs/story_1.2_part1_completion.md)
- [Sprint 1 Detailed Plan](../docs/sprint1_raft_detailed.md)
- [Living Document](../docs/living_doc_updated.md)

## Next Steps

1. Implement Raft FSM (`internal/raft/fsm.go`)
2. Initialize Raft cluster (`internal/raft/cluster.go`)
3. Implement gRPC server (`internal/api/grpc_server.go`)
4. Create main entry point (`cmd/raft-node/main.go`)

See [Story 1.2 Part 1 Completion Report](../docs/story_1.2_part1_completion.md) for details.

---

**Status**: Part 1 Complete ✅
**Last Updated**: 2025-10-19
