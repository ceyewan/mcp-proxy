package recovery

import (
	"log"
	"net/http"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Middleware 恢复中间件实现
type Middleware struct {
	name string
}

// New 创建新的恢复中间件
func New(name string) interfaces.Middleware {
	return &Middleware{
		name: name,
	}
}

// Handle 处理 HTTP 请求
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("<%s> Recovered from panic: %v", m.name, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// GetName 获取中间件名称
func (m *Middleware) GetName() string {
	return "recovery"
}
