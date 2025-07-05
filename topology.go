package swarm

import (
	"container/heap"
)

type Topology map[string]map[string]int

type Path struct {
	Cost int
	Node string
	Prev *Path
}

// Priority queue for Dijkstra
type PriorityQueue []*Path

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].Cost < pq[j].Cost }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*Path))
}
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

func Dijkstra(topology Topology, start string) map[string][]string {
	visited := map[string]bool{}
	dist := map[string]int{}
	prev := map[string]string{}

	for node := range topology {
		dist[node] = 1<<31 - 1 // infinity
	}
	dist[start] = 0

	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Path{Cost: 0, Node: start})

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*Path)
		u := current.Node
		if visited[u] {
			continue
		}
		visited[u] = true

		for neighbor, cost := range topology[u] {
			alt := dist[u] + cost
			if alt < dist[neighbor] {
				dist[neighbor] = alt
				prev[neighbor] = u
				heap.Push(pq, &Path{Cost: alt, Node: neighbor})
			}
		}
	}
	return reconstructPaths(prev, start)
}

func reconstructPaths(prev map[string]string, start string) map[string][]string {
	paths := make(map[string][]string)
	for dest := range prev {
		var path []string
		for at := dest; at != ""; at = prev[at] {
			path = append([]string{at}, path...)
		}
		if len(path) > 0 && path[0] != start {
			path = append([]string{start}, path...)
		}
		paths[dest] = path
	}
	return paths
}
