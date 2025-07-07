package main

import (
	"flag"
	"strings"
	"swarm"
)

func main() {
	port := flag.Int("port", 50051, "local port to bind the server")
	peers := flag.String("peers", "", "comma-separated list of peer addresses")
	// id := flag.String("id", "node1", "unique node identifier")
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
