package main

import (
	"io"
	"os"
)

// stderr exists so the per-OS files can pass the process's stderr through
// without each importing os for one line.
func stderr() io.Writer { return os.Stderr }
