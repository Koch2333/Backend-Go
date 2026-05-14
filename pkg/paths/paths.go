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
//
// During `go run` the executable lives under `/tmp/go-build*/exe/…`, which is
// useless for config persistence — when we detect that, we fall back to cwd so
// development still drops configs at the project root.
func ExecDir() string {
	if v := strings.TrimSpace(os.Getenv("CONFIG_DIR")); v != "" {
		return v
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		d := filepath.Dir(exe)
		if d != "" && !isGoBuildTempDir(d) {
			return d
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// isGoBuildTempDir reports whether the path looks like a Go toolchain build
// cache used by `go run`/`go test` (e.g. /tmp/go-build3849.../b001/exe).
func isGoBuildTempDir(dir string) bool {
	tmp := os.TempDir()
	if tmp != "" {
		rel, err := filepath.Rel(tmp, dir)
		if err == nil && !strings.HasPrefix(rel, "..") && strings.Contains(rel, "go-build") {
			return true
		}
	}
	return strings.Contains(dir, string(os.PathSeparator)+"go-build")
}

func CallerFileLine(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return file + ":" + strconv.Itoa(line)
	}
	return "unknown:0"
}
