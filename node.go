package swarm

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	RECHECK_PERIOD = 5
	PING_INTERVAL  = 5
)

type Node struct {
	Id          string
	Port        int
	Neighbors   map[string]*Neighbor
	Routes      map[string]string // destId -> nextHopId
	connManager *ConnectionManager
}

type Neighbor struct {
	Id      string
	Address string
	Cost    int
}

func NewNode(port int, peers []string) *Node {
	id := uuid.New().String()
	bus := NewEventBus()
	return &Node{
		Id:          id,
		Port:        port,
		Neighbors:   make(map[string]*Neighbor),
		Routes:      make(map[string]string),
		connManager: NewConnectionManager(id, port, peers, bus, RECHECK_PERIOD*time.Second, PING_INTERVAL*time.Second),
	}
}

func (node *Node) Start() {
	node.connManager.Start()
	slog.Info("Successfully started node", slog.String("Id", node.Id), slog.Int("ListenPort", node.Port))
}
