package raft

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	pb "ml-raft-control-plane/pkg/proto"

	"github.com/google/uuid"
	"github.com/hashicorp/raft"
)

// setupFSM creates a new FSM for testing
func setupFSM(t *testing.T) *TaskManifestFSM {
	return NewTaskManifestFSM()
}

// helper to apply a log entry to the FSM
func applyLog(t *testing.T, fsm *TaskManifestFSM, entryType LogEntryType, data interface{}) {
	t.Helper()
	encodedData, err := EncodeLogEntry(entryType, data)
	if err != nil {
		t.Fatalf("failed to encode log entry: %v", err)
	}

	log := &raft.Log{
		Data: encodedData,
	}

	resp := fsm.Apply(log)
	if resp != nil {
		if err, ok := resp.(error); ok {
			t.Fatalf("fsm.Apply() returned an error: %v", err)
		}
	}
}

func TestFSM_Apply_RegisterNode(t *testing.T) {
	fsm := setupFSM(t)
	nodeID := uuid.NewString()
	entry := RegisterNodeEntry{
		NodeID:       nodeID,
		Address:      "localhost:50051",
		RegisteredAt: time.Now().Unix(),
	}

	applyLog(t, fsm, LogEntryRegisterNode, entry)

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	if len(fsm.manifest.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(fsm.manifest.Nodes))
	}
	if _, ok := fsm.manifest.Nodes[nodeID]; !ok {
		t.Fatalf("node with ID %s was not registered", nodeID)
	}
	if fsm.manifest.Nodes[nodeID].Status != pb.NodeStatus_HEALTHY {
		t.Errorf("expected node status to be HEALTHY, got %s", fsm.manifest.Nodes[nodeID].Status)
	}
}

func TestFSM_Apply_AddTask(t *testing.T) {
	fsm := setupFSM(t)
	taskID := uuid.NewString()
	entry := AddTaskEntry{
		TaskID:   taskID,
		TaskType: "matmul",
	}

	applyLog(t, fsm, LogEntryAddTask, entry)

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	if len(fsm.manifest.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(fsm.manifest.Tasks))
	}
	if task, ok := fsm.manifest.Tasks[taskID]; !ok {
		t.Fatalf("task with ID %s was not added", taskID)
	} else if task.Status != pb.TaskStatus_PENDING {
		t.Errorf("expected task status to be PENDING, got %s", task.Status)
	}
}

func TestFSM_Apply_AssignTask(t *testing.T) {
	fsm := setupFSM(t)
	taskID := uuid.NewString()
	nodeID := uuid.NewString()

	// Pre-conditions: Add a task and a node
	applyLog(t, fsm, LogEntryAddTask, AddTaskEntry{TaskID: taskID})
	applyLog(t, fsm, LogEntryRegisterNode, RegisterNodeEntry{NodeID: nodeID})

	// Apply the assignment
	assignEntry := AssignTaskEntry{
		TaskID: taskID,
		NodeID: nodeID,
	}
	applyLog(t, fsm, LogEntryAssignTask, assignEntry)

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	task := fsm.manifest.Tasks[taskID]
	if task.Status != pb.TaskStatus_ASSIGNED {
		t.Errorf("expected task status ASSIGNED, got %s", task.Status)
	}
	if task.AssignedNodeId != nodeID {
		t.Errorf("expected task to be assigned to node %s, got %s", nodeID, task.AssignedNodeId)
	}

	node := fsm.manifest.Nodes[nodeID]
	if node.ActiveTasks != 1 {
		t.Errorf("expected node active tasks to be 1, got %d", node.ActiveTasks)
	}
}

func TestFSM_Apply_CompleteTask(t *testing.T) {
	fsm := setupFSM(t)
	taskID := uuid.NewString()
	nodeID := uuid.NewString()

	// Pre-conditions: Add task, register node, assign task
	applyLog(t, fsm, LogEntryAddTask, AddTaskEntry{TaskID: taskID})
	applyLog(t, fsm, LogEntryRegisterNode, RegisterNodeEntry{NodeID: nodeID})
	applyLog(t, fsm, LogEntryAssignTask, AssignTaskEntry{TaskID: taskID, NodeID: nodeID})

	// Apply completion
	resultData := json.RawMessage(`{"result": "success"}`)
	completeEntry := CompleteTaskEntry{
		TaskID:     taskID,
		ResultData: resultData,
	}
	applyLog(t, fsm, LogEntryCompleteTask, completeEntry)

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	task := fsm.manifest.Tasks[taskID]
	if task.Status != pb.TaskStatus_COMPLETED {
		t.Errorf("expected task status COMPLETED, got %s", task.Status)
	}
	if task.ResultData != string(resultData) {
		t.Errorf("unexpected result data: got %s", task.ResultData)
	}

	node := fsm.manifest.Nodes[nodeID]
	if node.ActiveTasks != 0 {
		t.Errorf("expected node active tasks to be 0 after completion, got %d", node.ActiveTasks)
	}
}

