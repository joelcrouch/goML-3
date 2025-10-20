# Sprint 1 Detailed Plan: Fault-Tolerant Raft Control Plane

## Week 1-2: Build a Production-Grade Orchestration Foundation

### 🎯 **Sprint 1 Objective**
Establish a working Raft consensus cluster across AWS and GCP that can survive Leader failures while maintaining a consistent Global Task Manifest. Python Node Agents heartbeat to the Control Plane, proving dynamic membership and fault detection.

### 📊 **Success Criteria**
- ✅ 6-node Raft cluster operational (3 AWS, 3 GCP)
- ✅ Leader election completes in <10 seconds
- ✅ Leader failure triggers automatic re-election
- ✅ Task Manifest state consistent across all nodes
- ✅ Python Node Agents successfully heartbeat to current Leader
- ✅ System detects and handles node failures

---

## 🗓️ **Week-by-Week Breakdown**

### **Week 1 (Days 1-7): Infrastructure & Raft Foundation**

#### **Day 1-2: Multi-Cloud Infrastructure Deployment (Story 1.1)**
**Effort**: 8 points

**Implementation Tasks:**
- ✅ Create Terraform modules for AWS infrastructure (3 EC2 instances)
- ✅ Create Terraform modules for GCP infrastructure (2 Compute Engine instances)
- [ ] Configure cross-cloud VPC peering or VPN tunnel (AWS ↔ GCP connectivity)
- [ ] Set up security groups and firewall rules for gRPC ports (default: 50051)
- [ ] Install Go runtime (1.21+) on all nodes
- [ ] Install Python 3.11+ on all nodes
- [ ] Configure persistent storage for Raft logs (EBS on AWS, Persistent Disk on GCP)

**Infrastructure Specifications:**
```hcl
# AWS EC2 Instance Spec
resource "aws_instance" "raft_node" {
  count         = 3
  instance_type = "t3.medium"  # 2 vCPU, 4GB RAM
  ami           = "ami-xxxxx"  # Ubuntu 22.04 LTS
  
  root_block_device {
    volume_size = 50  # GB for Raft logs
    volume_type = "gp3"
  }
  
  tags = {
    Name = "raft-node-aws-${count.index + 1}"
    Role = "raft-control-plane"
  }
}

# GCP Compute Engine Spec
resource "google_compute_instance" "raft_node" {
  count        = 2
  name         = "raft-node-gcp-${count.index + 1}"
  machine_type = "n1-standard-2"  # 2 vCPU, 7.5GB RAM
  
  boot_disk {
    initialize_params {
      size  = 50  # GB
      type  = "pd-ssd"
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
    }
  }
  
  labels = {
    role = "raft-control-plane"
  }
}
```

**Network Configuration:**
```hcl
# Cross-cloud connectivity options:

# Option 1: VPN Tunnel (recommended for production)
resource "aws_vpn_connection" "main" {
  vpn_gateway_id      = aws_vpn_gateway.main.id
  customer_gateway_id = aws_customer_gateway.gcp.id
  type                = "ipsec.1"
  static_routes_only  = true
}

# Option 2: Public IP with firewall rules (simpler for testing)
# AWS Security Group
resource "aws_security_group" "raft" {
  ingress {
    from_port   = 50051
    to_port     = 50051
    protocol    = "tcp"
    cidr_blocks = [var.gcp_subnet_cidr]
  }
}

# GCP Firewall Rule
resource "google_compute_firewall" "raft" {
  name    = "allow-raft-grpc"
  network = google_compute_network.main.name
  
  allow {
    protocol = "tcp"
    ports    = ["50051"]
  }
  
  source_ranges = [var.aws_subnet_cidr]
}
```

**Testing (Day 2):**
- [ ] Verify SSH access to all 5 nodes
- [ ] Test cross-cloud connectivity (ping AWS ↔ GCP)
- [ ] Verify gRPC port accessibility from all nodes
- [ ] Confirm Go and Python installed correctly
- [ ] Test persistent storage write/read performance

**Acceptance Criteria:**
- ✅ 5 nodes deployed and accessible (3 AWS, 2 GCP)
- ✅ Cross-cloud network connectivity verified (latency <50ms)<---this is highly unlikely, i think the cross cloud ping is at ~243ish ms and intra cloud pings are ~25ishms

