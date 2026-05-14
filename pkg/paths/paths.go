package paths

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
		if v := strings.TrimSpace(os.Getenv("PROJECT_ROOT")); v != "" && isDir(v) {
			cachedRoot = v
			return
		}
		if exe, err := os.Executable(); err == nil {
			if r, ok := findRoot(filepath.Dir(exe)); ok {
				cachedRoot = r
				return
			}
		}
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

func Join(elems ...string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{root}, elems...)...), nil
}

func findRoot(start string) (string, bool) {
	dir := start
	for i := 0; i < 50; i++ {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
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

// ExecDir returns the directory of the running executable, falling back to
// the current working directory. Used by module envinit for first-run config
// release so that running the binary in any folder drops config alongside it.
func ExecDir() string {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if d := filepath.Dir(exe); d != "" {
			return d
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func CallerFileLine(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return file + ":" + strconv.Itoa(line)
	}
	return "unknown:0"
}
