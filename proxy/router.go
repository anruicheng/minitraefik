package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Router 把一条规则绑定到一个 Service。
type Router struct {
	Name        string
	Rule        string
	ServiceName string
	matcher     *ruleMatcher
	priority    int
}

func NewRouter(name string, cfg *RouterConfig) (*Router, error) {
	m, err := parseRule(cfg.Rule)
	if err != nil {
		return nil, fmt.Errorf("router %q: %w", name, err)
	}

	priority := cfg.Priority
	if priority == 0 {
		priority = len(cfg.Rule) // Traefik 默认：规则越长优先级越高
	}
	return &Router{Name: name, Rule: cfg.Rule, ServiceName: cfg.Service, matcher: m, priority: priority}, nil
}

// ruleMatcher 是一组 AND 关系的谓词。
// 只支持 Host(...) && PathPrefix(...)
type ruleMatcher struct {
	preds []func(*http.Request) bool
}

func (m *ruleMatcher) Match(r *http.Request) bool {
	for _, p := range m.preds {
		if !p(r) {
			return false
		}
	}
	return true
}

// parseRule 按 && 切分，逐段识别 Host / PathPrefix。
func parseRule(rule string) (*ruleMatcher, error) {
	parts := strings.Split(rule, "&&")
	m := &ruleMatcher{}

	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			return nil, fmt.Errorf("empty segment in rule %q", rule)
		}

		switch {
		case strings.HasPrefix(part, "Host("):
			val, err := extractArg(part, "Host")
			if err != nil {
				return nil, err
			}
			host := val // 显式拷贝，避免循环变量捕获（go 1.21 语义）
			m.preds = append(m.preds, func(r *http.Request) bool {
				return r.Host == host
			})

		case strings.HasPrefix(part, "PathPrefix("):
			val, err := extractArg(part, "PathPrefix")
			if err != nil {
				return nil, err
			}
			prefix := val
			m.preds = append(m.preds, func(r *http.Request) bool {
				return strings.HasPrefix(r.URL.Path, prefix)
			})

		default:
			return nil, fmt.Errorf("unsupported matcher: %q", part)
		}
	}

	if len(m.preds) == 0 {
		return nil, fmt.Errorf("no matcher in rule %q", rule)
	}
	return m, nil
}

// extractArg 从 PathPrefix(`/users`) 里取出 /users
func extractArg(expr, name string) (string, error) {
	prefix := name + "("
	if !strings.HasPrefix(expr, prefix) || !strings.HasSuffix(expr, ")") {
		return "", fmt.Errorf("malformed %s matcher: %q", name, expr)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(expr, prefix), ")")
	inner = strings.TrimSpace(inner)
	inner = strings.Trim(inner, "`\"'")
	if inner == "" {
		return "", fmt.Errorf("empty arg in %s matcher", name)
	}
	return inner, nil
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
	for _, rt := range m.routers {
		if rt.matcher.Match(r) {
			svc, ok := m.services[rt.ServiceName]
			if !ok {
				http.Error(w, "internal error: service not found", http.StatusInternalServerError)
				return
			}
			svc.ServeHTTP(w, r)
			return
		}
	}
	m.notFound.ServeHTTP(w, r)
}
