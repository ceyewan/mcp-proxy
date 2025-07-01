package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ProxyServer 代理服务器实现
type ProxyServer struct {
	name         string
	proxyConfig  *interfaces.ProxyConfig
	serverConfig interfaces.ServerConfig
	mcpServer    *server.MCPServer
	handler      http.Handler
	client       interfaces.MCPClient
}

// NewProxyServer 创建新的代理服务器
func NewProxyServer(name string, proxyConfig *interfaces.ProxyConfig, serverConfig interfaces.ServerConfig) (*ProxyServer, error) {
	// 创建 MCP 服务器选项
	serverOpts := []server.ServerOption{
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	}

	// 根据配置决定是否启用日志
	if serverConfig.Options != nil && serverConfig.Options.LogEnabled != nil && *serverConfig.Options.LogEnabled {
		serverOpts = append(serverOpts, server.WithLogging())
	}

	// 创建 MCP 服务器
	mcpServer := server.NewMCPServer(
		proxyConfig.Name,
		proxyConfig.Version,
		serverOpts...,
	)

	// 创建 HTTP 处理器
	var handler http.Handler
	switch proxyConfig.Type {
	case interfaces.TransportTypeSSE:
		handler = server.NewSSEServer(
			mcpServer,
			server.WithStaticBasePath(name),
			server.WithBaseURL(proxyConfig.BaseURL),
		)
	case interfaces.TransportTypeHTTP:
		handler = server.NewStreamableHTTPServer(
			mcpServer,
			server.WithStateLess(true),
		)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", proxyConfig.Type)
	}

	return &ProxyServer{
		name:         name,
		proxyConfig:  proxyConfig,
		serverConfig: serverConfig,
		mcpServer:    mcpServer,
		handler:      handler,
	}, nil
}

// Start 启动代理服务器
func (ps *ProxyServer) Start(ctx context.Context) error {
	log.Printf("<%s> Proxy server started", ps.name)
	return nil
}

// Stop 停止代理服务器
func (ps *ProxyServer) Stop(ctx context.Context) error {
	log.Printf("<%s> Proxy server stopped", ps.name)
	return nil
}

// RegisterClient 注册客户端到代理服务器
func (ps *ProxyServer) RegisterClient(client interfaces.MCPClient) error {
	if ps.client != nil {
		return fmt.Errorf("client already registered for server %s", ps.name)
	}

	ps.client = client

	// 添加客户端的工具、资源等到代理服务器
	if err := ps.addClientResources(client); err != nil {
		return fmt.Errorf("failed to add client resources: %w", err)
	}

	log.Printf("<%s> Client registered successfully", ps.name)
	return nil
}

// UnregisterClient 注销客户端
func (ps *ProxyServer) UnregisterClient() error {
	if ps.client == nil {
		return fmt.Errorf("no client registered for server %s", ps.name)
	}

	ps.client = nil
	log.Printf("<%s> Client unregistered", ps.name)
	return nil
}

// GetClient 获取注册的客户端
func (ps *ProxyServer) GetClient() interfaces.MCPClient {
	return ps.client
}

// GetHandler 获取 HTTP 处理器
func (ps *ProxyServer) GetHandler() http.Handler {
	return ps.handler
}

// addClientResources 添加客户端资源到代理服务器
func (ps *ProxyServer) addClientResources(client interfaces.MCPClient) error {
	ctx := context.Background()

	// 添加工具
	if err := ps.addTools(ctx, client); err != nil {
		return fmt.Errorf("failed to add tools: %w", err)
	}

	// 添加提示词
	if err := ps.addPrompts(ctx, client); err != nil {
		log.Printf("<%s> Failed to add prompts: %v", ps.name, err)
	}

	// 添加资源
	if err := ps.addResources(ctx, client); err != nil {
		log.Printf("<%s> Failed to add resources: %v", ps.name, err)
	}

	// 添加资源模板
	if err := ps.addResourceTemplates(ctx, client); err != nil {
		log.Printf("<%s> Failed to add resource templates: %v", ps.name, err)
	}

	return nil
}

// addTools 添加工具
func (ps *ProxyServer) addTools(ctx context.Context, client interfaces.MCPClient) error {
	toolsRequest := mcp.ListToolsRequest{}

	// 工具过滤函数
	filterFunc := ps.createToolFilter()

	for {
		tools, err := client.ListTools(ctx, toolsRequest)
		if err != nil {
			return err
		}

		if len(tools.Tools) == 0 {
			break
		}

		log.Printf("<%s> Successfully listed %d tools", ps.name, len(tools.Tools))
		for _, tool := range tools.Tools {
			if filterFunc(tool.Name) {
				log.Printf("<%s> Adding tool %s", ps.name, tool.Name)
				ps.mcpServer.AddTool(tool, client.CallTool)
			}
		}

		if tools.NextCursor == "" {
			break
		}
		toolsRequest.Params.Cursor = tools.NextCursor
	}

	return nil
}

