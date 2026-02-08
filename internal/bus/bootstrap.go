package bus

import (
	"fmt"
	"log/slog"
	"strings"
)

type BootstrapOptions struct {
	MaxInFlight int
	Logger      *slog.Logger
	Component   string
}

func StartInproc(opts BootstrapOptions) (*Inproc, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	component := strings.TrimSpace(opts.Component)
	opts.Logger.Debug("bus_config",
		"component", component,
		"max_inflight", opts.MaxInFlight,
	)
	inprocBus, err := NewInproc(InprocOptions{
		MaxInFlight: opts.MaxInFlight,
		Logger:      opts.Logger,
	})
	if err != nil {
		if component == "" {
			return nil, fmt.Errorf("init inproc bus: %w", err)
		}
		return nil, fmt.Errorf("init %s inproc bus: %w", component, err)
	}
	opts.Logger.Info("bus_ready",
		"component", component,
		"mode", "inproc",
		"max_inflight", opts.MaxInFlight,
	)
	return inprocBus, nil
}
