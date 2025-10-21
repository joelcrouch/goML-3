package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ml-raft-control-plane/internal/raft"
	pb "ml-raft-control-plane/pkg/proto"
)

type NodeServiceServer struct {
	pb.UnimplementedNodeServiceServer
	cluster *raft.RaftCluster
}

func NewNodeServiceServer(cluster *raft.RaftCluster) *NodeServiceServer {
	return &NodeServiceServer{
		cluster: cluster,
	}
}

// heartbeat handles node heartbeat requests
func (s *NodeServiceServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	///check if we are the leader
	if !s.cluster.IsLeader() {
		leader := s.cluster.GetLeader()
		return &pb.HeartbeatResponse{
			Acknowledged:  false,
			LeaderAddress: leader,
		}, nil
	}
	//make Heartbeat log entry
	heartbeatEntry := raft.NodeHeartbeatEntry{
		NodeID:      req.NodeId,
		CPUUsage:    req.CpuUsage,
		MemoryUsage: req.MemoryUsage,
		ActiveTasks: req.ActiveTasks,
		Timestamp:   time.Now().Unix(),
	}
	//now encode it
	data, err := raft.EncodeLogEntry(raft.LogEntryNodeHeartbeat, heartbeatEntry)
	if err != nil {
		return &pb.HeartbeatResponse{
			Acknowledged:  false,
			LeaderAddress: s.cluster.GetLeader(),
		}, fmt.Errorf("failed to encode heartbeat: %w", err)
	}
	//apply to raft we dont wait for consusus on heartbeast to reduce latency
	go func() {
		if err := s.cluster.Apply(data, 1*time.Second); err != nil {
			//log the errotBtu dont fail the heartbeat
			fmt.Print("warning: faliled to apply heartbeat for %s: %w\n", req.NodeId, err)
		}
	}()

	return &pb.HeartbeatResponse{
		Acknowledged:  true,
		LeaderAddress: s.cluster.GetLeader(),
	}, nil
}

// PollTask allows nodes to request task assignments
func (s *NodeServiceServer) PollTask(ctx context.Context, req *pb.PollTaskRequest) (*pb.PollTaskResponse, error) {
	// Only leader can assign tasks
	if !s.cluster.IsLeader() {
		return &pb.PollTaskResponse{
			HasTask: false,
		}, nil
	}

	manifest := s.cluster.GetFSM().GetManifest()

	// Find a pending task
	pendingTasks := manifest.GetPendingTasks()
	if len(pendingTasks) == 0 {
		return &pb.PollTaskResponse{
			HasTask: false,
		}, nil
	}

	// Select first pending task
	task := pendingTasks[0]

	// Assign task to this node
	assignEntry := raft.AssignTaskEntry{
		TaskID:     task.TaskId,
		NodeID:     req.NodeId,
		AssignedAt: time.Now().Unix(),
	}

	data, err := raft.EncodeLogEntry(raft.LogEntryAssignTask, assignEntry)
	if err != nil {
		return &pb.PollTaskResponse{
			HasTask: false,
		}, fmt.Errorf("failed to encode assign entry: %w", err)
	}

	// Apply assignment
	if err := s.cluster.Apply(data, 5*time.Second); err != nil {
		return &pb.PollTaskResponse{
			HasTask: false,
		}, fmt.Errorf("failed to apply assignment: %w", err)
	}

	// Update task status to RUNNING
	updateEntry := raft.UpdateTaskStatusEntry{
		TaskID:    task.TaskId,
		Status:    "RUNNING",
		UpdatedAt: time.Now().Unix(),
	}

	updateData, _ := raft.EncodeLogEntry(raft.LogEntryUpdateTaskStatus, updateEntry)
	s.cluster.Apply(updateData, 5*time.Second)

	return &pb.PollTaskResponse{
		Task:    task,
		HasTask: true,
	}, nil
}

// ReportTaskResult handles task completion reports from nodes
func (s *NodeServiceServer) ReportTaskResult(ctx context.Context, req *pb.ReportTaskResultRequest) (*pb.ReportTaskResultResponse, error) {
	if !s.cluster.IsLeader() {
		return &pb.ReportTaskResultResponse{
			Acknowledged: false,
		}, nil
	}

	var logEntry interface{}
	var entryType raft.LogEntryType

	// Create appropriate log entry based on status
	if req.FinalStatus == pb.TaskStatus_COMPLETED {
		logEntry = raft.CompleteTaskEntry{
			TaskID:      req.TaskId,
			ResultData:  json.RawMessage(req.ResultData),
			CompletedAt: time.Now().Unix(),
		}
		entryType = raft.LogEntryCompleteTask
	} else if req.FinalStatus == pb.TaskStatus_FAILED {
		logEntry = raft.FailTaskEntry{
			TaskID:       req.TaskId,
			ErrorMessage: req.ResultData,
			FailedAt:     time.Now().Unix(),
		}
		entryType = raft.LogEntryFailTask
	} else {
		return &pb.ReportTaskResultResponse{
			Acknowledged: false,
		}, fmt.Errorf("invalid final status: %v", req.FinalStatus)
	}

	// Encode and apply
	data, err := raft.EncodeLogEntry(entryType, logEntry)
	if err != nil {
		return &pb.ReportTaskResultResponse{
			Acknowledged: false,
		}, fmt.Errorf("failed to encode log entry: %w", err)
	}

	if err := s.cluster.Apply(data, 5*time.Second); err != nil {
		return &pb.ReportTaskResultResponse{
			Acknowledged: false,
		}, fmt.Errorf("failed to apply log entry: %w", err)
	}

	return &pb.ReportTaskResultResponse{
		Acknowledged: true,
	}, nil
}

//there are a copule clear bits of code we can extract to 1) make it cleaner 2)easier to test
//what really jumps out at me immediatiely is the 'isleader' functionality in all o fhte funcitons
//and some other parts seem like they are kinda like python decorators waiting to be applied...ill take a closer
//look after i get it rinngin.
