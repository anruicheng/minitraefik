package main

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

const labelPrefix = "mini.traefik."

// DockerProvider 监听 Docker 事件，从容器 label 构建 Config 并热更新。
type DockerProvider struct {
	cli *client.Client
	ep  *EntryPoint
}

func newDockerProvider(ep *EntryPoint) (*DockerProvider, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &DockerProvider{cli: cli, ep: ep}, nil
}

func (p *DockerProvider) rebuild(ctx context.Context) error {
	list, err := p.cli.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	cfg := &Config{
		Routers:  make(map[string]*RouterConfig),
		Services: make(map[string]*ServiceConfig),
	}

	for _, c := range list.Items {
		p.parseContainerLabels(c, cfg)
	}

	if len(cfg.Routers) == 0 || len(cfg.Services) == 0 {
		log.Printf("docker provider: no routable containers found, keeping current handler")
		return nil
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config from docker labels: %w", err)
	}

	mux, err := NewRouterMux(cfg)
	if err != nil {
		return fmt.Errorf("build router mux: %w", err)
	}

	p.ep.UpdateHandler(mux)
	log.Printf("docker provider: config updated (%d routers, %d services)",
		len(cfg.Routers), len(cfg.Services))
	return nil
}

func (p *DockerProvider) parseContainerLabels(c container.Summary, cfg *Config) {
	ip := containerIP(c)
	type svcDecl struct {
		port int
	}
	svcDecls := map[string]*svcDecl{}

	for k, v := range c.Labels {
		if !strings.HasPrefix(k, labelPrefix) {
			continue
		}
		rest := strings.TrimPrefix(k, labelPrefix)
		parts := strings.SplitN(rest, ".", 3)
		if len(parts) != 3 {
			continue
		}
		kind, name, field := parts[0], parts[1], parts[2]
		if name == "" || field == "" {
			continue
		}

		switch kind {
		case "router":
			if cfg.Routers[name] == nil {
				cfg.Routers[name] = &RouterConfig{}
			}
			switch field {
			case "rule":
				cfg.Routers[name].Rule = v
			case "service":
				cfg.Routers[name].Service = v
			}
		case "service":
			if cfg.Services[name] == nil {
				cfg.Services[name] = &ServiceConfig{}
			}
			if svcDecls[name] == nil {
				svcDecls[name] = &svcDecl{}
			}
			if field == "port" {
				port, err := strconv.Atoi(v)
				if err != nil {
					log.Printf("docker provider: invalid port %q in label %q", v, k)
					continue
				}
				svcDecls[name].port = port
			}
		}
	}

	if ip == "" {
		return
	}
	for svcName, decl := range svcDecls {
		if decl.port == 0 {
			continue
		}
		url := fmt.Sprintf("http://%s:%d", ip, decl.port)
		if !slices.Contains(cfg.Services[svcName].Servers, url) {
			cfg.Services[svcName].Servers = append(cfg.Services[svcName].Servers, url)
		}
	}
}

func containerIP(c container.Summary) string {
	if c.NetworkSettings == nil {
		return ""
	}
	for name, ns := range c.NetworkSettings.Networks {
		if name == "bridge" {
			continue
		}
		if ns.IPAddress.IsValid() {
			return ns.IPAddress.String()
		}
	}
	return ""
}

func (p *DockerProvider) Start(ctx context.Context) (stop func(), err error) {
	if err := p.rebuild(ctx); err != nil {
		log.Printf("docker provider: initial build failed: %v", err)
	}

	eventsResult := p.cli.Events(ctx, client.EventsListOptions{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case msg, ok := <-eventsResult.Messages:
				if !ok {
					return
				}
				if msg.Type != events.ContainerEventType {
					continue
				}
				switch msg.Action {
				case events.ActionStart,
					events.ActionDie,
					events.ActionStop,
					events.ActionKill:
					if err := p.rebuild(context.Background()); err != nil {
						log.Printf("docker provider: rebuild failed: %v", err)
					}
				}
			case err, ok := <-eventsResult.Err:
				if !ok {
					return
				}
				if err != nil {
					log.Printf("docker provider: event error: %v", err)
				}
			}
		}
	}()

	return func() {
		<-done
		_ = p.cli.Close()
	}, nil
}
