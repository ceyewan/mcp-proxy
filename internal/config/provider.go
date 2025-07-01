package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Provider 配置提供者实现
type Provider struct{}

// NewProvider 创建新的配置提供者
func NewProvider() interfaces.ConfigProvider {
	return &Provider{}
}

// Load 加载配置文件
func (p *Provider) Load(path string) (*interfaces.Config, error) {
	var data []byte
	var err error

	// 判断是否为 HTTP URL
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		data, err = p.loadFromURL(path)
	} else {
		data, err = p.loadFromFile(path)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 解析 JSON
	var config interfaces.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 设置默认值
	p.setDefaults(&config)

	return &config, nil
}

// loadFromFile 从文件加载配置
func (p *Provider) loadFromFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// loadFromURL 从 HTTP URL 加载配置
func (p *Provider) loadFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// setDefaults 设置默认值
func (p *Provider) setDefaults(config *interfaces.Config) {
	// 设置代理默认值
	if config.Proxy.Type == "" {
		config.Proxy.Type = interfaces.TransportTypeSSE
	}
	if config.Proxy.Options == nil {
		config.Proxy.Options = &interfaces.OptionsConfig{}
	}

	// 为每个服务器设置默认值
	for name, serverConfig := range config.Servers {
		if serverConfig.Options == nil {
			serverConfig.Options = &interfaces.OptionsConfig{}
		}

		// 继承代理的默认配置
		p.inheritProxyDefaults(serverConfig.Options, config.Proxy.Options)

		// 自动检测传输类型
		if serverConfig.Transport == "" {
			serverConfig.Transport = p.detectTransportType(serverConfig)
		}

		// 更新配置
		config.Servers[name] = serverConfig
	}
}

// inheritProxyDefaults 继承代理的默认配置
func (p *Provider) inheritProxyDefaults(serverOptions, proxyOptions *interfaces.OptionsConfig) {
	if serverOptions.AuthTokens == nil {
		serverOptions.AuthTokens = proxyOptions.AuthTokens
	}
	if serverOptions.PanicIfInvalid == nil {
		serverOptions.PanicIfInvalid = proxyOptions.PanicIfInvalid
	}
	if serverOptions.LogEnabled == nil {
		serverOptions.LogEnabled = proxyOptions.LogEnabled
	}
}

// detectTransportType 自动检测传输类型
func (p *Provider) detectTransportType(config interfaces.ServerConfig) string {
	if config.Command != "" {
		return interfaces.ClientTypeStdio
	}
	if config.URL != "" {
		if config.Transport == interfaces.ClientTypeStreamable {
			return interfaces.ClientTypeStreamable
		}
		return interfaces.ClientTypeSSE
	}
	return interfaces.ClientTypeStdio
}

// Validate 验证配置
func (p *Provider) Validate(config *interfaces.Config) error {
	if config == nil {
		return errors.New("config is nil")
	}

	// 验证代理配置
	if err := p.validateProxyConfig(&config.Proxy); err != nil {
		return fmt.Errorf("invalid proxy config: %w", err)
	}

	// 验证服务器配置
	for name, serverConfig := range config.Servers {
		if err := p.validateServerConfig(name, serverConfig); err != nil {
			return fmt.Errorf("invalid server config for %s: %w", name, err)
		}
	}

	return nil
}

// validateProxyConfig 验证代理配置
func (p *Provider) validateProxyConfig(config *interfaces.ProxyConfig) error {
	if config.Name == "" {
		return errors.New("name is required")
	}
	if config.Addr == "" {
		return errors.New("addr is required")
	}
	if config.BaseURL == "" {
		return errors.New("baseURL is required")
	}
	if config.Version == "" {
		return errors.New("version is required")
	}

	// 验证传输类型
	validTypes := []string{interfaces.TransportTypeSSE, interfaces.TransportTypeHTTP}
	if config.Type != "" && !p.contains(validTypes, config.Type) {
		return fmt.Errorf("unsupported transport type: %s", config.Type)
	}

	return nil
}

// validateServerConfig 验证服务器配置
func (p *Provider) validateServerConfig(name string, config interfaces.ServerConfig) error {
	if name == "" {
		return errors.New("server name is required")
	}

	// 验证传输类型
	validTypes := []string{interfaces.ClientTypeStdio, interfaces.ClientTypeSSE, interfaces.ClientTypeStreamable}
	if config.Transport != "" && !p.contains(validTypes, config.Transport) {
		return fmt.Errorf("unsupported transport type: %s", config.Transport)
	}

	// 根据传输类型验证必要字段
	switch config.Transport {
	case interfaces.ClientTypeStdio:
		if config.Command == "" {
			return errors.New("command is required for stdio transport")
		}
	case interfaces.ClientTypeSSE, interfaces.ClientTypeStreamable:
		if config.URL == "" {
			return errors.New("url is required for sse/streamable transport")
		}
	}

	// 验证工具过滤配置
	if config.Options != nil && config.Options.ToolFilter != nil {
		if err := p.validateToolFilter(config.Options.ToolFilter); err != nil {
			return fmt.Errorf("invalid tool filter: %w", err)
		}
	}

	return nil
}

// validateToolFilter 验证工具过滤配置
func (p *Provider) validateToolFilter(filter *interfaces.ToolFilterConfig) error {
	if len(filter.List) > 0 {
		mode := strings.ToLower(filter.Mode)
		if mode != interfaces.ToolFilterModeAllow && mode != interfaces.ToolFilterModeBlock {
			return fmt.Errorf("invalid filter mode: %s, must be 'allow' or 'block'", filter.Mode)
		}
	}
	return nil
}

// contains 检查切片是否包含指定元素
func (p *Provider) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// BoolPtr 返回 bool 指针的辅助函数
func BoolPtr(b bool) *bool {
	return &b
}

// GetBool 安全获取 bool 指针的值
func GetBool(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// GetDuration 解析时间字符串
func GetDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}
