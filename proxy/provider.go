package main

import (
	"context"
	"fmt"
	"net/http"
)

type provider interface {
	Bootstrap() (http.Handler, error)
	Start(ctx context.Context, ep *EntryPoint) (stop func(), err error)
}

func newProvider(name, configPath string) (provider, error) {
	switch name {
	case "file":
		return &fileProvider{path: configPath}, nil
	case "docker":
		return &dockerProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q, want file or docker", name)
	}
}

// ---- file ----

type fileProvider struct {
	path string
}

func (p *fileProvider) Bootstrap() (http.Handler, error) {
	cfg, err := LoadConfig(p.path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return NewRouterMux(cfg)
}

func (p *fileProvider) Start(ctx context.Context, ep *EntryPoint) (func(), error) {
	return WatchConfig(p.path, ep)
}

// ---- docker ----

type dockerProvider struct{}

func (p *dockerProvider) Bootstrap() (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "docker provider not ready yet", http.StatusServiceUnavailable)
	}), nil
}

func (p *dockerProvider) Start(ctx context.Context, ep *EntryPoint) (func(), error) {
	dp, err := newDockerProvider(ep)
	if err != nil {
		return nil, err
	}
	return dp.Start(ctx)
}
