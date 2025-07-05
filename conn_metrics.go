package swarm

import (
	"context"
	"time"

	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

type ctxKeyPingTiming struct{}

type pingTiming struct {
	start time.Time
}

type connStatsHandler struct {
	bus *EventBus
}

func NewConnStatsHandler(bus *EventBus) *connStatsHandler {
	return &connStatsHandler{bus: bus}
}

func (h *connStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	if info.FullMethodName == "/swarm.NodeControl/Ping" {
		return context.WithValue(ctx, ctxKeyPingTiming{}, pingTiming{start: time.Now()})
	}
	return ctx
}

func (h *connStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	switch s.(type) {
	case *stats.End:
		if timing, ok := ctx.Value(ctxKeyPingTiming{}).(pingTiming); ok {
			latency := time.Since(timing.start)
			addr := peerFromContext(ctx)

			// Publish the event
			h.bus.Publish(PingLatencyEvent{
				Id:      "default",
				Addr:    addr,
				Latency: latency,
			})
		}
	}
}

func (h *connStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *connStatsHandler) HandleConn(ctx context.Context, _ stats.ConnStats) {
	// no-op for connection events
}

// --- Helper to extract peer address ---

func peerFromContext(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return "unknown"
}
