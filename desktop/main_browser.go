//go:build !webview

// Pure-Go fallback desktop binary with zero C/C++ dependencies, so it builds
// anywhere Go does. Instead of an embedded WKWebView it starts the server
// DETACHED and opens the URL in the default browser, then exits immediately.
//
// Exiting immediately matters: an .app whose executable blocks (without running
// a Cocoa event loop) is flagged by macOS as "not responding". By detaching the
// server and returning, the launcher process finishes cleanly while the server
// keeps running; the browser tab is the UI.
//
// To get a true native window instead, build with `-tags webview`.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// fixedAddr is a stable loopback address so relaunching reuses a running server
// (and opens a fresh browser tab) instead of spawning duplicates.
const fixedAddr = "127.0.0.1:8765"

func main() {
	url := "http://" + fixedAddr

	// Reuse an already-running server if present; otherwise start one detached.
	if waitReady(url, 400*time.Millisecond) != nil {
		if err := startDetached(fixedAddr); err != nil {
			fail(err)
		}
		if err := waitReady(url, 12*time.Second); err != nil {
			fail(fmt.Errorf("server did not become ready: %w", err))
		}
	}

	_ = exec.Command("open", url).Run()
	// Return immediately — the detached server keeps serving.
}

// startDetached launches the CLI -serve in its own session so it survives this
// launcher exiting, with output sent to a log file (no inherited console).
func startDetached(addr string) error {
	bin, err := resolveBinary()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "-serve", "-addr", addr, "-dump", defaultDump())
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if logf, err := os.OpenFile(filepath.Join(os.TempDir(), "rowback-server.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start server (%s): %w", bin, err)
	}
	return cmd.Process.Release()
}
