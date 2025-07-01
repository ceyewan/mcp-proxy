package client

import (
	"fmt"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Factory 客户端工厂实现
type Factory struct{}

// NewFactory 创建新的客户端工厂
func NewFactory() interfaces.ClientFactory {
	return &Factory{}
}

// CreateClient 创建客户端实例
func (f *Factory) CreateClient(name string, config interfaces.ServerConfig) (interfaces.MCPClient, error) {
	switch config.Transport {
	case interfaces.ClientTypeStdio:
		return NewStdioClient(name, config)
	case interfaces.ClientTypeSSE:
		return NewSSEClient(name, config)
	case interfaces.ClientTypeStreamable:
		return NewStreamableClient(name, config)
	default:
		return nil, fmt.Errorf("unsupported client type: %s", config.Transport)
	}
}

// SupportedTypes 获取支持的客户端类型
func (f *Factory) SupportedTypes() []string {
	return []string{
		interfaces.ClientTypeStdio,
		interfaces.ClientTypeSSE,
		interfaces.ClientTypeStreamable,
	}
}
