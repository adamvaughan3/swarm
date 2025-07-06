package main

import (
	"flag"
	"log"
	"strings"
	"swarm"
	"time"
)

func main() {
	port := flag.Int("port", 50051, "local port to bind the server")
	peers := flag.String("peers", "", "comma-separated list of peer addresses")
	id := flag.String("id", "node1", "unique node identifier")
	flag.Parse()

	eventBus := swarm.NewEventBus()

	connMgr := swarm.NewNodeConnectionManager(eventBus)
	go func() {
		if err := swarm.StartServer(*port, connMgr); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(2 * time.Second)

	if *peers != "" {
		for _, peer := range strings.Split(*peers, ",") {
			peer = strings.TrimSpace(peer)
			go swarm.RegisterWithPeer(peer, *id, *port)
		}
	}

	select {}
}
