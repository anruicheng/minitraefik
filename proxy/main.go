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

	// 启动文件监听（在 goroutine 中运行）
	stopWatcher, err := WatchConfig(*configPath, ep)
	if err != nil {
		log.Fatalf("watch config: %v", err)
	}

	// 优雅关闭信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("received signal, shutting down...")

		signal.Stop(sigCh)
		stopWatcher()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ep.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := ep.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("entrypoint %q: %v", ep.Name, err)
	}
	log.Println("proxy exited")
}
