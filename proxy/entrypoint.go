package main

import (
	"log"
	"net/http"
)

type EntryPoint struct {
	Name   string
	Addr   string
	server *http.Server
}

func NewEntryPoint(name, addr string, handler http.Handler) *EntryPoint {
	return &EntryPoint{
		Name: name,
		Addr: addr,
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

// Start 阻塞运行，直到 server 出错或关闭。
func (e *EntryPoint) Start() error {
	log.Printf("entrypoint %q listening on %s", e.Name, e.Addr)
	return e.server.ListenAndServe()
}
