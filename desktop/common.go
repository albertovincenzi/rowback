// Command rowback-desktop wraps the Rowback web UI in a native
// macOS experience. It launches the `rowback` CLI in -serve mode on a
// private localhost port so the desktop app and the web UI share one codebase.
//
// Two build modes share these helpers:
//
//   - default (no tags): a pure-Go "browser launcher" with zero C/C++ deps. It
//     starts the server and opens the URL in the default browser, then blocks so
//     quitting the app stops the server. Build with `go build .`.
//   - webview (`-tags webview`): a true native window via WKWebView. Requires a
//     working C++ toolchain. Build with `go build -tags webview .`.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// startServer launches the rowback CLI in -serve mode on a free
// loopback port and waits for it to answer. It returns the base URL and the
// running command (whose process the caller is responsible for killing).
func startServer() (string, *exec.Cmd, error) {
	bin, err := resolveBinary()
	if err != nil {
		return "", nil, err
	}

	port, err := freePort()
	if err != nil {
		return "", nil, fmt.Errorf("find free port: %w", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	url := "http://" + addr

	cmd := exec.Command(bin, "-serve", "-addr", addr, "-dump", defaultDump())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return "", nil, fmt.Errorf("start server (%s): %w", bin, err)
	}

	if err := waitReady(url, 8*time.Second); err != nil {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", nil, fmt.Errorf("server did not become ready: %w", err)
	}

	return url, cmd, nil
}

// resolveBinary locates the sibling rowback CLI binary.
func resolveBinary() (string, error) {
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "rowback"), // inside the .app bundle (MacOS/)
			filepath.Join(dir, "..", "rowback"),
			filepath.Join(dir, "..", "..", "rowback"),
		)
	}
	if p, err := exec.LookPath("rowback"); err == nil {
		candidates = append(candidates, p)
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			if abs, aerr := filepath.Abs(c); aerr == nil {
				return abs, nil
			}
			return c, nil
		}
	}
	return "", fmt.Errorf("could not locate the 'rowback' CLI binary; build it next to this app or put it on PATH")
}

// defaultDump returns a prefilled dump path if a common dump file exists in the
// working directory, else "" (the user fills it in the UI).
func defaultDump() string {
	for _, p := range []string{"dump.sql", "backup.sql"} {
		if _, err := os.Stat(p); err == nil {
			if abs, aerr := filepath.Abs(p); aerr == nil {
				return abs
			}
			return p
		}
	}
	return ""
}

// freePort asks the OS for an unused TCP port on the loopback interface.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitReady polls the URL until it answers or the deadline passes.
func waitReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(80 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %s", timeout)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "rowback-desktop:", err)
	os.Exit(1)
}
