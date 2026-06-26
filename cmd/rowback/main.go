// Command rowback surgically restores deleted rows from a massive MySQL/MariaDB
// dump. See https://github.com/albertovincenzi/rowback.
package main

import (
	"os"

	"github.com/albertovincenzi/rowback/internal/app"
)

func main() {
	os.Exit(app.Run())
}
