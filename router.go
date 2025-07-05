package swarm

type RoutingTable struct {
	Owner  string
	Routes map[string][]string
}

func (rt *RoutingTable) Update(topology Topology) {
	calculated := Dijkstra(topology, rt.Owner)
	if calculated != nil {
		rt.Routes = calculated
	}
}

func (rt *RoutingTable) Route(dest string) string {
	path := rt.Routes[dest]
	if len(path) == 0 {
		return dest
	}
	return path[0]
}

func (rt *RoutingTable) Clear() {
	rt.Routes = make(map[string][]string)
}
