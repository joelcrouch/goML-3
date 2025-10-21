package raft

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftCluster manages the Raft consensus cluster
type RaftCluster struct {
	raft          *raft.Raft
	fsm           *TaskManifestFSM
	config        *ClusterConfig
	transport     *raft.NetworkTransport
	logStore      *raftboltdb.BoltStore
	stableStore   *raftboltdb.BoltStore
	snapshotStore raft.SnapshotStore
}

// ClusterConfig holds Raft cluster configuration
type ClusterConfig struct {
	NodeID            string
	BindAddress       string
	DataDir           string
	BootstrapExpect   int
	Peers             []string
	HeartbeatTimeout  time.Duration
	ElectionTimeout   time.Duration
	CommitTimeout     time.Duration
	SnapshotInterval  time.Duration
	SnapshotThreshold uint64
}

// NewRaftCluster creates and initializes a new Raft cluster
func NewRaftCluster(config *ClusterConfig) (*RaftCluster, error) {
	// Create FSM
	fsm := NewTaskManifestFSM()

	// Setup Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(config.NodeID)
	raftConfig.HeartbeatTimeout = config.HeartbeatTimeout
	raftConfig.ElectionTimeout = config.ElectionTimeout
	raftConfig.CommitTimeout = config.CommitTimeout
	raftConfig.SnapshotInterval = config.SnapshotInterval
	raftConfig.SnapshotThreshold = config.SnapshotThreshold

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Setup log store (BoltDB)
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(config.DataDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %w", err)
	}

	// Setup stable store (BoltDB)
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(config.DataDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %w", err)
	}

	// Setup snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(config.DataDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Setup network transport
	addr, err := net.ResolveTCPAddr("tcp", config.BindAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve bind address: %w", err)
	}

	transport, err := raft.NewTCPTransport(config.BindAddress, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Create Raft instance
	raftNode, err := raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft node: %w", err)
	}

	cluster := &RaftCluster{
		raft:          raftNode,
		fsm:           fsm,
		config:        config,
		transport:     transport,
		logStore:      logStore,
		stableStore:   stableStore,
		snapshotStore: snapshotStore,
	}

	// Bootstrap cluster if needed
	if config.BootstrapExpect > 0 {
		if err := cluster.bootstrap(); err != nil {
			return nil, fmt.Errorf("failed to bootstrap cluster: %w", err)
		}
	}

	return cluster, nil
}

// bootstrap initializes the Raft cluster
func (rc *RaftCluster) bootstrap() error {
	// Check if already bootstrapped
	hasState, err := raft.HasExistingState(rc.logStore, rc.stableStore, rc.snapshotStore)
	if err != nil {
		return fmt.Errorf("failed to check existing state: %w", err)
	}

	if hasState {
		fmt.Println("Cluster already bootstrapped, skipping bootstrap")
		return nil
	}

	// Build server configuration
	servers := []raft.Server{
		{
			ID:      raft.ServerID(rc.config.NodeID),
			Address: raft.ServerAddress(rc.config.BindAddress),
		},
	}

	// Add peer servers
	for _, peerAddr := range rc.config.Peers {
		// Extract node ID from address (assuming format: ip:port)
		peerID := fmt.Sprintf("node-%s", peerAddr)
		servers = append(servers, raft.Server{
			ID:      raft.ServerID(peerID),
			Address: raft.ServerAddress(peerAddr),
		})
	}

	configuration := raft.Configuration{
		Servers: servers,
	}

	// Bootstrap the cluster
	future := rc.raft.BootstrapCluster(configuration)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to bootstrap cluster: %w", err)
	}

	fmt.Printf("Cluster bootstrapped with %d servers\n", len(servers))
	return nil
}

// Apply submits a log entry to Raft for consensus
func (rc *RaftCluster) Apply(data []byte, timeout time.Duration) error {
	future := rc.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to apply log entry: %w", err)
	}

	// Check if the apply returned an error
	if response := future.Response(); response != nil {
		if err, ok := response.(error); ok {
			return fmt.Errorf("FSM apply error: %w", err)
		}
	}

	return nil
}

// IsLeader returns true if this node is the current leader
func (rc *RaftCluster) IsLeader() bool {
	return rc.raft.State() == raft.Leader
}

// GetLeader returns the address of the current leader
func (rc *RaftCluster) GetLeader() string {
	_, leaderID := rc.raft.LeaderWithID()
	return string(leaderID)
}

// GetFSM returns the FSM
func (rc *RaftCluster) GetFSM() *TaskManifestFSM {
	return rc.fsm
}

// GetStats returns Raft statistics
func (rc *RaftCluster) GetStats() map[string]string {
	return rc.raft.Stats()
}

// WaitForLeader blocks until a leader is elected or timeout
func (rc *RaftCluster) WaitForLeader(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if rc.GetLeader() != "" {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for leader election")
			}
		}
	}
}

// Shutdown gracefully shuts down the Raft cluster
func (rc *RaftCluster) Shutdown() error {
	future := rc.raft.Shutdown()
	if err := future.Error(); err != nil {
		return fmt.Errorf("failed to shutdown raft: %w", err)
	}

	// Close stores
	if err := rc.logStore.Close(); err != nil {
		return fmt.Errorf("failed to close log store: %w", err)
	}

	if err := rc.stableStore.Close(); err != nil {
		return fmt.Errorf("failed to close stable store: %w", err)
	}

	return nil
}

// AddVoter adds a new voting member to the cluster
func (rc *RaftCluster) AddVoter(nodeID, address string, timeout time.Duration) error {
	if !rc.IsLeader() {
		return fmt.Errorf("only leader can add voters")
	}

	future := rc.raft.AddVoter(
		raft.ServerID(nodeID),
		raft.ServerAddress(address),
		0,
		timeout,
	)

	return future.Error()
}

// RemoveServer removes a server from the cluster
func (rc *RaftCluster) RemoveServer(nodeID string, timeout time.Duration) error {
	if !rc.IsLeader() {
		return fmt.Errorf("only leader can remove servers")
	}

	future := rc.raft.RemoveServer(
		raft.ServerID(nodeID),
		0,
		timeout,
	)

	return future.Error()
}