- ✅ gRPC ports open and accessible
- ✅ Go 1.21+ and Python 3.11+ installed on all nodes
- ✅ Persistent storage mounted and writable

#### **Day 3-5: Raft State, Log, & Leader Election (Story 1.2)**
**Effort**: 8 points

**Implementation Tasks:**
- [ ] Integrate HashiCorp Raft library (`github.com/hashicorp/raft`)
- [ ] Implement Raft configuration and initialization
- [ ] Set up persistent storage backend for Raft logs (BoltDB)
- [ ] Configure Raft transport layer (TCP via `raft-boltdb` and `raft-tcp`)
- [ ] Implement Leader election logic and state transitions
- [ ] Add logging and monitoring for Raft events
- [ ] Create cluster bootstrap script for initial setup

**Code Structure:**
```go
// cmd/control-plane/main.go
package main

import (
    "github.com/hashicorp/raft"
    raftboltdb "github.com/hashicorp/raft-boltdb"
)

type RaftNode struct {
    raft          *raft.Raft
    fsm           *TaskManifestFSM
    config        *raft.Config
    transport     *raft.NetworkTransport
    logStore      *raftboltdb.BoltStore
    stableStore   *raftboltdb.BoltStore
    snapshotStore raft.SnapshotStore
}

func NewRaftNode(nodeID, raftDir, bindAddr string, peers []string) (*RaftNode, error) {
    // 1. Create Raft configuration
    config := raft.DefaultConfig()
    config.LocalID = raft.ServerID(nodeID)
    config.HeartbeatTimeout = 1000 * time.Millisecond
    config.ElectionTimeout = 1000 * time.Millisecond
    config.CommitTimeout = 50 * time.Millisecond
    config.LeaderLeaseTimeout = 500 * time.Millisecond
    
    // 2. Set up persistent storage (BoltDB)
    logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
    if err != nil {
        return nil, fmt.Errorf("failed to create log store: %w", err)
    }
    
    stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
    if err != nil {
        return nil, fmt.Errorf("failed to create stable store: %w", err)
    }
    
    snapshotStore, err := raft.NewFileSnapshotStore(raftDir, 2, os.Stderr)
    if err != nil {
        return nil, fmt.Errorf("failed to create snapshot store: %w", err)
    }
    
    // 3. Set up network transport (TCP)
    addr, err := net.ResolveTCPAddr("tcp", bindAddr)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve bind address: %w", err)
    }
    
    transport, err := raft.NewTCPTransport(bindAddr, addr, 3, 10*time.Second, os.Stderr)
    if err != nil {
        return nil, fmt.Errorf("failed to create transport: %w", err)
    }
    
    // 4. Create FSM (finite state machine)
    fsm := NewTaskManifestFSM()
    
    // 5. Create Raft instance
    raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
    if err != nil {
        return nil, fmt.Errorf("failed to create raft: %w", err)
    }
    
    // 6. Bootstrap cluster if this is the first node
    if len(peers) > 0 {
        configuration := raft.Configuration{
            Servers: make([]raft.Server, 0, len(peers)+1),
        }
        
        // Add self
        configuration.Servers = append(configuration.Servers, raft.Server{
            ID:      raft.ServerID(nodeID),
            Address: raft.ServerAddress(bindAddr),
        })
        
        // Add peers
        for _, peer := range peers {
            configuration.Servers = append(configuration.Servers, raft.Server{
                ID:      raft.ServerID(peer),
                Address: raft.ServerAddress(peer),
            })
        }
        
        raftNode.BootstrapCluster(configuration)
    }
    
    return &RaftNode{
        raft:          raftNode,
        fsm:           fsm,
        config:        config,
        transport:     transport,
        logStore:      logStore,
        stableStore:   stableStore,
        snapshotStore: snapshotStore,
    }, nil
}

func (rn *RaftNode) IsLeader() bool {
    return rn.raft.State() == raft.Leader
}

func (rn *RaftNode) GetLeader() string {
    return string(rn.raft.Leader())
}

func (rn *RaftNode) WaitForLeader(timeout time.Duration) error {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    timeoutCh := time.After(timeout)
    
    for {
        select {
        case <-ticker.C:
            if leader := rn.GetLeader(); leader != "" {
                return nil
            }
        case <-timeoutCh:
            return fmt.Errorf("timeout waiting for leader election")
        }
    }
}
```

