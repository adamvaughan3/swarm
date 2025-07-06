package main

import (
	"flag"
	"strings"
	"swarm"
	"time"
)

func main() {
	port := flag.Int("port", 50051, "local port to bind the server")
	peers := flag.String("peers", "", "comma-separated list of peer addresses")
	id := flag.String("id", "node1", "unique node identifier")
	flag.Parse()

	parts := strings.Split(*peers, ",")

	var peerList []string
	for _, part := range parts {
		peerList = append(peerList, strings.TrimSpace(part))
	}

	server := swarm.NewServer(*id, *port)
	manager := swarm.NewConnectionManager(server, peerList, 5*time.Second, 10*time.Second)
	manager.Start()

	select {} // Run forever
}
