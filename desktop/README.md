# Rowback — Desktop

A native macOS wrapper around the **Rowback** web UI. It launches the sibling
`rowback` CLI in `-serve` mode on a private `127.0.0.1` port and shows that exact
web UI — so the desktop app and the browser app are one codebase.

Module: `rowback-desktop` (its own Go module, separate from the parent CLI).

## Build

```bash
./build-app.sh        # builds the CLI + GUI and assembles Rowback.app
open "Rowback.app"
```

`build-app.sh` tries the native WebView build first and automatically falls back
to the pure‑Go browser launcher if the C++ toolchain isn't available.

## Two build modes

| File | Build tag | Role |
|------|-----------|------|
| `common.go` | _(none)_ | Shared helpers: `resolveBinary`, `freePort`, `waitReady`, `defaultDump`. |
| `main_webview.go` | `//go:build webview` | **Primary.** True native window via `WKWebView` (`webview_go`, CGO/C++). |
| `main_browser.go` | `//go:build !webview` | **Fallback.** Pure‑Go: starts the server detached and opens the URL in the default browser. Zero C/C++ deps. |

```bash
go build -o rowback-desktop .               # browser launcher (works everywhere)
go build -tags webview -o rowback-desktop . # native WKWebView window
```

## The native window needs a working C++ toolchain

`webview_go` compiles C++. If your macOS Command Line Tools are missing the C++
standard library you'll see `fatal error: 'algorithm' file not found`, and the
build falls back to the browser launcher. To enable the native window:

```bash
sudo rm -rf /Library/Developer/CommandLineTools
xcode-select --install
```

Then re‑run `./build-app.sh` — it will pick the WebView mode automatically.

## How it finds the CLI

The bundle ships the `rowback` CLI inside `Rowback.app/Contents/MacOS/` so the GUI
finds it next to itself. Outside a bundle, `resolveBinary` also checks the
executable's directory, its parents, and `PATH`.
