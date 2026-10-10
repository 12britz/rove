//go:build windows

package launch

import "fmt"

// Exec is not supported on Windows (no syscall.Exec); the caller should
// fall back to spawning the child process instead.
func Exec(path string, args []string) error {
	return fmt.Errorf("exec replacement is not supported on windows (target %s)", path)
}
