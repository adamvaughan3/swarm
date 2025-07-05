package main

import (
	"flag"
	"log"
	"strings"
	"swarm"
	"time"
)

func main() {
	addr := flag.String("addr", ":50051", "address to bind the server")
	peers := flag.String("peers", "", "comma-separated list of peer addresses")
	id := flag.String("id", "node1", "unique node identifier")
	flag.Parse()

	connMgr := swarm.NewNodeConnectionManager()
	go func() {
		if err := swarm.StartServer(*addr, connMgr); err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(2 * time.Second)

	if *peers != "" {
		for _, peer := range strings.Split(*peers, ",") {
			peer = strings.TrimSpace(peer)
			go swarm.RegisterWithPeer(peer, *id, *addr)
		}
	}

	select {}
}