// createToolFilter 创建工具过滤函数
func (ps *ProxyServer) createToolFilter() func(string) bool {
	// 默认全部通过
	filterFunc := func(toolName string) bool {
		return true
	}

	// 根据配置设置过滤逻辑
	if ps.serverConfig.Options != nil && ps.serverConfig.Options.ToolFilter != nil && len(ps.serverConfig.Options.ToolFilter.List) > 0 {
		filterSet := make(map[string]struct{})
		mode := strings.ToLower(ps.serverConfig.Options.ToolFilter.Mode)
		for _, toolName := range ps.serverConfig.Options.ToolFilter.List {
			filterSet[toolName] = struct{}{}
		}

		switch mode {
		case interfaces.ToolFilterModeAllow:
			filterFunc = func(toolName string) bool {
				_, inList := filterSet[toolName]
				if !inList {
					log.Printf("<%s> Ignoring tool %s as it is not in allow list", ps.name, toolName)
				}
				return inList
			}
		case interfaces.ToolFilterModeBlock:
			filterFunc = func(toolName string) bool {
				_, inList := filterSet[toolName]
				if inList {
					log.Printf("<%s> Ignoring tool %s as it is in block list", ps.name, toolName)
				}
				return !inList
			}
		default:
			log.Printf("<%s> Unknown tool filter mode: %s, skipping tool filter", ps.name, mode)
		}
	}

	return filterFunc
}

// addPrompts 添加提示词
func (ps *ProxyServer) addPrompts(ctx context.Context, client interfaces.MCPClient) error {
	promptsRequest := mcp.ListPromptsRequest{}
	for {
		prompts, err := client.ListPrompts(ctx, promptsRequest)
		if err != nil {
			return err
		}

		if len(prompts.Prompts) == 0 {
			break
		}

		log.Printf("<%s> Successfully listed %d prompts", ps.name, len(prompts.Prompts))
		for _, prompt := range prompts.Prompts {
			log.Printf("<%s> Adding prompt %s", ps.name, prompt.Name)
			ps.mcpServer.AddPrompt(prompt, client.GetPrompt)
		}

		if prompts.NextCursor == "" {
			break
		}
		promptsRequest.Params.Cursor = prompts.NextCursor
	}
	return nil
}

// addResources 添加资源
func (ps *ProxyServer) addResources(ctx context.Context, client interfaces.MCPClient) error {
	resourcesRequest := mcp.ListResourcesRequest{}
	for {
		resources, err := client.ListResources(ctx, resourcesRequest)
		if err != nil {
			return err
		}

		if len(resources.Resources) == 0 {
			break
		}

		log.Printf("<%s> Successfully listed %d resources", ps.name, len(resources.Resources))
		for _, resource := range resources.Resources {
			log.Printf("<%s> Adding resource %s", ps.name, resource.Name)
			ps.mcpServer.AddResource(resource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				readResource, e := client.ReadResource(ctx, request)
				if e != nil {
					return nil, e
				}
				return readResource.Contents, nil
			})
		}

		if resources.NextCursor == "" {
			break
		}
		resourcesRequest.Params.Cursor = resources.NextCursor
	}
	return nil
}

// addResourceTemplates 添加资源模板
func (ps *ProxyServer) addResourceTemplates(ctx context.Context, client interfaces.MCPClient) error {
	resourceTemplatesRequest := mcp.ListResourceTemplatesRequest{}
	for {
		resourceTemplates, err := client.ListResourceTemplates(ctx, resourceTemplatesRequest)
		if err != nil {
			return err
		}

		if len(resourceTemplates.ResourceTemplates) == 0 {
			break
		}

		log.Printf("<%s> Successfully listed %d resource templates", ps.name, len(resourceTemplates.ResourceTemplates))
		for _, resourceTemplate := range resourceTemplates.ResourceTemplates {
			log.Printf("<%s> Adding resource template %s", ps.name, resourceTemplate.Name)
			ps.mcpServer.AddResourceTemplate(resourceTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				readResource, e := client.ReadResource(ctx, request)
				if e != nil {
					return nil, e
				}
				return readResource.Contents, nil
			})
		}

		if resourceTemplates.NextCursor == "" {
			break
		}
		resourceTemplatesRequest.Params.Cursor = resourceTemplates.NextCursor
	}
	return nil
}
