package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

// requestInfo 在 RouterMux 和 Service 之间传递“这次请求命中了谁”。
// 用指针是为了让 Service 能把 backend 写回，RouterMux 读出来。
type requestInfo struct {
	router  string
	service string
	backend string
}

type requestInfoKey struct{}

// statusRecorder 包装 ResponseWriter，捕获状态码。
// 同时保留 Flusher 能力，避免破坏 SSE / 流式响应。
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// logAccess 输出一行结构化访问日志。
func logAccess(r *http.Request, rec *statusRecorder, d time.Duration, info *requestInfo) {
	backend := info.backend
	if backend == "" {
		backend = "-"
	}
	log.Printf("access method=%s path=%q remote=%s router=%s service=%s backend=%s status=%d duration=%s",
		r.Method,
		r.URL.Path,
		r.RemoteAddr,
		info.router,
		info.service,
		backend,
		rec.status,
		d.Round(time.Microsecond),
	)
}

// withRequestInfo 把 requestInfo 塞进 context。
func withRequestInfo(ctx context.Context, info *requestInfo) context.Context {
	return context.WithValue(ctx, requestInfoKey{}, info)
}

// requestInfoFrom 从 context 取出 requestInfo，可能为 nil。
func requestInfoFrom(ctx context.Context) *requestInfo {
	info, _ := ctx.Value(requestInfoKey{}).(*requestInfo)
	return info
}
