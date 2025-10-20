package raft

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// NodeConfig represents the configuration file structure
type NodeConfig struct {
	NodeID           string   `json:"node_id"`
	BindAddress      string   `json:"bind_address"`
	AdvertiseAddress string   `json:"advertise_address"`
	CloudProvider    string   `json:"cloud_provider"`
	Region           string   `json:"region"`
	DataDir          string   `json:"data_dir"`
	BootstrapExpect  int      `json:"bootstrap_expect"`
	Peers            []string `json:"peers"`
	Raft             struct {
		HeartbeatTimeout  string `json:"heartbeat_timeout"`
		ElectionTimeout   string `json:"election_timeout"`
		CommitTimeout     string `json:"commit_timeout"`
		SnapshotInterval  string `json:"snapshot_interval"`
		SnapshotThreshold uint64 `json:"snapshot_threshold"`
	} `json:"raft"`
	GRPC struct {
		Port                 int `json:"port"`
		MaxConcurrentStreams int `json:"max_concurrent_streams"`
	} `json:"grpc"`
}

// LoadConfig reads configuration from a JSON file
func LoadConfig(path string) (*NodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if config.NodeID == "" {
		return nil, fmt.Errorf("node_id is required")
	}
	if config.BindAddress == "" {
		return nil, fmt.Errorf("bind_address is required")
	}
	if config.DataDir == "" {
		return nil, fmt.Errorf("data_dir is required")
	}

	return &config, nil
}

// ToClusterConfig converts NodeConfig to ClusterConfig
func (nc *NodeConfig) ToClusterConfig() (*ClusterConfig, error) {
	// Parse durations
	heartbeatTimeout, err := time.ParseDuration(nc.Raft.HeartbeatTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid heartbeat_timeout: %w", err)
	}

	electionTimeout, err := time.ParseDuration(nc.Raft.ElectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid election_timeout: %w", err)
	}

	commitTimeout, err := time.ParseDuration(nc.Raft.CommitTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid commit_timeout: %w", err)
	}

	snapshotInterval, err := time.ParseDuration(nc.Raft.SnapshotInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot_interval: %w", err)
	}

	return &ClusterConfig{
		NodeID:            nc.NodeID,
		BindAddress:       nc.BindAddress,
		DataDir:           nc.DataDir,
		BootstrapExpect:   nc.BootstrapExpect,
		Peers:             nc.Peers,
		HeartbeatTimeout:  heartbeatTimeout,
		ElectionTimeout:   electionTimeout,
		CommitTimeout:     commitTimeout,
		SnapshotInterval:  snapshotInterval,
		SnapshotThreshold: nc.Raft.SnapshotThreshold,
	}, nil
}
