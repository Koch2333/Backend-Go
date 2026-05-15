package paths

import (
	"errors"
	"log"
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

	execDirOnce sync.Once
	execDirVal  string
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

// ExecDir returns the directory used to release per-module config files.
// Resolution order:
//  1. CONFIG_DIR env var (absolute or relative)
//  2. Directory of the current executable — Windows: C:\path\to\server.exe
//     → C:\path\to ; POSIX: /opt/roast/server → /opt/roast
//  3. Cwd (used when the exe lives under a Go toolchain build cache, i.e.
//     `go run` / `go test` produced it)
//
// Result is cached and logged the first time it's resolved so first-run
// behaviour is visible in startup logs.
func ExecDir() string {
	execDirOnce.Do(func() {
		execDirVal, _ = resolveExecDir()
		log.Printf("[paths] config base = %s", execDirVal)
	})
	return execDirVal
}

func resolveExecDir() (dir, source string) {
	if v := strings.TrimSpace(os.Getenv("CONFIG_DIR")); v != "" {
		if abs, err := filepath.Abs(v); err == nil {
			v = abs
		}
		return v, "CONFIG_DIR"
	}
	if exe, err := os.Executable(); err == nil && exe != "" {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
			exe = resolved
		}
		d := filepath.Dir(exe)
		if d != "" && !isGoBuildTempDir(d) {
			return d, "exe"
		}
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return wd, "cwd"
	}
	return ".", "fallback"
}

// isGoBuildTempDir reports whether the path looks like a Go toolchain build
// cache used by `go run`/`go test` (e.g. /tmp/go-build3849…/b001/exe on Linux,
// C:\Users\xxx\AppData\Local\Temp\go-build…\exe on Windows). On Windows the
// path separator check covers both "\\go-build" and "/go-build" since Go file
// APIs sometimes return mixed slashes.
func isGoBuildTempDir(dir string) bool {
	tmp := os.TempDir()
	if tmp != "" {
		if rel, err := filepath.Rel(tmp, dir); err == nil && !strings.HasPrefix(rel, "..") && strings.Contains(rel, "go-build") {
			return true
		}
	}
	return strings.Contains(dir, `\go-build`) || strings.Contains(dir, "/go-build")
}

func CallerFileLine(skip int) string {
	if _, file, line, ok := runtime.Caller(skip + 1); ok {
		return file + ":" + strconv.Itoa(line)
	}
	return "unknown:0"
}
