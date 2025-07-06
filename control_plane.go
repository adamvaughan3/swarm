package swarm

import (
	"context"
	"fmt"
	"log"
	"net"
	"swarm/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	REGISTRATION_BACKOFF = 5
)

type NodeControlServer struct {
	proto.UnimplementedNodeControlServer
	connMgr *NodeConnectionManager
}

func NewNodeControlServer(mgr *NodeConnectionManager) *NodeControlServer {
	return &NodeControlServer{connMgr: mgr}
}

func (s *NodeControlServer) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func StartServer(port int, mgr *NodeConnectionManager) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	proto.RegisterNodeControlServer(grpcServer, NewNodeControlServer(mgr))
	log.Printf("Listening on %d", port)
	return grpcServer.Serve(lis)
}

func (s *NodeControlServer) RegisterNode(ctx context.Context, info *proto.NodeInfo) (*proto.RegisterResponse, error) {
	var address string
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		address, _, _ = net.SplitHostPort(p.Addr.String())
		address = fmt.Sprintf("%s:%d", address, info.Port)
	}
	log.Printf("Register request from %s (%s)", info.Id, address)
	if err := s.connMgr.AddNode(address); err != nil {
		return &proto.RegisterResponse{Success: false, Message: err.Error()}, nil
	}
	return &proto.RegisterResponse{Success: true, Message: "Node registered"}, nil
}

func RegisterWithPeer(peerAddr string, myId string, myPort int) {
	for {
		log.Printf("Attempting to register with peer %s", peerAddr)
		conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			client := proto.NewNodeControlClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err = client.RegisterNode(ctx, &proto.NodeInfo{Id: myId, Port: int32(myPort)})
			cancel()
			conn.Close()
			if err == nil {
				log.Printf("Successfully registered with peer %s", peerAddr)
				return
			}
		}
		log.Printf("Failed to register with peer %s: %v", peerAddr, err)
		time.Sleep(time.Duration(REGISTRATION_BACKOFF) * time.Second)
	}
}
