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