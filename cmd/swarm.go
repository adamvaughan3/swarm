package main

import (
	"flag"
	"strings"
	"swarm"
)

func main() {
	port := flag.Int("port", 50051, "local port to bind the server")
	peers := flag.String("peers", "127.0.0.1:50052, 127.0.0.1:50053", "comma-separated list of peer addresses")
	flag.Parse()

	parts := strings.Split(*peers, ",")
	var peerList []string
	for _, part := range parts {
		peerList = append(peerList, strings.TrimSpace(part))
	}

	// manager := swarm.NewConnectionManager(*id, *port, peerList, swarm.NewEventBus(), 5*time.Second, 2*time.Second)
	// manager.Start()
	node := swarm.NewNode(*port, peerList)
	node.Start()

	select {}
}
