package auth

import (
	"net/http"
	"strings"

	"github.com/ceyewan/mcp-proxy/internal/interfaces"
)

// Middleware 认证中间件实现
type Middleware struct {
	tokens map[string]struct{}
}

// New 创建新的认证中间件
func New(tokens []string) interfaces.Middleware {
	tokenSet := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		tokenSet[token] = struct{}{}
	}

	return &Middleware{
		tokens: tokenSet,
	}
}

// Handle 处理 HTTP 请求
func (m *Middleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(m.tokens) == 0 {
			// 没有配置 token，直接通过
			next.ServeHTTP(w, r)
			return
		}

		// 获取 Authorization 头
		token := r.Header.Get("Authorization")
		token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 验证 token
		if _, ok := m.tokens[token]; !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetName 获取中间件名称
func (m *Middleware) GetName() string {
	return "auth"
}
