package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"ml-raft-control-plane/internal/api"
	"ml-raft-control-plane/internal/raft"
)

var (
	configFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "raft-node",
		Short: "Multi-cloud ML orchestration Raft node",
		Long:  `Raft-based control plane node for distributed ML task orchestration`,
		RunE:  runNode,
	}

	rootCmd.Flags().StringVar(&configFile, "config", "/etc/raft/config.json", "Path to configuration file")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runNode(cmd *cobra.Command, args []string) error {
	fmt.Println("=== Starting Raft Control Plane Node ===")

	// Load configuration
	fmt.Printf("Loading configuration from %s...\n", configFile)
	nodeConfig, err := raft.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Node ID: %s\n", nodeConfig.NodeID)
	fmt.Printf("Cloud: %s/%s\n", nodeConfig.CloudProvider, nodeConfig.Region)
	fmt.Printf("Bind Address: %s\n", nodeConfig.BindAddress)

	// Convert to cluster config
	clusterConfig, err := nodeConfig.ToClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to convert config: %w", err)
	}

	// Create Raft cluster
	fmt.Println("\nInitializing Raft cluster...")
	cluster, err := raft.NewRaftCluster(clusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}
	defer cluster.Shutdown()

	fmt.Println("✅ Raft cluster initialized")

	// Wait for leader election
	fmt.Println("\nWaiting for leader election...")
	if err := cluster.WaitForLeader(30 * time.Second); err != nil {
		return fmt.Errorf("failed to elect leader: %w", err)
	}

	leader := cluster.GetLeader()
	isLeader := cluster.IsLeader()

	if isLeader {
		fmt.Printf("✅ This node is the LEADER\n")
	} else {
		fmt.Printf("✅ Leader elected: %s\n", leader)
	}

	// Print Raft stats
	fmt.Println("\nRaft Statistics:")
	for key, value := range cluster.GetStats() {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Start gRPC server
	fmt.Printf("\nStarting gRPC server on port %d...\n", nodeConfig.GRPC.Port)
	grpcServer, err := api.NewGRPCServer(nodeConfig.GRPC.Port, cluster)
	if err != nil {
		return fmt.Errorf("failed to create gRPC server: %w", err)
	}

	// Start server in goroutine
	go func() {
		if err := grpcServer.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "gRPC server error: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("✅ gRPC server listening on %s\n", grpcServer.GetAddr())

	fmt.Println("\n=== Node Running ===")
	fmt.Println("Press Ctrl+C to shutdown")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n\nShutting down gracefully...")
	grpcServer.Stop()

	return nil
}
