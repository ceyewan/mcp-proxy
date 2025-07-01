package client

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
	"github.com/mark3labs/mcp-go/mcp"
)

// Manager 客户端管理器实现
type Manager struct {
	clients map[string]interfaces.MCPClient
	mutex   sync.RWMutex
	factory interfaces.ClientFactory
}

// NewManager 创建新的客户端管理器
func NewManager(factory interfaces.ClientFactory) interfaces.ClientManager {
	return &Manager{
		clients: make(map[string]interfaces.MCPClient),
		factory: factory,
	}
}

// AddClient 添加客户端
func (m *Manager) AddClient(client interfaces.MCPClient) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := client.GetName()
	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("client %s already exists", name)
	}

	m.clients[name] = client
	log.Printf("Added client: %s (type: %s)", name, client.GetType())
	return nil
}

// RemoveClient 移除客户端
func (m *Manager) RemoveClient(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	// 断开连接
	if err := client.Disconnect(); err != nil {
		log.Printf("Error disconnecting client %s: %v", name, err)
	}

	delete(m.clients, name)
	log.Printf("Removed client: %s", name)
	return nil
}

// GetClient 获取客户端
func (m *Manager) GetClient(name string) interfaces.MCPClient {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.clients[name]
}

// GetClients 获取所有客户端
func (m *Manager) GetClients() map[string]interfaces.MCPClient {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 返回副本以避免并发问题
	result := make(map[string]interfaces.MCPClient)
	for name, client := range m.clients {
		result[name] = client
	}
	return result
}

// StartAll 启动所有客户端
func (m *Manager) StartAll(ctx context.Context, clientInfo mcp.Implementation) error {
	m.mutex.RLock()
	clients := make(map[string]interfaces.MCPClient)
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mutex.RUnlock()

	if len(clients) == 0 {
		log.Printf("No clients to start")
		return nil
	}

	// 并发启动所有客户端
	var wg sync.WaitGroup
	errChan := make(chan error, len(clients))

	for name, client := range clients {
		wg.Add(1)
		go func(name string, client interfaces.MCPClient) {
			defer wg.Done()

			log.Printf("Starting client: %s", name)
			if err := client.Connect(ctx, clientInfo); err != nil {
				log.Printf("Failed to start client %s: %v", name, err)
				select {
				case errChan <- fmt.Errorf("failed to start client %s: %w", name, err):
				default:
				}
				return
			}
			log.Printf("Successfully started client: %s", name)
		}(name, client)
	}

	// 等待所有客户端启动完成
	wg.Wait()
	close(errChan)

	// 收集所有错误
	var startErrors []error
	for err := range errChan {
		startErrors = append(startErrors, err)
	}

	if len(startErrors) > 0 {
		// 如果有错误，返回第一个错误
		return startErrors[0]
	}

	log.Printf("All clients started successfully")
	return nil
}

// StopAll 停止所有客户端
func (m *Manager) StopAll() error {
	m.mutex.RLock()
	clients := make(map[string]interfaces.MCPClient)
	for name, client := range m.clients {
		clients[name] = client
	}
	m.mutex.RUnlock()

	if len(clients) == 0 {
		log.Printf("No clients to stop")
		return nil
	}

	// 并发停止所有客户端
	var wg sync.WaitGroup
	errChan := make(chan error, len(clients))

	for name, client := range clients {
		wg.Add(1)
		go func(name string, client interfaces.MCPClient) {
			defer wg.Done()

			log.Printf("Stopping client: %s", name)
			if err := client.Disconnect(); err != nil {
				log.Printf("Error stopping client %s: %v", name, err)
				select {
				case errChan <- fmt.Errorf("failed to stop client %s: %w", name, err):
				default:
				}
				return
			}
			log.Printf("Successfully stopped client: %s", name)
		}(name, client)
	}

	// 等待所有客户端停止完成
	wg.Wait()
	close(errChan)

	// 收集错误
	var stopErrors []error
	for err := range errChan {
		stopErrors = append(stopErrors, err)
	}

	if len(stopErrors) > 0 {
		// 记录所有错误但不返回错误（确保清理工作完成）
		for _, err := range stopErrors {
			log.Printf("Stop error: %v", err)
		}
	}

	log.Printf("All clients stopped")
	return nil
}

// CreateAndAddClient 创建并添加客户端
func (m *Manager) CreateAndAddClient(name string, config interfaces.ServerConfig) error {
	client, err := m.factory.CreateClient(name, config)
	if err != nil {
		return fmt.Errorf("failed to create client %s: %w", name, err)
	}

	return m.AddClient(client)
}

// GetConnectedClients 获取已连接的客户端
func (m *Manager) GetConnectedClients() map[string]interfaces.MCPClient {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]interfaces.MCPClient)
	for name, client := range m.clients {
		if client.IsConnected() {
			result[name] = client
		}
	}
	return result
}

// GetClientStats 获取客户端统计信息
func (m *Manager) GetClientStats() map[string]map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]map[string]interface{})
	for name, client := range m.clients {
		result[name] = map[string]interface{}{
			"type":      client.GetType(),
			"connected": client.IsConnected(),
			"needsPing": client.NeedsPing(),
		}
	}
	return result
}