**Raft Configuration Tuning:**
```go
// Aggressive timeouts for testing (production should be higher)
config.HeartbeatTimeout = 1000 * time.Millisecond  // Leader heartbeat frequency
config.ElectionTimeout = 1000 * time.Millisecond   // Follower election trigger
config.CommitTimeout = 50 * time.Millisecond       // Log commit timeout
config.LeaderLeaseTimeout = 500 * time.Millisecond // Leader lease duration

// Snapshot configuration
config.SnapshotInterval = 120 * time.Second        // Snapshot every 2 minutes
config.SnapshotThreshold = 8192                    // Snapshot after 8K log entries
```

**Testing (Day 4-5):**
- [ ] Unit tests for Raft node initialization
- [ ] Integration test: Start 5 nodes, verify cluster formation
- [ ] Leader election test: Verify one Leader elected within 10 seconds
- [ ] Failover test: Kill Leader, verify new Leader elected
- [ ] Network partition test: Isolate minority, verify majority continues
- [ ] Log persistence test: Restart node, verify log recovery

**Acceptance Criteria:**
- ✅ Raft cluster forms successfully with 5 nodes
- ✅ Leader elected within 10 seconds of startup
- ✅ All nodes transition correctly (Follower → Candidate → Leader)
- ✅ Kill Leader → new Leader elected in <5 seconds
- ✅ Raft logs persisted to disk (survive restarts)
- ✅ Network partition handled correctly (majority partition continues)

#### **Day 6-7: Raft State Application & Task Manifest (Story 1.3)**
**Effort**: 5 points

**Implementation Tasks:**
- [ ] Define Task Manifest data structures
- [ ] Implement Finite State Machine (FSM) with `Apply()` method
- [ ] Create log entry types for task operations
- [ ] Implement FSM snapshot and restore methods
- [ ] Add RPC endpoint for proposing tasks to Leader
- [ ] Implement read-only queries from FSM

**Code Structure:**
```go
// internal/fsm/task_manifest.go
package fsm

import (
    "encoding/json"
    "io"
    "sync"
    
    "github.com/hashicorp/raft"
)

// Task represents a work unit to be executed
type Task struct {
    ID          string            `json:"id"`
    Type        string            `json:"type"`  // "ingestion", "processing", etc.
    Status      string            `json:"status"` // "pending", "assigned", "running", "complete", "failed"
    AssignedTo  string            `json:"assigned_to,omitempty"`
    Payload     map[string]string `json:"payload"`
    CreatedAt   int64             `json:"created_at"`
    CompletedAt int64             `json:"completed_at,omitempty"`
}

// NodeInfo represents a registered node
type NodeInfo struct {
    ID            string  `json:"id"`
    Address       string  `json:"address"`
    Cloud         string  `json:"cloud"`  // "aws" or "gcp"
    LastHeartbeat int64   `json:"last_heartbeat"`
    Status        string  `json:"status"` // "healthy", "degraded", "unreachable"
    Capacity      int     `json:"capacity"` // Max concurrent tasks
    ActiveTasks   int     `json:"active_tasks"`
}

// TaskManifestFSM is the Raft state machine
type TaskManifestFSM struct {
    mu          sync.RWMutex
    tasks       map[string]*Task       // Task ID → Task
    nodes       map[string]*NodeInfo   // Node ID → NodeInfo
    assignments map[string]string      // Task ID → Node ID
}

func NewTaskManifestFSM() *TaskManifestFSM {
    return &TaskManifestFSM{
        tasks:       make(map[string]*Task),
        nodes:       make(map[string]*NodeInfo),
        assignments: make(map[string]string),
    }
}

// LogEntryType defines the type of Raft log entry
type LogEntryType string

const (
    AddTaskLog        LogEntryType = "add_task"
    AssignTaskLog     LogEntryType = "assign_task"
    CompleteTaskLog   LogEntryType = "complete_task"
    FailTaskLog       LogEntryType = "fail_task"
    NodeHeartbeatLog  LogEntryType = "node_heartbeat"
    NodeRegisterLog   LogEntryType = "node_register"
    NodeUnregisterLog LogEntryType = "node_unregister"
)

// LogEntry represents a single Raft log entry
type LogEntry struct {
    Type    LogEntryType    `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

