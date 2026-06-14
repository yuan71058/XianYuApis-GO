package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用完整配置。
type Config struct {
	Cookies map[string]string `yaml:"cookies"` // 登录 Cookie
	Log     LogConfig         `yaml:"log"`     // 日志配置
	WS      WSConfig          `yaml:"ws"`      // WebSocket 配置
}

// LogConfig 日志配置。
type LogConfig struct {
	Level string `yaml:"level"` // debug / info / warn / error
	JSON  bool   `yaml:"json"`  // JSON 格式输出
}

// WSConfig WebSocket 配置。
type WSConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`  // 读取缓冲区
	WriteBufferSize int           `yaml:"write_buffer_size"` // 写入缓冲区
	Heartbeat       time.Duration `yaml:"heartbeat"`         // 心跳间隔
	TokenRefresh    time.Duration `yaml:"token_refresh"`     // Token 刷新间隔
}

// Default 返回默认配置。
func Default() *Config {
	return &Config{
		Log: LogConfig{Level: "info", JSON: true},
		WS: WSConfig{
			Heartbeat:       15 * time.Second,
			TokenRefresh:    10 * time.Minute,
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
		},
	}
}

// Load 从配置文件加载配置，环境变量覆盖，默认值兜底。
//
// 优先级: 环境变量 > 配置文件 > 默认值
func Load(configPath string) (*Config, error) {
	cfg := Default()

	// 从文件加载
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("config: read file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config: parse yaml: %w", err)
		}
	}

	// 环境变量覆盖
	if v := os.Getenv("XIANYU_COOKIE_UNB"); v != "" {
		if cfg.Cookies == nil {
			cfg.Cookies = make(map[string]string)
		}
		cfg.Cookies["unb"] = v
	}
	if v := os.Getenv("XIANYU_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}

	return cfg, nil
}
