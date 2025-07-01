package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Manager 服务器管理器实现
type Manager struct {
	servers map[string]*ProxyServer
	mutex   sync.RWMutex
}

// NewManager 创建新的服务器管理器
func NewManager() interfaces.ServerManager {
	return &Manager{
		servers: make(map[string]*ProxyServer),
	}
}

// Start 启动服务器
func (m *Manager) Start(ctx context.Context) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for name, server := range m.servers {
		log.Printf("Starting server: %s", name)
		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("failed to start server %s: %w", name, err)
		}
	}

	log.Printf("All servers started successfully")
	return nil
}

// Stop 停止服务器
func (m *Manager) Stop(ctx context.Context) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var errors []error
	for name, server := range m.servers {
		log.Printf("Stopping server: %s", name)
		if err := server.Stop(ctx); err != nil {
			log.Printf("Error stopping server %s: %v", name, err)
			errors = append(errors, fmt.Errorf("failed to stop server %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return errors[0] // 返回第一个错误
	}

	log.Printf("All servers stopped")
	return nil
}

// AddClient 添加客户端
func (m *Manager) AddClient(client interfaces.MCPClient) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	name := client.GetName()
	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server for client %s not found", name)
	}

	return server.RegisterClient(client)
}

// RemoveClient 移除客户端
func (m *Manager) RemoveClient(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server for client %s not found", name)
	}

	return server.UnregisterClient()
}

// GetClients 获取所有客户端
func (m *Manager) GetClients() map[string]interfaces.MCPClient {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]interfaces.MCPClient)
	for name, server := range m.servers {
		if client := server.GetClient(); client != nil {
			result[name] = client
		}
	}
	return result
}

// AddServer 添加服务器
func (m *Manager) AddServer(name string, server *ProxyServer) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server %s already exists", name)
	}

	m.servers[name] = server
	log.Printf("Added server: %s", name)
	return nil
}

// RemoveServer 移除服务器
func (m *Manager) RemoveServer(name string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	server, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	// 停止服务器
	ctx := context.Background()
	if err := server.Stop(ctx); err != nil {
		log.Printf("Error stopping server %s: %v", name, err)
	}

	delete(m.servers, name)
	log.Printf("Removed server: %s", name)
	return nil
}

// GetServer 获取服务器
func (m *Manager) GetServer(name string) *ProxyServer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.servers[name]
}

// GetServers 获取所有服务器
func (m *Manager) GetServers() map[string]*ProxyServer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*ProxyServer)
	for name, server := range m.servers {
		result[name] = server
	}
	return result
}
