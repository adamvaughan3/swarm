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

type Server struct {
	pb.UnimplementedPingPongServer
	selfID      string
	selfPort    int
	mu          sync.Mutex
	pongConfirm map[string]chan struct{}
}

func NewServer(id string, port int) *Server {
	return &Server{
		selfID:      id,
		selfPort:    port,
		pongConfirm: make(map[string]chan struct{}),
	}
}

func StartPingPongServer(id string, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPingPongServer(grpcServer, NewServer(id, port))
	log.Printf("Listening on %d", port)
	return grpcServer.Serve(lis)
}

func (s *Server) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract peer")
	}
	host, _, err := net.SplitHostPort(peerInfo.Addr.String())
	if err != nil {
		return nil, fmt.Errorf("invalid peer address")
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
			Id:          s.selfID,
			ListenPort:  int32(s.selfPort),
			HandshakeId: req.HandshakeId, // pass back same handshake ID
		})
		if err != nil {
			log.Printf("Failed to send Pong to %s: %v", target, err)
		}
	}()

	return &pb.PingResponse{Message: "pong"}, nil
}

func (s *Server) Pong(ctx context.Context, req *pb.PongRequest) (*pb.PongResponse, error) {
	s.mu.Lock()
	ch, ok := s.pongConfirm[req.HandshakeId]
	if ok {
		close(ch)
		delete(s.pongConfirm, req.HandshakeId)
	}
	s.mu.Unlock()

	if !ok {
		log.Printf("[%s] No pending handshake found for %s", s.selfID, req.HandshakeId)
	}
	return &pb.PongResponse{Message: "ack"}, nil
}

func (s *Server) PingWithHandshake(peerAddr string) bool {
	handshakeID := fmt.Sprintf("%d", time.Now().UnixNano())

	ch := make(chan struct{})
	s.mu.Lock()
	s.pongConfirm[handshakeID] = ch
	s.mu.Unlock()

	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[%s] Dial failed: %v", s.selfID, err)
		return false
	}
	defer conn.Close()

	client := pb.NewPingPongClient(conn)
	_, err = client.Ping(context.Background(), &pb.PingRequest{
		Id:          s.selfID,
		ListenPort:  int32(s.selfPort),
		HandshakeId: handshakeID,
	})
	if err != nil {
		log.Printf("[%s] Ping RPC failed: %v", s.selfID, err)
		return false
	}

	select {
	case <-ch:
		// log.Printf("[%s] Successful handshake with %s", s.selfID, peerAddr)
		return true
	case <-time.After(3 * time.Second):
		log.Printf("[%s] No Pong from %s within timeout", s.selfID, peerAddr)
		s.mu.Lock()
		delete(s.pongConfirm, handshakeID)
		s.mu.Unlock()
		return false
	}
}
