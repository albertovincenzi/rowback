//go:build webview

// This is the primary desktop binary: a true native window backed by WKWebView.
// It requires a working C++ toolchain (CGO). Build with `go build -tags webview .`.
package main

import (
	webview "github.com/webview/webview_go"
)

func main() {
	url, cmd, err := startServer()
	if err != nil {
		fail(err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Rowback")
	w.SetSize(860, 960, webview.HintNone)
	w.SetSize(560, 640, webview.HintMin)
	w.Navigate(url)
	w.Run()
}
