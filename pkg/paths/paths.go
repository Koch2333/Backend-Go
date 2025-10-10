package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	once       sync.Once
	cachedRoot string
	cachedErr  error
)

func Root() (string, error) {
	once.Do(func() {
		if v := strings.TrimSpace(os.Getenv("PROJECT_ROOT")); v != "" {
			if isDir(v) {
				cachedRoot = v
				return
			}
		}

		if exe, err := os.Executable(); err == nil {
			if r, ok := findRoot(filepath.Dir(exe)); ok {
				cachedRoot = r
				return
			}
		}

		// 3) 从当前工作目录向上找 go.mod（方便本地运行 `go run` / IDE）
		if wd, err := os.Getwd(); err == nil {
			if r, ok := findRoot(wd); ok {
				cachedRoot = r
				return
			}
		}

		cachedErr = errors.New("paths: project root not found (no go.mod upward)")
	})
	if cachedErr != nil {
		return "", cachedErr
	}
	return cachedRoot, nil
}

// Join = filepath.Join(Root(), elems...)
func Join(elems ...string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	all := append([]string{root}, elems...)
	return filepath.Join(all...), nil
}

func findRoot(start string) (string, bool) {
	dir := start
	for i := 0; i < 50; i++ {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func CallerFileLine(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return file + ":" + strconvItoa(line)
	}
	return "unknown:0"
}

func strconvItoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	sign := ""
	if i < 0 {
		sign = "-"
		i = -i
	}
	var b [20]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = digits[i%10]
		i /= 10
	}
	if sign != "" {
		n--
		b[n] = '-'
	}
	return string(b[n:])
}
