package debug

import (
	"fmt"
	"os"
)

// Printf prints debug messages to stderr
func Printf(format string, args ...interface{}) {
	if os.Getenv("GOTMPL_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
