package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"ml-raft-control-plane/internal/models"
	pb "ml-raft-control-plane/pkg/proto"

	"github.com/hashicorp/raft"
)

// TaskManifestFSM implements the Raft FSM interface for Task Manifest
type TaskManifestFSM struct {
	mu       sync.RWMutex
	manifest *models.TaskManifest
}

// NewTaskManifestFSM creates a new FSM
func NewTaskManifestFSM() *TaskManifestFSM {
	return &TaskManifestFSM{
		manifest: models.NewTaskManifest(),
	}
}

// Apply applies a Raft log entry to the FSM
// This is called by Raft when a log entry is committed
func (fsm *TaskManifestFSM) Apply(log *raft.Log) interface{} {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// Decode the log entry
	entry, err := DecodeLogEntry(log.Data)
	if err != nil {
		return fmt.Errorf("failed to decode log entry: %w", err)
	}

	// Apply based on entry type
	switch entry.Type {
	case LogEntryAddTask:
		return fsm.applyAddTask(entry.Data)
	case LogEntryAssignTask:
		return fsm.applyAssignTask(entry.Data)
	case LogEntryUpdateTaskStatus:
		return fsm.applyUpdateTaskStatus(entry.Data)
	case LogEntryCompleteTask:
		return fsm.applyCompleteTask(entry.Data)
	case LogEntryFailTask:
		return fsm.applyFailTask(entry.Data)
	case LogEntryNodeHeartbeat:
		return fsm.applyNodeHeartbeat(entry.Data)
	case LogEntryRegisterNode:
		return fsm.applyRegisterNode(entry.Data)
	default:
		return fmt.Errorf("unknown log entry type: %d", entry.Type)
	}
}

// applyAddTask adds a new task to the manifest
func (fsm *TaskManifestFSM) applyAddTask(data []byte) interface{} {
	var entry AddTaskEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal AddTaskEntry: %w", err)
	}

	task := &pb.Task{
		TaskId:    entry.TaskID,
		TaskType:  entry.TaskType,
		Status:    pb.TaskStatus_PENDING,
		TaskData:  entry.TaskData,
		CreatedAt: entry.CreatedAt,
	}

	fsm.manifest.AddTask(task)
	return nil
}

// applyAssignTask assigns a task to a node
func (fsm *TaskManifestFSM) applyAssignTask(data []byte) interface{} {
	var entry AssignTaskEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal AssignTaskEntry: %w", err)
	}

	if !fsm.manifest.AssignTask(entry.TaskID, entry.NodeID) {
		return fmt.Errorf("failed to assign task %s to node %s", entry.TaskID, entry.NodeID)
	}

	return nil
}

// applyUpdateTaskStatus updates task status
func (fsm *TaskManifestFSM) applyUpdateTaskStatus(data []byte) interface{} {
	var entry UpdateTaskStatusEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal UpdateTaskStatusEntry: %w", err)
	}

	// Convert string status to protobuf enum
	status := stringToTaskStatus(entry.Status)

	if !fsm.manifest.UpdateTaskStatus(entry.TaskID, status) {
		return fmt.Errorf("failed to update task %s status", entry.TaskID)
	}

	return nil
}

// applyCompleteTask marks a task as completed
func (fsm *TaskManifestFSM) applyCompleteTask(data []byte) interface{} {
	var entry CompleteTaskEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal CompleteTaskEntry: %w", err)
	}

	if !fsm.manifest.CompleteTask(entry.TaskID, string(entry.ResultData)) {
		return fmt.Errorf("failed to complete task %s", entry.TaskID)
	}

	return nil
}

// applyFailTask marks a task as failed
func (fsm *TaskManifestFSM) applyFailTask(data []byte) interface{} {
	var entry FailTaskEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal FailTaskEntry: %w", err)
	}

	fsm.manifest.UpdateTaskStatus(entry.TaskID, pb.TaskStatus_FAILED)
	return nil
}

// applyNodeHeartbeat updates node heartbeat
func (fsm *TaskManifestFSM) applyNodeHeartbeat(data []byte) interface{} {
	var entry NodeHeartbeatEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal NodeHeartbeatEntry: %w", err)
	}

	fsm.manifest.UpdateNodeHeartbeat(
		entry.NodeID,
		entry.CPUUsage,
		entry.MemoryUsage,
		entry.ActiveTasks,
	)

	return nil
}

