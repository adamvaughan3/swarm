package swarm

type Node struct {
	Id        string
	Neighbors map[string]*Neighbor
	Routes    map[string]string // destId -> nextHopId
}

type Neighbor struct {
	Id      string
	Address string
	Cost    int
}

func (n *Node) Heartbeat() {
	for id, neighbor := range n.Neighbors {
		go func(neighbor *Neighbor) {
			if !TestConn(neighbor.Address) {
				delete(n.Neighbors, id)
				n.gossip()
			}
		}(neighbor)
	}
}

// TODO - Send Gossip message to random neighbors
func (n *Node) gossip() {
}

// TODO - Implement as part of gRPC
func TestConn(neighbor string) bool {
	return true
}
