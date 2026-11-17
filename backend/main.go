package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
)

type Response struct {
	Service string `json:"service"`
	Path    string `json:"path"`
	Method  string `json:"method"`
}

func main() {
	// 通过 flag 指定端口和名字
	addr := flag.String("addr", ":9001", "listen address")
	name := flag.String("name", "backend-9001", "service name in response")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Service: *name,
			Path:    r.URL.Path,
			Method:  r.Method,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(resp)
	})
	log.Printf("%s listening on %s", *name, *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}