// applyRegisterNode registers a new node
func (fsm *TaskManifestFSM) applyRegisterNode(data []byte) interface{} {
	var entry RegisterNodeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to unmarshal RegisterNodeEntry: %w", err)
	}

	node := &pb.Node{
		NodeId:        entry.NodeID,
		Address:       entry.Address,
		CloudProvider: entry.CloudProvider,
		Region:        entry.Region,
		Status:        pb.NodeStatus_HEALTHY,
		LastHeartbeat: entry.RegisteredAt,
	}

	fsm.manifest.Nodes[entry.NodeID] = node
	return nil
}

// Snapshot creates a point-in-time snapshot of the FSM state
// This is called periodically by Raft for compaction
func (fsm *TaskManifestFSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	// Create a deep copy of the manifest for snapshot
	snapshot := &TaskManifestSnapshot{
		manifest: fsm.copyManifest(),
	}

	return snapshot, nil
}

// Restore restores the FSM state from a snapshot
// This is called when a node is catching up from a snapshot
func (fsm *TaskManifestFSM) Restore(rc io.ReadCloser) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()
	defer rc.Close()

	// Decode snapshot data
	decoder := json.NewDecoder(rc)
	var snapshot TaskManifestSnapshotData
	if err := decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	// Restore manifest state
	fsm.manifest.Tasks = snapshot.Tasks
	fsm.manifest.Nodes = snapshot.Nodes

	return nil
}

// GetManifest returns a read-only view of the manifest
func (fsm *TaskManifestFSM) GetManifest() *models.TaskManifest {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()
	return fsm.manifest
}

// copyManifest creates a deep copy of the manifest
func (fsm *TaskManifestFSM) copyManifest() *models.TaskManifest {
	copy := models.NewTaskManifest()

	// Deep copy tasks
	for id, task := range fsm.manifest.Tasks {
		taskCopy := &pb.Task{
			TaskId:         task.TaskId,
			TaskType:       task.TaskType,
			Status:         task.Status,
			AssignedNodeId: task.AssignedNodeId,
			TaskData:       task.TaskData,
			CreatedAt:      task.CreatedAt,
			StartedAt:      task.StartedAt,
			CompletedAt:    task.CompletedAt,
			ResultData:     task.ResultData,
		}
		copy.Tasks[id] = taskCopy
	}

	// Deep copy nodes
	for id, node := range fsm.manifest.Nodes {
		nodeCopy := &pb.Node{
			NodeId:        node.NodeId,
			Address:       node.Address,
			CloudProvider: node.CloudProvider,
			Region:        node.Region,
			Status:        node.Status,
			LastHeartbeat: node.LastHeartbeat,
			CpuUsage:      node.CpuUsage,
			MemoryUsage:   node.MemoryUsage,
			ActiveTasks:   node.ActiveTasks,
		}
		copy.Nodes[id] = nodeCopy
	}

	return copy
}

// Helper: convert string to TaskStatus enum
func stringToTaskStatus(status string) pb.TaskStatus {
	switch status {
	case "PENDING":
		return pb.TaskStatus_PENDING
	case "ASSIGNED":
		return pb.TaskStatus_ASSIGNED
	case "RUNNING":
		return pb.TaskStatus_RUNNING
	case "COMPLETED":
		return pb.TaskStatus_COMPLETED
	case "FAILED":
		return pb.TaskStatus_FAILED
	default:
		return pb.TaskStatus_PENDING
	}
}

// TaskManifestSnapshot implements raft.FSMSnapshot
type TaskManifestSnapshot struct {
	manifest *models.TaskManifest
}

// Persist writes the snapshot to a sink
func (s *TaskManifestSnapshot) Persist(sink raft.SnapshotSink) error {
	// Encode snapshot as JSON
	snapshot := TaskManifestSnapshotData{
		Tasks: s.manifest.Tasks,
		Nodes: s.manifest.Nodes,
	}

	encoder := json.NewEncoder(sink)
	if err := encoder.Encode(snapshot); err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return sink.Close()
}

// Release is called when the snapshot is no longer needed
func (s *TaskManifestSnapshot) Release() {
	// No cleanup needed for in-memory snapshot
}

// TaskManifestSnapshotData represents serialized snapshot data
type TaskManifestSnapshotData struct {
	Tasks map[string]*pb.Task `json:"tasks"`
	Nodes map[string]*pb.Node `json:"nodes"`
}
