package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 是整份动态配置的根
type Config struct {
	Routers  map[string]*RouterConfig  `json:"routers"`
	Services map[string]*ServiceConfig `json:"services"`
}

// RouterConfig 是 Router 的数据描述，通过名字引用 Service。
type RouterConfig struct {
	Rule     string `json:"rule"`
	Service  string `json:"service"`            // 引用 Config.Services 中的 key
	Priority int    `json:"priority,omitempty"` // 0 表示使用规则长度作为优先级
}

// ServiceConfig 是 Service 的数据描述。
type ServiceConfig struct {
	Servers []string `json:"servers"` // 后端 URL 列表
}

// LoadConfig 从文件读取 JSON 配置。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &cfg, nil
}

// Validate 检查 Router 引用的 Service 是否存在，以及 Service 是否有后端。
func (c *Config) Validate() error {
	if len(c.Services) == 0 {
		return fmt.Errorf("config: no services defined")
	}
	for name, svc := range c.Services {
		if len(svc.Servers) == 0 {
			return fmt.Errorf("service %q: no servers", name)
		}
	}
	if len(c.Routers) == 0 {
		return fmt.Errorf("config: no routers defined")
	}
	for name, rt := range c.Routers {
		if rt.Rule == "" {
			return fmt.Errorf("router %q: empty rule", name)
		}
		if _, ok := c.Services[rt.Service]; !ok {
			return fmt.Errorf("router %q: references unknown service %q", name, rt.Service)
		}
	}
	return nil
}
