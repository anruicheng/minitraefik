package main

import (
	"log"
	"net/http"
)

func main() {
	// user 业务
	svcUser, err := NewService("svc-user", []string{
		"http://127.0.0.1:9001",
		"http://127.0.0.1:9002",
	})
	if err != nil {
		log.Fatalf("build svc-user: %v", err)
	}

	// order 业务
	svcOrder, err := NewService("svc-order", []string{
		"http://127.0.0.1:9003",
		"http://127.0.0.1:9004",
	})
	if err != nil {
		log.Fatalf("build svc-order: %v", err)
	}

	// 按路径前缀分流到两组业务
	routerUser, err := NewRouter("user", "PathPrefix(`/users`)", svcUser)
	if err != nil {
		log.Fatalf("build router user: %v", err)
	}

	routerOrder, err := NewRouter("order", "PathPrefix(`/orders`)", svcOrder)
	if err != nil {
		log.Fatalf("build router order: %v", err)
	}

	mux := NewRouterMux([]*Router{routerUser, routerOrder})

	ep := NewEntryPoint("web", ":8080", mux)
	if err := ep.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("entrypoint %q: %v", ep.Name, err)
	}
}
