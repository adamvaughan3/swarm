package swarm

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	pb "swarm/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
)

const (
	RESPONSE_TIMEOUT = 3
)

type PeerInfo struct {
	Id   string
	Address string
}

type Server struct {
	pb.UnimplementedPingPongServer
	selfId      string
	selfPort    int
	eventBus    *EventBus
	mu          sync.Mutex
	pongConfirm map[string]chan *PeerInfo
}

func NewServer(id string, port int, eventBus *EventBus) *Server {
	return &Server{
		selfId:      id,
		selfPort:    port,
		eventBus:    eventBus,
		pongConfirm: make(map[string]chan *PeerInfo),
	}
}

func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	host, err := addressFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	target := net.JoinHostPort(host, fmt.Sprintf("%d", req.ListenPort))

	go func() {
		conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect back to %s: %v", target, err)
			return
		}
		defer conn.Close()

		client := pb.NewPingPongClient(conn)
		_, err = client.Pong(context.Background(), &pb.PongRequest{
			Id:          s.selfId,
			ListenPort:  int32(s.selfPort),
			HandshakeId: req.HandshakeId,
		})
		if err != nil {
			log.Printf("Failed to send Pong to %s: %v", target, err)
		}
	}()

	return &pb.PingResponse{Message: "pong"}, nil
}

func (s *Server) Pong(ctx context.Context, req *pb.PongRequest) (*pb.PongResponse, error) {
	host, err := addressFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	peerAddr := net.JoinHostPort(host, fmt.Sprintf("%d", req.ListenPort))

	s.mu.Lock()
	ch, ok := s.pongConfirm[req.HandshakeId]
	if ok {
		ch <- &PeerInfo{
			Id:   req.Id,
			Address: peerAddr,
		}
		close(ch)
		delete(s.pongConfirm, req.HandshakeId)
	}
	s.mu.Unlock()

	if !ok {
		log.Printf("[%s] No pending handshake found for %s", s.selfId, req.HandshakeId)
	}
	return &pb.PongResponse{Message: "ack"}, nil
}

func (s *Server) TestConn(peerAddr string) (*PeerInfo, bool) {
	handshakeId := fmt.Sprintf("%d", time.Now().UnixNano())

	ch := make(chan *PeerInfo)
	s.mu.Lock()
	s.pongConfirm[handshakeId] = ch
	s.mu.Unlock()

	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[%s] Dial failed: %v", s.selfId, err)
		return nil, false
	}
	defer conn.Close()

	client := pb.NewPingPongClient(conn)
	_, err = client.Ping(context.Background(), &pb.PingRequest{
		Id:          s.selfId,
		ListenPort:  int32(s.selfPort),
		HandshakeId: handshakeId,
	})
	if err != nil {
		return nil, false
	}

	select {
	case peerInfo := <-ch:
		return peerInfo, true
	case <-time.After(RESPONSE_TIMEOUT * time.Second):
		log.Printf("[%s] No Pong from %s within timeout", s.selfId, peerAddr)
		s.mu.Lock()
		delete(s.pongConfirm, handshakeId)
		s.mu.Unlock()
		return nil, false
	}
}

func addressFromCtx(ctx context.Context) (string, error) {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("failed to extract peer")
	}
	host, _, err := net.SplitHostPort(peerInfo.Addr.String())
	if err != nil {
		return "", fmt.Errorf("invalid peer address")
	}
	return host, nil
}
