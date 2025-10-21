package api

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"ml-raft-control-plane/internal/raft"
	pb "ml-raft-control-plane/pkg/proto"
)

// GRPCServer manages the gRPC server
type GRPCServer struct {
	server      *grpc.Server
	listener    net.Listener
	taskService *TaskServiceServer
	nodeService *NodeServiceServer
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(port int, cluster *raft.RaftCluster) (*GRPCServer, error) {
	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxConcurrentStreams(1000),
	}

	server := grpc.NewServer(opts...)

	// Create and register services
	taskService := NewTaskServiceServer(cluster)
	nodeService := NewNodeServiceServer(cluster)

	pb.RegisterTaskServiceServer(server, taskService)
	pb.RegisterNodeServiceServer(server, nodeService)

	return &GRPCServer{
		server:      server,
		listener:    listener,
		taskService: taskService,
		nodeService: nodeService,
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	fmt.Printf("Starting gRPC server on %s\n", s.listener.Addr().String())
	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() {
	fmt.Println("Stopping gRPC server...")
	s.server.GracefulStop()
}

// GetAddr returns the server address
func (s *GRPCServer) GetAddr() string {
	return s.listener.Addr().String()
}
