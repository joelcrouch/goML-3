package models

import (
	"time"

	pb "ml-raft-control-plane/pkg/proto"
)

// TaskManifest represents the global state of all tasks (FSM state)
type TaskManifest struct {
	Tasks map[string]*pb.Task // task_id -> Task
	Nodes map[string]*pb.Node // node_id -> Node
}

// NewTaskManifest creates empty task manifest
func NewTaskManifest() *TaskManifest {
	return &TaskManifest{
		Tasks: make(map[string]*pb.Task),
		Nodes: make(map[string]*pb.Node),
	}
}

// AddTask adds a new task to the manifest
func (tm *TaskManifest) AddTask(task *pb.Task) {
	tm.Tasks[task.TaskId] = task
}

// GetTask retrieves a task by ID
func (tm *TaskManifest) GetTask(taskID string) (*pb.Task, bool) {
	task, exists := tm.Tasks[taskID]
	return task, exists
}

// UpdateTaskStatus updates task status
func (tm *TaskManifest) UpdateTaskStatus(taskID string, status pb.TaskStatus) bool {
	if task, exists := tm.Tasks[taskID]; exists {
		task.Status = status
		return true
	}
	return false
}

// AssignTask assigns task to a node
func (tm *TaskManifest) AssignTask(taskID, nodeID string) bool {
	if task, exists := tm.Tasks[taskID]; exists {
		task.AssignedNodeId = nodeID
		task.Status = pb.TaskStatus_ASSIGNED
		task.StartedAt = time.Now().Unix()

		// Increment node's active task count
		if node, nodeExists := tm.Nodes[nodeID]; nodeExists {
			node.ActiveTasks++
		}
		return true
	}
	return false
}

// CompleteTask marks task as completed
func (tm *TaskManifest) CompleteTask(taskID, resultData string) bool {
	if task, exists := tm.Tasks[taskID]; exists {
		task.Status = pb.TaskStatus_COMPLETED
		task.CompletedAt = time.Now().Unix()
		task.ResultData = resultData

		// Decrement node's active task count
		if node, nodeExists := tm.Nodes[task.AssignedNodeId]; nodeExists && node.ActiveTasks > 0 {
			node.ActiveTasks--
		}
		return true
	}
	return false
}

// FailTask marks task as failed
func (tm *TaskManifest) FailTask(taskID, errorMessage string) bool {
	if task, exists := tm.Tasks[taskID]; exists {
		task.Status = pb.TaskStatus_FAILED
		task.CompletedAt = time.Now().Unix()
		task.ResultData = errorMessage

		// Decrement node's active task count
		if node, nodeExists := tm.Nodes[task.AssignedNodeId]; nodeExists && node.ActiveTasks > 0 {
			node.ActiveTasks--
		}
		return true
	}
	return false
}

// UpdateNodeHeartbeat updates node status
func (tm *TaskManifest) UpdateNodeHeartbeat(nodeID string, cpuUsage, memUsage float64, activeTasks int32) {
	node, exists := tm.Nodes[nodeID]
	if !exists {
		// Create new node
		node = &pb.Node{
			NodeId: nodeID,
			Status: pb.NodeStatus_HEALTHY,
		}
		tm.Nodes[nodeID] = node
	}

	node.LastHeartbeat = time.Now().Unix()
	node.CpuUsage = cpuUsage
	node.MemoryUsage = memUsage
	node.ActiveTasks = activeTasks
	node.Status = pb.NodeStatus_HEALTHY
}

// MarkNodeUnhealthy marks a node as unhealthy
func (tm *TaskManifest) MarkNodeUnhealthy(nodeID string) bool {
	if node, exists := tm.Nodes[nodeID]; exists {
		node.Status = pb.NodeStatus_UNHEALTHY
		return true
	}
	return false
}

// GetPendingTasks returns all pending tasks
func (tm *TaskManifest) GetPendingTasks() []*pb.Task {
	var pending []*pb.Task
	for _, task := range tm.Tasks {
		if task.Status == pb.TaskStatus_PENDING {
			pending = append(pending, task)
		}
	}
	return pending
}

// GetTasksByStatus returns all tasks with given status
func (tm *TaskManifest) GetTasksByStatus(status pb.TaskStatus) []*pb.Task {
	var tasks []*pb.Task
	for _, task := range tm.Tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetAllTasks returns all tasks
func (tm *TaskManifest) GetAllTasks() []*pb.Task {
	tasks := make([]*pb.Task, 0, len(tm.Tasks))
	for _, task := range tm.Tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetHealthyNodes returns all healthy nodes
func (tm *TaskManifest) GetHealthyNodes() []*pb.Node {
	var healthy []*pb.Node
	for _, node := range tm.Nodes {
		if node.Status == pb.NodeStatus_HEALTHY {
			healthy = append(healthy, node)
		}
	}
	return healthy
}

// GetAllNodes returns all nodes
func (tm *TaskManifest) GetAllNodes() []*pb.Node {
	nodes := make([]*pb.Node, 0, len(tm.Nodes))
	for _, node := range tm.Nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// SelectLeastLoadedNode returns node with fewest active tasks
func (tm *TaskManifest) SelectLeastLoadedNode() *pb.Node {
	healthy := tm.GetHealthyNodes()
	if len(healthy) == 0 {
		return nil
	}

	minNode := healthy[0]
	for _, node := range healthy[1:] {
		if node.ActiveTasks < minNode.ActiveTasks {
			minNode = node
		}
	}
	return minNode
}

// CheckStaleNodes checks for nodes that haven't sent heartbeat recently
func (tm *TaskManifest) CheckStaleNodes(timeoutSeconds int64) []string {
	now := time.Now().Unix()
	var staleNodes []string

	for nodeID, node := range tm.Nodes {
		if node.Status == pb.NodeStatus_HEALTHY {
			if now-node.LastHeartbeat > timeoutSeconds {
				staleNodes = append(staleNodes, nodeID)
			}
		}
	}
	return staleNodes
}

// GetNodeTaskCount returns number of active tasks for a node
func (tm *TaskManifest) GetNodeTaskCount(nodeID string) int32 {
	if node, exists := tm.Nodes[nodeID]; exists {
		return node.ActiveTasks
	}
	return 0
}
