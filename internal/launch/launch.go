//go:build !windows

package launch

import (
	"fmt"
	"os"
	"syscall"
)

// Exec replaces the current process with the agent binary so the agent
// inherits this terminal, cwd and environment seamlessly.
func Exec(path string, args []string) error {
	argv := append([]string{path}, args...)
	if err := syscall.Exec(path, argv, os.Environ()); err != nil {
		return fmt.Errorf("failed to launch %s: %w", path, err)
	}
	return nil // unreachable
}
