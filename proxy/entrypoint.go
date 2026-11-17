package main

import (
	"context"
	"log"
	"net/http"
	"sync"
)

type EntryPoint struct {
	Name   string
	Addr   string
	server *http.Server

	mu      sync.RWMutex
	handler http.Handler //当前生效的handler
}

func NewEntryPoint(name, addr string, handler http.Handler) *EntryPoint {
	ep := &EntryPoint{
		Name:    name,
		Addr:    addr,
		handler: handler,
	}
	ep.server = &http.Server{
		Addr:    addr,
		Handler: ep,
	}
	return ep
}

// ServeHTTP 实现 http.Handler，每次请求时读取最新的 handler。
func (e *EntryPoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mu.RLock()
	h := e.handler
	e.mu.RUnlock()
	h.ServeHTTP(w, r)
}

func (e *EntryPoint) UpdateHandler(h http.Handler) {
	e.mu.Lock()
	e.handler = h
	e.mu.Unlock()
}

// Start 阻塞运行，直到 server 出错或关闭。
func (e *EntryPoint) Start() error {
	log.Printf("entrypoint %q listening on %s", e.Name, e.Addr)
	return e.server.ListenAndServe()
}

func (e *EntryPoint) Shutdown(ctx context.Context) error {
	log.Printf("entrypoint %q shutting down", e.Name)
	return e.server.Shutdown(ctx)
}
