package app

import (
	"embed"
	"io/fs"
)

// webFS holds the embedded production frontend (HTML, CSS, JS). The assets live
// as real files under internal/app/web/ and are compiled into the binary, so
// rowback ships as a single self-contained executable with no runtime
// dependencies and no UI markup buried in Go string literals.
//
//go:embed web/index.html web/app.css web/app.js
var webFS embed.FS

// webRoot is webFS rooted at the web/ directory so it serves "/index.html",
// "/app.css", and "/app.js" at the URL root.
func webRoot() fs.FS {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		// Unreachable: the embed pattern guarantees web/ exists at build time.
		panic("rowback: embedded web assets missing: " + err.Error())
	}
	return sub
}