// Apply applies a committed Raft log entry to the FSM
func (fsm *TaskManifestFSM) Apply(log *raft.Log) interface{} {
    var entry LogEntry
    if err := json.Unmarshal(log.Data, &entry); err != nil {
        return fmt.Errorf("failed to unmarshal log entry: %w", err)
    }
    
    fsm.mu.Lock()
    defer fsm.mu.Unlock()
    
    switch entry.Type {
    case AddTaskLog:
        return fsm.applyAddTask(entry.Payload)
    case AssignTaskLog:
        return fsm.applyAssignTask(entry.Payload)
    case CompleteTaskLog:
        return fsm.applyCompleteTask(entry.Payload)
    case FailTaskLog:
        return fsm.applyFailTask(entry.Payload)
    case NodeHeartbeatLog:
        return fsm.applyNodeHeartbeat(entry.Payload)
    case NodeRegisterLog:
        return fsm.applyNodeRegister(entry.Payload)
    case NodeUnregisterLog:
        return fsm.applyNodeUnregister(entry.Payload)
    default:
        return fmt.Errorf("unknown log entry type: %s", entry.Type)
    }
}

func (fsm *TaskManifestFSM) applyAddTask(payload json.RawMessage) interface{} {
    var task Task
    if err := json.Unmarshal(payload, &task); err != nil {
        return err
    }
    
    task.Status = "pending"
    task.CreatedAt = time.Now().Unix()
    fsm.tasks[task.ID] = &task
    
    return nil
}

func (fsm *TaskManifestFSM) applyAssignTask(payload json.RawMessage) interface{} {
    var assignment struct {
        TaskID string `json:"task_id"`
        NodeID string `json:"node_id"`
    }
    
    if err := json.Unmarshal(payload, &assignment); err != nil {
        return err
    }
    
    if task, ok := fsm.tasks[assignment.TaskID]; ok {
        task.Status = "assigned"
        task.AssignedTo = assignment.NodeID
        fsm.assignments[assignment.TaskID] = assignment.NodeID
        
        if node, ok := fsm.nodes[assignment.NodeID]; ok {
            node.ActiveTasks++
        }
    }
    
    return nil
}

func (fsm *TaskManifestFSM) applyCompleteTask(payload json.RawMessage) interface{} {
    var completion struct {
        TaskID string `json:"task_id"`
    }
    
    if err := json.Unmarshal(payload, &completion); err != nil {
        return err
    }
    
    if task, ok := fsm.tasks[completion.TaskID]; ok {
        task.Status = "complete"
        task.CompletedAt = time.Now().Unix()
        
        if nodeID, ok := fsm.assignments[completion.TaskID]; ok {
            if node, ok := fsm.nodes[nodeID]; ok {
                node.ActiveTasks--
            }
        }
    }
    
    return nil
}

func (fsm *TaskManifestFSM) applyNodeHeartbeat(payload json.RawMessage) interface{} {
    var heartbeat struct {
        NodeID string `json:"node_id"`
        Status string `json:"status"`
    }
    
    if err := json.Unmarshal(payload, &heartbeat); err != nil {
        return err
    }
    
    if node, ok := fsm.nodes[heartbeat.NodeID]; ok {
        node.LastHeartbeat = time.Now().Unix()
        node.Status = heartbeat.Status
    }
    
    return nil
}

// Snapshot returns a snapshot of the current FSM state
func (fsm *TaskManifestFSM) Snapshot() (raft.FSMSnapshot, error) {
    fsm.mu.RLock()
    defer fsm.mu.RUnlock()
    
    // Deep copy the state
    snapshot := &TaskManifestSnapshot{
        tasks:       make(map[string]*Task),
        nodes:       make(map