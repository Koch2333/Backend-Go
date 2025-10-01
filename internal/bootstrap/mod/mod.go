package mod

import (
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type Module interface {
	// 模块唯一名（用于开关、前缀环境变量命名）
	Name() string
	// 模块默认前缀（如 /api/aicweb）
	DefaultPrefix() string
	// 默认是否启用（当没有任何开关时的兜底）
	DefaultEnabled() bool
	// 模块自行加载/生成配置（例如 envinit.Init()）
	InitEnv()
	// 实际挂载，prefix 已经计算好传入
	Mount(engine *gin.Engine, prefix string) error
}

var registry = map[string]Module{}

func Register(m Module) {
	name := strings.ToLower(m.Name())
	if _, ok := registry[name]; ok {
		log.Printf("[mod] duplicate register: %s", name)
	}
	registry[name] = m
}

// MountAll 会根据环境变量自动决定“哪些模块、什么顺序、用什么前缀”进行挂载。
// 开关与顺序：
//
//	MODULES=aicweb,redirect          # 仅挂这些，并按给定顺序
//	MODULES_DISABLE=redirect         # 全部默认启用的基础上，禁用这些
//	aicweb_ENABLED=true|false        # 单模块覆盖
//
// 前缀：
//
//	API_ROOT_PREFIX=/api/v1          # 给所有模块前缀统一加根（可选）
//	aicweb_PREFIX=/x/aicweb          # 单模块前缀覆盖（优先级更高）
//
// 兼容：若都不设置，则使用模块默认前缀与默认启用策略。
func MountAll(engine *gin.Engine) {
	if len(registry) == 0 {
		log.Printf("[mod] no modules registered")
		return
	}

	enabledList := parseList(os.Getenv("MODULES"))
	disabledSet := toSet(parseList(os.Getenv("MODULES_DISABLE")))
	root := strings.TrimSpace(os.Getenv("API_ROOT_PREFIX"))

	// 决定挂载顺序
	var order []string
	if len(enabledList) > 0 {
		for _, n := range enabledList {
			n = strings.ToLower(n)
			if _, ok := registry[n]; ok {
				order = append(order, n)
			} else {
				log.Printf("[mod] MODULES includes unknown: %s", n)
			}
		}
	} else {
		// 没有显式列表：按模块名排序
		for n := range registry {
			order = append(order, n)
		}
		sort.Strings(order)
	}

	for _, name := range order {
		m := registry[name]
		if m == nil {
			continue
		}
		en := decideEnabled(name, m.DefaultEnabled(), enabledList, disabledSet)
		if !en {
			log.Printf("[mod] skip %s (disabled)", name)
			continue
		}

		// 计算前缀：env 覆盖 > 默认
		prefix := modulePrefix(name, m.DefaultPrefix(), root)
		if prefix == "" {
			prefix = "/"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}

		// 交给模块处理自己的配置加载
		m.InitEnv()

		// 挂载
		if err := m.Mount(engine, prefix); err != nil {
			log.Printf("[mod] mount %s failed: %v", name, err)
			continue
		}
		log.Printf("[mod] mounted %s at %s", name, prefix)
	}
}

func decideEnabled(name string, def bool, explicitOrder []string, disabledSet map[string]struct{}) bool {
	// 单模块强制开关优先：<name>_ENABLED
	if v := os.Getenv(strings.ToLower(name) + "_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	// 指定了 MODULES，则只有列在里面的才算启用
	if len(explicitOrder) > 0 {
		for _, n := range explicitOrder {
			if strings.ToLower(n) == name {
				// 再看有无禁用
				_, dis := disabledSet[name]
				return !dis
			}
		}
		return false
	}
	// 未指定 MODULES：默认启用，除非在 MODULES_DISABLE 里
	if _, dis := disabledSet[name]; dis {
		return false
	}
	return def
}

func modulePrefix(name, def, root string) string {
	// <name>_PREFIX 覆盖
	if v := os.Getenv(strings.ToLower(name) + "_PREFIX"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if strings.TrimSpace(root) == "" {
		return def
	}
	// 合并 root 与 def
	join := path.Join(root, strings.TrimPrefix(def, "/"))
	if !strings.HasPrefix(join, "/") {
		join = "/" + join
	}
	return join
}

func parseList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toSet(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, v := range list {
		m[v] = struct{}{}
	}
	return m
}