func TestFSM_Apply_NodeHeartbeat(t *testing.T) {
	fsm := setupFSM(t)
	nodeID := uuid.NewString()

	// Pre-condition: Register node
	applyLog(t, fsm, LogEntryRegisterNode, RegisterNodeEntry{NodeID: nodeID})

	// Apply heartbeat
	heartbeatEntry := NodeHeartbeatEntry{
		NodeID:      nodeID,
		CPUUsage:    55.5,
		MemoryUsage: 66.6,
		ActiveTasks: 2,
		Timestamp:   time.Now().Unix(),
	}
	applyLog(t, fsm, LogEntryNodeHeartbeat, heartbeatEntry)

	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	node := fsm.manifest.Nodes[nodeID]
	if node.CpuUsage != 55.5 {
		t.Errorf("expected CPU usage 55.5, got %f", node.CpuUsage)
	}
	if node.MemoryUsage != 66.6 {
		t.Errorf("expected memory usage 66.6, got %f", node.MemoryUsage)
	}
	if node.ActiveTasks != 2 {
		t.Errorf("expected active tasks 2, got %d", node.ActiveTasks)
	}
	if node.LastHeartbeat < heartbeatEntry.Timestamp {
		t.Error("heartbeat timestamp was not updated")
	}
}

func TestFSM_Snapshot_Restore(t *testing.T) {
	fsm := setupFSM(t)

	// 1. Apply some state to the FSM
	taskID := uuid.NewString()
	nodeID := uuid.NewString()
	applyLog(t, fsm, LogEntryAddTask, AddTaskEntry{TaskID: taskID, TaskType: "type1"})
	applyLog(t, fsm, LogEntryRegisterNode, RegisterNodeEntry{NodeID: nodeID, Address: "addr1"})

	// 2. Create a snapshot
	snapshot, err := fsm.Snapshot()
	if err != nil {
		t.Fatalf("fsm.Snapshot() returned error: %v", err)
	}

	// 3. Persist the snapshot to a buffer
	var buf bytes.Buffer
	sink := &mockSnapshotSink{writer: &buf}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("snapshot.Persist() returned error: %v", err)
	}
	snapshot.Release()

	// 4. Create a new FSM and restore from the buffer
	newFSM := setupFSM(t)
	reader := io.NopCloser(&buf)
	if err := newFSM.Restore(reader); err != nil {
		t.Fatalf("newFSM.Restore() returned error: %v", err)
	}

	// 5. Verify the state of the new FSM matches the original
	newFSM.mu.RLock()
	defer newFSM.mu.RUnlock()

	if len(newFSM.manifest.Tasks) != 1 {
		t.Fatalf("restored FSM has wrong number of tasks: got %d, want 1", len(newFSM.manifest.Tasks))
	}
	if _, ok := newFSM.manifest.Tasks[taskID]; !ok {
		t.Errorf("restored FSM missing task %s", taskID)
	}

	if len(newFSM.manifest.Nodes) != 1 {
		t.Fatalf("restored FSM has wrong number of nodes: got %d, want 1", len(newFSM.manifest.Nodes))
	}
	if _, ok := newFSM.manifest.Nodes[nodeID]; !ok {
		t.Errorf("restored FSM missing node %s", nodeID)
	}
}

// mockSnapshotSink is a helper for testing snapshot persistence
type mockSnapshotSink struct {
	writer io.Writer
	is     bool
}

func (m *mockSnapshotSink) Write(p []byte) (n int, err error) {
	return m.writer.Write(p)
}

func (m *mockSnapshotSink) Close() error {
	m.is = true
	return nil
}

func (m *mockSnapshotSink) ID() string {
	return "mock-sink"
}

func (m *mockSnapshotSink) Cancel() error {
	m.is = true
	return nil
}
