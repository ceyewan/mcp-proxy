package logger

import (
	"log"
	"net/http"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Middleware 日志中间件实现
type Middleware struct {
	prefix string
}

// New 创建新的日志中间件
func New(prefix string) interfaces.Middleware {
	return &Middleware{
		prefix: prefix,
	}
}

// Handle 处理 HTTP 请求
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("<%s> Request [%s] %s", m.prefix, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// GetName 获取中间件名称
func (m *Middleware) GetName() string {
	return "logger"
}
