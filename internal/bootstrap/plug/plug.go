package plug

import (
	"log"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// Module 由各业务模块实现
type Module interface {
	Name() string
	DefaultPrefix() string
	DefaultEnabled() bool
	InitEnv()
	Mount(e *gin.Engine, prefix string) error
}

var registry = map[string]Module{}

// Register 在各模块的 init() 中调用
func Register(m Module) {
	if m == nil {
		return
	}
	name := strings.ToLower(m.Name())
	if _, ok := registry[name]; ok {
		log.Printf("[plug] duplicate register: %s", name)
	}
	registry[name] = m
}

// Names 返回已注册模块名（升序）
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Get 按名取模块
func Get(name string) Module { return registry[strings.ToLower(name)] }

// All 返回 name->module 映射（只读）
func All() map[string]Module { return registry }
