package envinit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirEmail      = "config/email"
	baseEmailEnv  = "config/email/.env"
	localEmailEnv = "config/email/local.env"
	exampleBase   = "config/email/.env.example"
	exampleLocal  = "config/email/local.env.example"
)

// Init：按约定生成示例、加载 env、规范化并输出安全日志。
func Init() {
	// 1) 目录与示例文件
	_ = os.MkdirAll(dirEmail, 0o755)
	_ = writeIfNotExists(exampleBase, exampleBaseContent())
	_ = writeIfNotExists(exampleLocal, exampleLocalContent())

	// 2) 加载 .env（仅在未设置时生效）
	_ = loadDotenv(baseEmailEnv, false)

	// 3) 加载 local.env（强覆盖）
	_ = loadDotenv(localEmailEnv, true)

	// 4) 规范化与默认值
	normalizeLower("EMAIL_STRATEGY", "none") // graph | smtp | log | none
	normalizeLower("GRAPH_CLOUD", "global")  // global | cn

	// 5) 安全日志
	logSummary()
}

// 仅当文件不存在时写入内容（不覆盖用户已存在的修改）
func writeIfNotExists(path, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // 已存在就不动
	} else if !os.IsNotExist(err) {
		return err
	}
	// 确保父目录存在
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return os.WriteFile(path, []byte(strings.TrimRight(content, "\n")+"\n"), 0o644)
}

// 读取 KEY=VALUE 文件；override=false：仅当环境变量未设置时写入；true：强覆盖
func loadDotenv(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(raw, ";") {
			continue
		}
		k, v, ok := cutKV(raw)
		if !ok || k == "" {
			continue
		}
		if !override {
			if _, exists := os.LookupEnv(k); exists {
				continue
			}
		}
		_ = os.Setenv(k, v)
	}
	return sc.Err()
}

func cutKV(s string) (k, v string, ok bool) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	k = strings.TrimSpace(parts[0])
	v = strings.TrimSpace(parts[1])

	// 去掉包裹引号
	if len(v) >= 2 {
		if (strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`)) ||
			(strings.HasPrefix(v, `'`) && strings.HasSuffix(v, `'`)) {
			v = v[1 : len(v)-1]
		}
	}
	return k, v, true
}

func normalizeLower(key, def string) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		val = def
	}
	_ = os.Setenv(key, strings.ToLower(val))
}

func logSummary() {
	mask := func(s string) string {
		if s == "" {
			return "<empty>"
		}
		if len(s) <= 6 {
			return "***"
		}
		return s[:3] + "..." + s[len(s)-3:]
	}

	fmt.Printf("[env/email] EMAIL_STRATEGY=%q GRAPH_CLOUD=%q\n",
		os.Getenv("EMAIL_STRATEGY"), os.Getenv("GRAPH_CLOUD"))

	fmt.Printf("[env/email] GRAPH_TENANT_ID=%s GRAPH_CLIENT_ID=%s GRAPH_CLIENT_SECRET=%s\n",
		mask(os.Getenv("GRAPH_TENANT_ID")),
		mask(os.Getenv("GRAPH_CLIENT_ID")),
		mask(os.Getenv("GRAPH_CLIENT_SECRET")),
	)

	if id := os.Getenv("GRAPH_FROM_ID"); id != "" {
		fmt.Printf("[env/email] FROM: id=%s (优先对象ID)\n", mask(id))
	} else {
		fmt.Printf("[env/email] FROM: upn=%q\n", os.Getenv("GRAPH_FROM_UPN"))
	}
}

// -------------------- example contents --------------------

func exampleBaseContent() string {
	return `# ===========================
# Email configuration defaults (.env)
# This file provides defaults and is only applied when a key is NOT set elsewhere.
# Copy/override with config/email/local.env for local development.
# ===========================

# Choose email strategy: graph | smtp | log | none
EMAIL_STRATEGY=graph

# Cloud: global (default) or cn (21Vianet)
GRAPH_CLOUD=global

# ---- Microsoft Graph (Application) ----
# Values from Microsoft Entra ID (portal.azure.com for global / portal.azure.cn for cn)
GRAPH_TENANT_ID=
GRAPH_CLIENT_ID=
GRAPH_CLIENT_SECRET=

# Prefer object ID (ExternalDirectoryObjectId). Fallback to UPN if empty.
GRAPH_FROM_ID=
# GRAPH_FROM_UPN=noreply@your-domain.com

# ---- SMTP fallback (optional) ----
# Global: smtp.office365.com:587 / China: smtp.partner.outlook.cn:587
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=No Reply <noreply@your-domain.com>`
}

func exampleLocalContent() string {
	return `# ===========================
# Email configuration local overrides (local.env)
# Rename this file to local.env and fill real values.
# ===========================

EMAIL_STRATEGY=graph
GRAPH_CLOUD=global

GRAPH_TENANT_ID=00000000-0000-0000-0000-000000000000
GRAPH_CLIENT_ID=00000000-0000-0000-0000-000000000000
GRAPH_CLIENT_SECRET=change_me_secret

# Prefer object id
GRAPH_FROM_ID=00000000-0000-0000-0000-000000000000
# GRAPH_FROM_UPN=noreply@example.com

# SMTP fallback (optional)
SMTP_HOST=smtp.office365.com
SMTP_PORT=587
SMTP_USERNAME=svc-sender@example.com
SMTP_PASSWORD=change_me_password
SMTP_FROM=No Reply <noreply@example.com>`
}
