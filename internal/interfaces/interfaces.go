package interfaces

import (
	"context"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Transport 定义传输层接口，支持不同的协议传输方式
type Transport interface {
	// Start 启动传输服务
	Start(ctx context.Context) error
	// Stop 停止传输服务
	Stop(ctx context.Context) error
	// GetHandler 获取 HTTP 处理器
	GetHandler() http.Handler
	// GetType 获取传输类型
	GetType() string
}

// MCPClient 定义 MCP 客户端接口，抽象不同类型的客户端实现
type MCPClient interface {
	// Connect 连接到 MCP 服务器
	Connect(ctx context.Context, clientInfo mcp.Implementation) error
	// Disconnect 断开连接
	Disconnect() error
	// GetName 获取客户端名称
	GetName() string
	// GetType 获取客户端类型
	GetType() string
	// IsConnected 检查连接状态
	IsConnected() bool
	// NeedsPing 是否需要定期 ping
	NeedsPing() bool
	// Ping 发送 ping 消息
	Ping(ctx context.Context) error

	// MCP 协议方法
	Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error)
	GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
	ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error)
	ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
	ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error)
}

// Middleware 定义中间件接口
type Middleware interface {
	// Handle 处理 HTTP 请求
	Handle(next http.Handler) http.Handler
	// GetName 获取中间件名称
	GetName() string
}

// ConfigProvider 定义配置提供者接口
type ConfigProvider interface {
	// Load 加载配置
	Load(path string) (*Config, error)
	// Validate 验证配置
	Validate(config *Config) error
}

// TransportFactory 定义传输工厂接口
type TransportFactory interface {
	// CreateTransport 创建传输实例
	CreateTransport(config TransportConfig) (Transport, error)
	// SupportedTypes 获取支持的传输类型
	SupportedTypes() []string
}

// ClientFactory 定义客户端工厂接口
type ClientFactory interface {
	// CreateClient 创建客户端实例
	CreateClient(name string, config ServerConfig) (MCPClient, error)
	// SupportedTypes 获取支持的客户端类型
	SupportedTypes() []string
}

// MiddlewareFactory 定义中间件工厂接口
type MiddlewareFactory interface {
	// CreateMiddleware 创建中间件实例
	CreateMiddleware(config MiddlewareConfig) (Middleware, error)
	// SupportedTypes 获取支持的中间件类型
	SupportedTypes() []string
}

// ServerManager 定义服务器管理器接口
type ServerManager interface {
	// Start 启动服务器
	Start(ctx context.Context) error
	// Stop 停止服务器
	Stop(ctx context.Context) error
	// AddClient 添加客户端
	AddClient(client MCPClient) error
	// RemoveClient 移除客户端
	RemoveClient(name string) error
	// GetClients 获取所有客户端
	GetClients() map[string]MCPClient
}

// ClientManager 定义客户端管理器接口
type ClientManager interface {
	// AddClient 添加客户端
	AddClient(client MCPClient) error
	// RemoveClient 移除客户端
	RemoveClient(name string) error
	// GetClient 获取客户端
	GetClient(name string) MCPClient
	// GetClients 获取所有客户端
	GetClients() map[string]MCPClient
	// StartAll 启动所有客户端
	StartAll(ctx context.Context, clientInfo mcp.Implementation) error
	// StopAll 停止所有客户端
	StopAll() error
}

// 配置结构体定义

// Config 主配置
type Config struct {
	Proxy   ProxyConfig             `json:"proxy"`
	Servers map[string]ServerConfig `json:"servers"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	BaseURL string         `json:"baseURL"`
	Addr    string         `json:"addr"`
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Type    string         `json:"type"`
	Options *OptionsConfig `json:"options,omitempty"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   time.Duration     `json:"timeout,omitempty"`
	Options   *OptionsConfig    `json:"options,omitempty"`
}

// OptionsConfig 选项配置
type OptionsConfig struct {
	PanicIfInvalid *bool             `json:"panicIfInvalid,omitempty"`
	LogEnabled     *bool             `json:"logEnabled,omitempty"`
	AuthTokens     []string          `json:"authTokens,omitempty"`
	ToolFilter     *ToolFilterConfig `json:"toolFilter,omitempty"`
}

// ToolFilterConfig 工具过滤配置
type ToolFilterConfig struct {
	Mode string   `json:"mode,omitempty"`
	List []string `json:"list,omitempty"`
}

// TransportConfig 传输配置
type TransportConfig struct {
	Type    string                 `json:"type"`
	BaseURL string                 `json:"baseURL,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// 常量定义

// 传输类型
const (
	TransportTypeSSE   = "sse"
	TransportTypeHTTP  = "streamable-http"
	TransportTypeStdio = "stdio"
)

// 客户端类型
const (
	ClientTypeStdio      = "stdio"
	ClientTypeSSE        = "sse"
	ClientTypeStreamable = "streamable-http"
)

// 中间件类型
const (
	MiddlewareTypeAuth     = "auth"
	MiddlewareTypeLogger   = "logger"
	MiddlewareTypeRecovery = "recovery"
)

// 工具过滤模式
const (
	ToolFilterModeAllow = "allow"
	ToolFilterModeBlock = "block"
)
