package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	configPath := flag.String("config", "config.json", "path to dynamic config file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	mux, err := NewRouterMux(cfg)
	if err != nil {
		log.Fatalf("build router mux: %v", err)
	}

	ep := NewEntryPoint("web", ":8080", mux)
	if err := ep.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("entrypoint %q: %v", ep.Name, err)
	}
}
