package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "minitraefik_requests_total",
			Help: "Total number of requests handled by the gateway.",
		},
		[]string{"router", "service", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "minitraefik_request_duration_seconds",
			Help:    "Request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"router", "service"},
	)
)

// observe 记录一次请求的指标。
func observe(info *requestInfo, rec *statusRecorder, d time.Duration) {
	router := info.router
	if router == "" {
		router = "<no-router>"
	}
	service := info.service
	if service == "" {
		service = "<none>"
	}

	requestTotal.WithLabelValues(router, service, statusLabel(rec.status)).Inc()
	requestDuration.WithLabelValues(router, service).Observe(d.Seconds())
}

func statusLabel(code int) string {
	switch {
	case code < 200:
		return "1xx"
	case code < 300:
		return "2xx"
	case code < 400:
		return "3xx"
	case code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}

// metricsHandler 返回 /metrics 的 handler。
func metricsHandler() http.Handler {
	return promhttp.Handler()
}
