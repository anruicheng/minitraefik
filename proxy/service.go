package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

type Service struct {
	Name    string
	servers []*backendServer
	next    atomic.Uint64
}
type backendServer struct {
	url   *url.URL
	proxy *httputil.ReverseProxy
}

func NewService(name string, rawURLs []string) (*Service, error) {
	if len(rawURLs) == 0 {
		return nil, fmt.Errorf("service %q: no servers", name)
	}

	servers := make([]*backendServer, 0, len(rawURLs))
	for _, raw := range rawURLs {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("service %q: parse %q: %w", name, raw, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("service %q: invalid url %q, need scheme://host", name, raw)
		}

		p := httputil.NewSingleHostReverseProxy(u)
		p.ErrorHandler = newProxyErrorHandler(name, u)

		servers = append(servers, &backendServer{url: u, proxy: p})
	}

	return &Service{Name: name, servers: servers}, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idx := s.next.Add(1) - 1
	srv := s.servers[idx%uint64(len(s.servers))]
	srv.proxy.ServeHTTP(w, r)
}

func newProxyErrorHandler(serviceName string, u *url.URL) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("service %q -> %s: %v", serviceName, u.String(), err)
		http.Error(w, "bad gateway: backend unreachable", http.StatusBadGateway)
	}
}
