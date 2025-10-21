package raft

import (
    "encoding/json"
    "fmt"
)

// LogEntryType represents the type of operation in a Raft log entry
type LogEntryType uint8

const (
    LogEntryAddTask LogEntryType = iota
    LogEntryAssignTask
    LogEntryUpdateTaskStatus
    LogEntryCompleteTask
    LogEntryFailTask
    LogEntryNodeHeartbeat
    LogEntryRegisterNode
)

// LogEntry represents an operation to be applied to the FSM
type LogEntry struct {
    Type LogEntryType `json:"type"`
    Data []byte       `json:"data"`
}

// AddTaskEntry represents adding a new task
type AddTaskEntry struct {
    TaskID      string            `json:"task_id"`
    TaskType    string            `json:"task_type"`
    TaskData    json.RawMessage   `json:"task_data"`
    CreatedAt   int64             `json:"created_at"`
}

// AssignTaskEntry represents assigning a task to a node
type AssignTaskEntry struct {
    TaskID    string `json:"task_id"`
    NodeID    string `json:"node_id"`
    AssignedAt int64 `json:"assigned_at"`
}

// UpdateTaskStatusEntry represents updating task status
type UpdateTaskStatusEntry struct {
    TaskID    string `json:"task_id"`
    Status    string `json:"status"`
    UpdatedAt int64  `json:"updated_at"`
}

// CompleteTaskEntry represents completing a task
type CompleteTaskEntry struct {
    TaskID      string          `json:"task_id"`
    ResultData  json.RawMessage `json:"result_data"`
    CompletedAt int64           `json:"completed_at"`
}

// FailTaskEntry represents a task failure
type FailTaskEntry struct {
    TaskID       string `json:"task_id"`
    ErrorMessage string `json:"error_message"`
    FailedAt     int64  `json:"failed_at"`
}

// NodeHeartbeatEntry represents a node heartbeat
type NodeHeartbeatEntry struct {
    NodeID      string  `json:"node_id"`
    CPUUsage    float64 `json:"cpu_usage"`
    MemoryUsage float64 `json:"memory_usage"`
    ActiveTasks int32   `json:"active_tasks"`
    Timestamp   int64   `json:"timestamp"`
}

// RegisterNodeEntry represents registering a new node
type RegisterNodeEntry struct {
    NodeID        string `json:"node_id"`
    Address       string `json:"address"`
    CloudProvider string `json:"cloud_provider"`
    Region        string `json:"region"`
    RegisteredAt  int64  `json:"registered_at"`
}

// EncodeLogEntry creates a log entry from typed data
func EncodeLogEntry(entryType LogEntryType, data interface{}) ([]byte, error) {
    entryData, err := json.Marshal(data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal log entry data: %w", err)
    }
    
    entry := LogEntry{
        Type: entryType,
        Data: entryData,
    }
    
    encoded, err := json.Marshal(entry)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal log entry: %w", err)
    }
    
    return encoded, nil
}

// DecodeLogEntry parses a log entry
func DecodeLogEntry(data []byte) (*LogEntry, error) {
    var entry LogEntry
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, fmt.Errorf("failed to unmarshal log entry: %w", err)
    }
    return &entry, nil
}
