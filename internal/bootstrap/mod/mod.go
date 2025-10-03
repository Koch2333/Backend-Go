package mod

import (
	"log"
	"os"
	"path"
	"strings"

	"backend-go/internal/bootstrap/plug"

	"github.com/gin-gonic/gin"
)

func MountAll(engine *gin.Engine) {
	if len(plug.All()) == 0 {
		log.Printf("[mod] no modules registered")
		return
	}

	enabledList := parseList(os.Getenv("MODULES"))
	disabledSet := toSet(parseList(os.Getenv("MODULES_DISABLE")))
	root := strings.TrimSpace(os.Getenv("API_ROOT_PREFIX"))

	// 计算挂载顺序
	var order []string
	if len(enabledList) > 0 {
		for _, n := range enabledList {
			if plug.Get(n) != nil {
				order = append(order, strings.ToLower(n))
			} else {
				log.Printf("[mod] MODULES includes unknown: %s", n)
			}
		}
	} else {
		order = plug.Names()
	}

	for _, name := range order {
		m := plug.Get(name)
		if m == nil {
			continue
		}
		en := decideEnabled(name, m.DefaultEnabled(), enabledList, disabledSet)
		if !en {
			log.Printf("[mod] skip %s (disabled)", name)
			continue
		}

		// 前缀：env 覆盖 > root+默认
		prefix := modulePrefix(name, m.DefaultPrefix(), root)
		if prefix == "" {
			prefix = "/"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}

		m.InitEnv()
		if err := m.Mount(engine, prefix); err != nil {
			log.Printf("[mod] mount %s failed: %v", name, err)
			continue
		}
		log.Printf("[mod] mounted %s at %s", name, prefix)
	}
}

func decideEnabled(name string, def bool, explicitOrder []string, disabledSet map[string]struct{}) bool {
	// <name>_ENABLED 覆盖
	if v := os.Getenv(strings.ToLower(name) + "_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	if len(explicitOrder) > 0 {
		for _, n := range explicitOrder {
			if strings.ToLower(n) == name {
				_, dis := disabledSet[name]
				return !dis
			}
		}
		return false
	}
	if _, dis := disabledSet[name]; dis {
		return false
	}
	return def
}

func modulePrefix(name, def, root string) string {
	if v := os.Getenv(strings.ToLower(name) + "_PREFIX"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	if strings.TrimSpace(root) == "" {
		return def
	}
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
