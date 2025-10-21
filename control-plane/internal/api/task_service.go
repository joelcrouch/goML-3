package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	//"google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"

	"ml-raft-control-plane/internal/raft"
	pb "ml-raft-control-plane/pkg/proto"
)

// tsakserviceServer =>the taskservide grpc servide
type TaskServiceServer struct {
	pb.UnimplementedTaskServiceServer
	cluster *raft.RaftCluster
}

// => makes a new taskService TaskServiceServ
func NewTaskServiceServer(cluster *raft.RaftCluster) *TaskServiceServer {
	return &TaskServiceServer{
		cluster: cluster,
	}
}

// SubmitTask handles task submission requests
func (s *TaskServiceServer) SubmitTask(ctx context.Context, req *pb.SubmitTaskRequest) (*pb.SubmitTaskResponse, error) {
	// Check if we're the leader
	if !s.cluster.IsLeader() {
		leader := s.cluster.GetLeader()
		return &pb.SubmitTaskResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("not leader, current leader: %s", leader),
		}, nil
	}

	// Generate task ID
	taskID := uuid.New().String()

	// Create AddTask log entry
	addEntry := raft.AddTaskEntry{
		TaskID:    taskID,
		TaskType:  req.TaskType,
		TaskData:  json.RawMessage(req.TaskData),
		CreatedAt: time.Now().Unix(),
	}

	// Encode log entry
	data, err := raft.EncodeLogEntry(raft.LogEntryAddTask, addEntry)
	if err != nil {
		return &pb.SubmitTaskResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to encode log entry: %v", err),
		}, nil
	}

	// Apply to Raft cluster
	if err := s.cluster.Apply(data, 5*time.Second); err != nil {
		return &pb.SubmitTaskResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to apply log entry: %v", err),
		}, nil
	}

	return &pb.SubmitTaskResponse{
		TaskId:  taskID,
		Success: true,
	}, nil
}

// GetTask retrieves a task by ID
func (s *TaskServiceServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	// Get manifest from FSM (read-only, no consensus needed)
	manifest := s.cluster.GetFSM().GetManifest()

	task, found := manifest.Tasks[req.TaskId]
	if !found {
		return &pb.GetTaskResponse{
			Found: false,
		}, nil
	}

	return &pb.GetTaskResponse{
		Task:  task,
		Found: true,
	}, nil
}

// ListTasks lists tasks with optional status filter
func (s *TaskServiceServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	manifest := s.cluster.GetFSM().GetManifest()

	var tasks []*pb.Task

	// Apply filter if specified
	for _, task := range manifest.Tasks {
		// If status filter is set and doesn't match, skip
		if req.StatusFilter != pb.TaskStatus_PENDING && task.Status != req.StatusFilter {
			continue
		}

		tasks = append(tasks, task)

		// Apply limit
		if req.Limit > 0 && int32(len(tasks)) >= req.Limit {
			break
		}
	}

	return &pb.ListTasksResponse{
		Tasks: tasks,
	}, nil
}
