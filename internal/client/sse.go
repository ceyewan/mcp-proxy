package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// SSEClient SSE 客户端实现
type SSEClient struct {
	name      string
	config    interfaces.ServerConfig
	client    *client.Client
	connected bool
}

// NewSSEClient 创建新的 SSE 客户端
func NewSSEClient(name string, config interfaces.ServerConfig) (interfaces.MCPClient, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("url is required for SSE client")
	}

	return &SSEClient{
		name:   name,
		config: config,
	}, nil
}

// Connect 连接到 MCP 服务器
func (c *SSEClient) Connect(ctx context.Context, clientInfo mcp.Implementation) error {
	if c.connected {
		return nil
	}

	// 创建 SSE 客户端选项
	var options []transport.ClientOption
	if len(c.config.Headers) > 0 {
		options = append(options, client.WithHeaders(c.config.Headers))
	}

	// 创建 SSE 客户端
	mcpClient, err := client.NewSSEMCPClient(c.config.URL, options...)
	if err != nil {
		return fmt.Errorf("failed to create SSE client: %w", err)
	}

	c.client = mcpClient

	// 启动客户端
	err = c.client.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start SSE client: %w", err)
	}

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

	log.Printf("<%s> Successfully initialized SSE MCP client", c.name)

	// 启动定期 ping
	go c.startPingTask(ctx)

	return nil
}

// startPingTask 启动定时 ping 任务，保持连接活跃
func (c *SSEClient) startPingTask(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("<%s> Context done, stopping ping", c.name)
			return
		case <-ticker.C:
			if c.connected && c.client != nil {
				_ = c.client.Ping(ctx)
			}
		}
	}
}

// Disconnect 断开连接
func (c *SSEClient) Disconnect() error {
	if !c.connected || c.client == nil {
		return nil
	}

	err := c.client.Close()
	c.connected = false
	c.client = nil
	return err
}

// GetName 获取客户端名称
func (c *SSEClient) GetName() string {
	return c.name
}

// GetType 获取客户端类型
func (c *SSEClient) GetType() string {
	return interfaces.ClientTypeSSE
}

// IsConnected 检查连接状态
func (c *SSEClient) IsConnected() bool {
	return c.connected
}

// NeedsPing 是否需要定期 ping
func (c *SSEClient) NeedsPing() bool {
	return true // SSE 客户端需要 ping
}

// Ping 发送 ping 消息
func (c *SSEClient) Ping(ctx context.Context) error {
	if !c.connected || c.client == nil {
		return fmt.Errorf("client not connected")
	}
	return c.client.Ping(ctx)
}

// MCP 协议方法实现

func (c *SSEClient) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.Initialize(ctx, request)
}

func (c *SSEClient) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListTools(ctx, request)
}

func (c *SSEClient) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.CallTool(ctx, request)
}

func (c *SSEClient) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListPrompts(ctx, request)
}

func (c *SSEClient) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.GetPrompt(ctx, request)
}

func (c *SSEClient) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListResources(ctx, request)
}

func (c *SSEClient) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ReadResource(ctx, request)
}

func (c *SSEClient) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("client not connected")
	}
	return c.client.ListResourceTemplates(ctx, request)
}
