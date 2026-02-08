package bus

import (
	"context"
	"fmt"
)

type Router struct {
	bus *Inproc
}

func NewRouter(bus *Inproc) (*Router, error) {
	if bus == nil {
		return nil, fmt.Errorf("bus is required")
	}
	return &Router{bus: bus}, nil
}

func (r *Router) Handle(topic string, handler HandlerFunc) error {
	if r == nil || r.bus == nil {
		return fmt.Errorf("router is not initialized")
	}
	return r.bus.Subscribe(topic, handler)
}

func (r *Router) Publish(ctx context.Context, msg BusMessage) error {
	if r == nil || r.bus == nil {
		return fmt.Errorf("router is not initialized")
	}
	return r.bus.Publish(ctx, msg)
}
