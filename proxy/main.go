package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	providerName := flag.String("provider", "file", "config provider: file or docker")
	configPath := flag.String("config", "config.json", "path to dynamic config file (file provider only)")
	addr := flag.String("addr", ":8080", "entrypoint listen address")
	metricsAddr := flag.String("metrics-addr", ":8081", "metrics listen address")
	flag.Parse()

	p, err := newProvider(*providerName, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	initialHandler, err := p.Bootstrap()
	if err != nil {
		log.Fatal(err)
	}

	// metrics entrypoint 独立，不影响业务热更新
	metricsEP := NewEntryPoint("metrics", *metricsAddr, metricsHandler())
	go func() {
		if err := metricsEP.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics entrypoint: %v", err)
		}
	}()

	ep := NewEntryPoint("web", *addr, initialHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopProvider, err := p.Start(ctx, ep)
	if err != nil {
		log.Fatalf("start provider: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("received signal, shutting down...")
		signal.Stop(sigCh)

		cancel()
		stopProvider()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := ep.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := ep.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("entrypoint %q: %v", ep.Name, err)
	}
	log.Println("proxy exited")
}
