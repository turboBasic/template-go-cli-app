//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package output

import "os"

func isTerminalFile(*os.File) bool { return false }
