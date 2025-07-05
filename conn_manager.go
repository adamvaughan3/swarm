package swarm

import (
	"context"
	"fmt"
	"log"
	"swarm/proto"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/emptypb"
)

type NodeConnection struct {
	Addr       string
	ClientConn *grpc.ClientConn
	cancelFunc context.CancelFunc
	isHealthy  bool
}

type NodeConnectionManager struct {
	mu                sync.RWMutex
	connections       map[string]*NodeConnection
	pendingReconnects map[string]struct{}
	stopChans         map[string]chan struct{}
	dialOptions       []grpc.DialOption
}

func NewNodeConnectionManager() *NodeConnectionManager {
	return &NodeConnectionManager{
		connections:       make(map[string]*NodeConnection),
		pendingReconnects: make(map[string]struct{}),
		stopChans:         make(map[string]chan struct{}),
		dialOptions: []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			}),
		},
	}
}

func (m *NodeConnectionManager) AddNode(addr string) error {
	m.mu.Lock()
	if _, exists := m.connections[addr]; exists {
		m.mu.Unlock()
		log.Printf("connection already exists")
		return nil
	}
	m.mu.Unlock()

	_, cancel := context.WithCancel(context.Background())
	conn, err := grpc.NewClient(addr, m.dialOptions...)
	if err != nil {
		cancel()
		return err
	}

	// Verify node is responsive using Ping
	client := proto.NewNodeControlClient(conn)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_, err = client.Ping(pingCtx, &emptypb.Empty{})
	pingCancel()
	if err != nil {
		conn.Close()
		cancel()
		return fmt.Errorf("ping check to %s failed: %w", addr, err)
	}

	node := &NodeConnection{
		Addr:       addr,
		ClientConn: conn,
		cancelFunc: cancel,
		isHealthy:  true,
	}

	m.mu.Lock()
	m.connections[addr] = node
	delete(m.pendingReconnects, addr)
	m.mu.Unlock()

	go m.monitorHealth(node)
	return nil
}

func (m *NodeConnectionManager) RemoveNode(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if node, ok := m.connections[addr]; ok {
		node.cancelFunc()
		node.ClientConn.Close()
		delete(m.connections, addr)
	}
	if ch, ok := m.stopChans[addr]; ok {
		close(ch)
		delete(m.stopChans, addr)
	}
	delete(m.pendingReconnects, addr)
}

func (m *NodeConnectionManager) monitorHealth(node *NodeConnection) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	client := proto.NewNodeControlClient(node.ClientConn)
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.Ping(ctx, &emptypb.Empty{})
		cancel()
		if err != nil {
			log.Printf("Heartbeat failed to %s: %v", node.Addr, err)
			m.RemoveNode(node.Addr)
			m.mu.Lock()
			if _, exists := m.pendingReconnects[node.Addr]; !exists {
				m.pendingReconnects[node.Addr] = struct{}{}
				ch := make(chan struct{})
				m.stopChans[node.Addr] = ch
				go m.retryConnection(node.Addr, ch)
			}
			m.mu.Unlock()
			return
		}
	}
}

func (m *NodeConnectionManager) retryConnection(addr string, stop chan struct{}) {
	for {
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
			log.Printf("Retrying connection to %s", addr)
			if err := m.AddNode(addr); err == nil {
				log.Printf("Successfully reconnected to %s", addr)
				return
			} else {
				log.Printf("Reconnect failed to %s: %v", addr, err)
			}
		}
	}
}
