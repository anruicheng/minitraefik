package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"minitraefik/proxy/rules"
)

// Router 把一条规则绑定到一个 Service。
type Router struct {
	Name        string
	Rule        string
	ServiceName string
	matcher     rules.Node
	priority    int
}

func NewRouter(name string, cfg *RouterConfig) (*Router, error) {
	node, err := rules.Parse(cfg.Rule)
	if err != nil {
		return nil, fmt.Errorf("router %q: %w", name, err)
	}

	priority := cfg.Priority
	if priority == 0 {
		priority = len(cfg.Rule) // Traefik 默认：规则越长优先级越高
	}
	return &Router{Name: name, Rule: cfg.Rule, ServiceName: cfg.Service, matcher: node, priority: priority}, nil
}

// RouterMux 按声明顺序遍历 Router，第一个匹配的胜出。
type RouterMux struct {
	routers  []*Router
	services map[string]*Service
	notFound http.Handler
}

func NewRouterMux(cfg *Config) (*RouterMux, error) {
	services := make(map[string]*Service, len(cfg.Services))
	for name, v := range cfg.Services {
		svc, err := NewService(name, v.Servers)
		if err != nil {
			return nil, fmt.Errorf("service %q: %w", name, err)
		}
		services[name] = svc
	}

	routers := make([]*Router, 0, len(cfg.Routers))
	for name, routerConfig := range cfg.Routers {
		rt, err := NewRouter(name, routerConfig)
		if err != nil {
			return nil, err
		}
		routers = append(routers, rt)
	}

	sort.SliceStable(routers, func(i, j int) bool {
		if routers[i].priority != routers[j].priority {
			return routers[i].priority > routers[j].priority
		}
		return routers[i].Name < routers[j].Name
	})

	return &RouterMux{
		routers:  routers,
		services: services,
		notFound: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "404 page not found: no router matched", http.StatusNotFound)
		}),
	}, nil
}

func (m *RouterMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	info := &requestInfo{router: "<no-router>", service: ""}

	for _, rt := range m.routers {
		if rt.matcher.Match(r) {
			info.router = rt.Name
			info.service = rt.ServiceName
			svc, ok := m.services[rt.ServiceName]
			if !ok {
				http.Error(rec, "internal error: service not found", http.StatusInternalServerError)
				logAccess(r, rec, time.Since(start), info)
				observe(info, rec, time.Since(start))
				return
			}
			ctx := withRequestInfo(r.Context(), info)
			svc.ServeHTTP(rec, r.WithContext(ctx))

			logAccess(r, rec, time.Since(start), info)
			observe(info, rec, time.Since(start))
			return
		}
	}
	m.notFound.ServeHTTP(rec, r)
	logAccess(r, rec, time.Since(start), info)
	observe(info, rec, time.Since(start))
}
