package client

import (
	"context"
	"fmt"
	"log"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// StdioClient stdio 客户端实现
type StdioClient struct {
	name      string
	config    interfaces.ServerConfig
	client    *client.Client
	connected bool
}

// NewStdioClient 创建新的 stdio 客户端
func NewStdioClient(name string, config interfaces.ServerConfig) (interfaces.MCPClient, error) {
	if config.Command == "" {
		return nil, fmt.Errorf("command is required for stdio client")
	}

	return &StdioClient{
		name:   name,
		config: config,
	}, nil
}

// Connect 连接到 MCP 服务器
func (c *StdioClient) Connect(ctx context.Context, clientInfo mcp.Implementation) error {
	if c.connected {
		return nil
	}

	// 构造环境变量
	envs := make([]string, 0, len(c.config.Env))
	for key, value := range c.config.Env {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}

	// 创建 stdio 客户端
	mcpClient, err := client.NewStdioMCPClient(c.config.Command, envs, c.config.Args...)
	if err != nil {
		return fmt.Errorf("failed to create stdio client: %w", err)
	}

	c.client = mcpClient
	c.connected = true

	// 初始化请求
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = clientInfo
	initRequest.Params.Capabilities = mcp.ClientCapabilities{
		Experimental: make(map[string]interface{}),
		Roots:        nil,
		Sampling:     nil,
	}

	_, err = c.client.Initialize(ctx, initRequest)
	if err != nil {
		c.connected = false
		return fmt.Errorf("failed to initialize client: %w", err)
	}

	log.Printf("<%s> Successfully initialized stdio MCP client", c.name)
	return nil
}

// Disconnect 断开连接
func (c *StdioClient) Disconnect() error {
	if !c.connected || c.client == nil {
		return nil
	}

	err := c.client.Close()
	c.connected = false
	c.client = nil
	return err
}

// GetName 获取客户端名称
func (c *StdioClient) GetName() string {
	return c.name
}

// GetType 获取客户端类型
func (c *StdioClient) GetType() string {
	return interfaces.ClientTypeStdio
}

// IsConnected 检查连接状态
func (c *StdioClient) IsConnected() bool {
	return c.connected
}

// NeedsPing 是否需要定期 ping
func (c *StdioClient) NeedsPing() bool {
	return false // stdio 客户端不需要 ping
}

// Ping 发送 ping 消息
func (c *StdioClient) Ping(ctx context.Context) error {
	if !c.connected || c.client == nil {
		return fmt.Errorf("client not connected")
	}
	return c.client.Ping(ctx)
}

// MCP 协议方法实现

func (c *StdioClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.Initialize(ctx, request)
}

func (c *StdioClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListTools(ctx, request)
}

func (c *StdioClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.CallTool(ctx, request)
}

func (c *StdioClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListPrompts(ctx, request)
}

func (c *StdioClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.GetPrompt(ctx, request)
}

func (c *StdioClient) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListResources(ctx, request)
}

func (c *StdioClient) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ReadResource(ctx, request)
}

func (c *StdioClient) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListResourceTemplates(ctx, request)
}
