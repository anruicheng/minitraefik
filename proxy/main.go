package main

import (
	"log"
	"net/http"
)

func main() {
	svc, err := NewService("backend", []string{
		"http://localhost:9001",
		"http://localhost:9002",
	})
	if err != nil {
		log.Fatalf("build service: %v", err)
	}

	ep := NewEntryPoint("web", ":8080", svc)
	if err := ep.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("entrypoint %q: %v", ep.Name, err)
	}
}
