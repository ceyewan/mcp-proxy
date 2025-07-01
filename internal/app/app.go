package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/ceyewan/mcp-proxy/internal/client"
	"github.com/ceyewan/mcp-proxy/internal/config"
	"github.com/ceyewan/mcp-proxy/internal/interfaces"
	"github.com/ceyewan/mcp-proxy/internal/middleware/auth"
	"github.com/ceyewan/mcp-proxy/internal/middleware/logger"
	"github.com/ceyewan/mcp-proxy/internal/middleware/recovery"
	"github.com/ceyewan/mcp-proxy/internal/server"
	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/sync/errgroup"
)

// Application 应用程序主体
type Application struct {
	configProvider interfaces.ConfigProvider
	clientFactory  interfaces.ClientFactory
	clientManager  interfaces.ClientManager
	serverManager  interfaces.ServerManager
}

// New 创建新的应用实例
func New() (*Application, error) {
	// 创建配置提供者
	configProvider := config.NewProvider()

	// 创建客户端工厂
	clientFactory := client.NewFactory()

	// 创建客户端管理器
	clientManager := client.NewManager(clientFactory)

	// 创建服务器管理器
	serverManager := server.NewManager()

	return &Application{
		configProvider: configProvider,
		clientFactory:  clientFactory,
		clientManager:  clientManager,
		serverManager:  serverManager,
	}, nil
}

// Run 运行应用程序
func (app *Application) Run(configPath string) error {
	// 加载配置
	config, err := app.configProvider.Load(configPath)
	if err != nil {
		return err
	}

	// 验证配置
	if err := app.configProvider.Validate(config); err != nil {
		return err
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建所有客户端
	for name, serverConfig := range config.Servers {
		client, err := app.clientFactory.CreateClient(name, serverConfig)
		if err != nil {
			return fmt.Errorf("failed to create client %s: %w", name, err)
		}
		if err := app.clientManager.AddClient(client); err != nil {
			return fmt.Errorf("failed to add client %s: %w", name, err)
		}
	}

	// 启动所有客户端
	clientInfo := mcp.Implementation{
		Name: config.Proxy.Name,
	}
	if err := app.clientManager.StartAll(ctx, clientInfo); err != nil {
		return err
	}

	// 创建并启动 HTTP 服务器
	httpServer, err := app.createHTTPServer(config)
	if err != nil {
		return err
	}

	// 启动 HTTP 服务
	go func() {
		log.Printf("Starting HTTP server on %s", config.Proxy.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second)
	defer shutdownCancel()

	// 关闭 HTTP 服务器
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// 停止所有客户端
	if err := app.clientManager.StopAll(); err != nil {
		log.Printf("Error stopping clients: %v", err)
	}

	log.Println("Application shutdown complete")
	return nil
}

// createHTTPServer 创建 HTTP 服务器
func (app *Application) createHTTPServer(config *interfaces.Config) (*http.Server, error) {
	// 解析基础 URL
	baseURL, err := url.Parse(config.Proxy.BaseURL)
	if err != nil {
		return nil, err
	}

	// 创建 HTTP 多路复用器
	mux := http.NewServeMux()

	// 创建错误组用于并发初始化
	var errorGroup errgroup.Group

	// 为每个客户端创建代理服务器和路由
	clients := app.clientManager.GetClients()
	for name, mcpClient := range clients {
		serverConfig := config.Servers[name]

		errorGroup.Go(func() error {
			// 创建代理服务器
			proxyServer, err := server.NewProxyServer(name, &config.Proxy, serverConfig)
			if err != nil {
				return err
			}

			// 注册客户端到代理服务器
			if err := proxyServer.RegisterClient(mcpClient); err != nil {
				return err
			}

			// 创建中间件链
			middlewares := app.createMiddlewares(name, &serverConfig)

			// 构造路由前缀
			mcpRoute := path.Join(baseURL.Path, name)
			if !strings.HasPrefix(mcpRoute, "/") {
				mcpRoute = "/" + mcpRoute
			}
			if !strings.HasSuffix(mcpRoute, "/") {
				mcpRoute += "/"
			}

			// 注册路由
			handler := app.chainMiddleware(proxyServer.GetHandler(), middlewares...)
			mux.Handle(mcpRoute, handler)

			log.Printf("<%s> Registered route: %s", name, mcpRoute)
			return nil
		})
	}

	// 等待所有代理服务器初始化完成
	if err := errorGroup.Wait(); err != nil {
		return nil, err
	}

	// 创建 HTTP 服务器
	httpServer := &http.Server{
		Addr:    config.Proxy.Addr,
		Handler: mux,
	}

	return httpServer, nil
}

// createMiddlewares 创建中间件链
func (app *Application) createMiddlewares(clientName string, config *interfaces.ServerConfig) []interfaces.Middleware {
	var middlewares []interfaces.Middleware

	// 恢复中间件（最外层）
	middlewares = append(middlewares, recovery.New(clientName))

	// 日志中间件
	if config.Options != nil && config.Options.LogEnabled != nil && *config.Options.LogEnabled {
		middlewares = append(middlewares, logger.New(clientName))
	}

	// 认证中间件
	if config.Options != nil && len(config.Options.AuthTokens) > 0 {
		middlewares = append(middlewares, auth.New(config.Options.AuthTokens))
	}

	return middlewares
}

// chainMiddleware 链式组合多个中间件
func (app *Application) chainMiddleware(handler http.Handler, middlewares ...interfaces.Middleware) http.Handler {
	// 从后往前包裹中间件
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i].Handle(handler)
	}
	return handler
}
