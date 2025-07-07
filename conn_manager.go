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

const (
	CONN_RETRY_COUNT int = 3
)

type ConnectionManager struct {
	server        *Server
	addressBook   []string
	peerInfo      map[string]*PeerInfo
	activeConns   map[string]*grpc.ClientConn
	eventBus      *EventBus
	mu            sync.Mutex
	recheckPeriod time.Duration
	pingInterval  time.Duration
}

func NewConnectionManager(id string, listenPort int, peers []string, eventBus *EventBus, recheckPeriod, pingInterval time.Duration) *ConnectionManager {
	return &ConnectionManager{
		server:        NewServer(id, listenPort, eventBus),
		addressBook:   peers,
		peerInfo:      make(map[string]*PeerInfo),
		activeConns:   make(map[string]*grpc.ClientConn),
		eventBus:      eventBus,
		recheckPeriod: recheckPeriod,
		pingInterval:  pingInterval,
	}
}

func (cm *ConnectionManager) Start() {
	go func() {
		lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cm.server.selfPort))
		grpcServer := grpc.NewServer(
			grpc.StatsHandler(NewConnStatsHandler(cm.eventBus)),
		)
		pb.RegisterPingPongServer(grpcServer, cm.server)
		log.Printf("Listening on %d", cm.server.selfPort)
		grpcServer.Serve(lis)
	}()

	for _, peer := range cm.addressBook {
		go cm.monitorPeer(peer)
	}
}

func (cm *ConnectionManager) monitorPeer(peerAddr string) {
	for {
		cm.mu.Lock()
		_, connected := cm.activeConns[peerAddr]
		cm.mu.Unlock()

		if !connected {
			info, success := cm.server.TestConn(peerAddr)
			if success {
				conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				if err != nil {
					log.Printf("[CM] Failed to dial after handshake: %v", err)
					continue
				}
				cm.mu.Lock()
				cm.activeConns[peerAddr] = conn
				cm.peerInfo[peerAddr] = info
				cm.mu.Unlock()

				log.Printf("[CM] Connected to %s", peerAddr)

				cm.eventBus.Publish(NodeConnectedEvent{PeerInfo: *info})

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
		for retry := range CONN_RETRY_COUNT {
			info, ok := cm.server.TestConn(peerAddr)
			if !ok {
				if retry != CONN_RETRY_COUNT-1 {
					log.Printf("[CM] Connection to %s failed ping, %d retries left", peerAddr, CONN_RETRY_COUNT-retry-1)
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("[CM] Connection to %s failed ping, closing connection", peerAddr)
				cm.mu.Lock()
				if conn, exists := cm.activeConns[peerAddr]; exists {
					conn.Close()
					delete(cm.activeConns, peerAddr)

					// Failed connection test, use last known peer info in event
					cm.eventBus.Publish(NodeDisconnectedEvent{PeerInfo: *cm.peerInfo[peerAddr]})
				}
				cm.mu.Unlock()
				return
			} else {
				log.Printf("[CM] Heartbeat successful to %s", peerAddr)
				cm.peerInfo[peerAddr] = info
				break
			}
		}
	}
}
