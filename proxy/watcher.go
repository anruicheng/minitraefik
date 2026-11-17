// proxy/watcher.go
package main

import (
	"github.com/fsnotify/fsnotify"
	"log"
)

// WatchConfig 监听 configPath 的变化，变化时重新加载配置并热更新 handler。
// 返回一个 stop 函数用于优雅关闭。
func WatchConfig(path string, ep *EntryPoint) (stop func(), err error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := watcher.Add(path); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// 只关心写入和创建事件（编辑器保存文件时行为不一，两种都接）
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				reload(path, ep)

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()

	return func() {
		_ = watcher.Close()
		<-done
	}, nil
}

// reload
func reload(path string, ep *EntryPoint) {
	log.Printf("config %q changed, reloading...", path)

	cfg, err := LoadConfig(path)
	if err != nil {
		log.Printf("reload: %v", err)
		return
	}
	if err := cfg.Validate(); err != nil {
		log.Printf("reload: invalid config: %v", err)
		return
	}

	mux, err := NewRouterMux(cfg)
	if err != nil {
		log.Printf("reload: build mux: %v", err)
		return
	}

	ep.UpdateHandler(mux)
	log.Printf("reload: done")
}
