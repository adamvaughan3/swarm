package swarm

import (
	"fmt"
	"log"
	"net"
	pb "swarm/proto"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ConnectionManager struct {
	server        *Server
	peers         []string // List of all known peer addresses
	activeConns   map[string]*grpc.ClientConn
	mu            sync.Mutex
	recheckPeriod time.Duration
	pingInterval  time.Duration
}

func NewConnectionManager(server *Server, peers []string, recheckPeriod, pingInterval time.Duration) *ConnectionManager {
	return &ConnectionManager{
		server:        server,
		peers:         peers,
		activeConns:   make(map[string]*grpc.ClientConn),
		recheckPeriod: recheckPeriod,
		pingInterval:  pingInterval,
	}
}

func (cm *ConnectionManager) Start() {
	go func() {
		lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cm.server.selfPort))
		grpcServer := grpc.NewServer()
		pb.RegisterPingPongServer(grpcServer, cm.server)
		log.Printf("Listening on %d", cm.server.selfPort)
		grpcServer.Serve(lis)
	}()

	for _, peer := range cm.peers {
		go cm.monitorPeer(peer)
	}
}

func (cm *ConnectionManager) monitorPeer(peerAddr string) {
	for {
		cm.mu.Lock()
		_, connected := cm.activeConns[peerAddr]
		cm.mu.Unlock()

		if !connected {
			success := cm.server.PingWithHandshake(peerAddr)
			if success {
				conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					log.Printf("[CM] Failed to dial after handshake: %v", err)
					continue
				}
				cm.mu.Lock()
				cm.activeConns[peerAddr] = conn
				cm.mu.Unlock()

				log.Printf("[CM] Connected to %s", peerAddr)

				// Start a monitor loop for this connection
				go func() {
					cm.keepAlive(peerAddr)
				}()
			}
		}
		time.Sleep(cm.recheckPeriod)
	}
}

func (cm *ConnectionManager) keepAlive(peerAddr string) {
	ticker := time.NewTicker(cm.pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		ok := cm.server.PingWithHandshake(peerAddr)
		if !ok {
			log.Printf("[CM] Connection to %s failed ping, closing", peerAddr)
			cm.mu.Lock()
			if conn, exists := cm.activeConns[peerAddr]; exists {
				conn.Close()
				delete(cm.activeConns, peerAddr)
			}
			cm.mu.Unlock()
			return
		}
	}
}

// GetActiveConnections returns a copy of active connections
func (cm *ConnectionManager) GetActiveConnections() map[string]*grpc.ClientConn {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Return a copy to avoid external mutation
	copy := make(map[string]*grpc.ClientConn)
	for k, v := range cm.activeConns {
		copy[k] = v
	}
	return copy
}
