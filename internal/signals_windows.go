//go:build windows

package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// parseSignal converts a signal string to os.Signal, with Windows-specific limitations
func parseSignal(signal string) (os.Signal, error) {
	var sig os.Signal
	switch strings.ToUpper(signal) {
	case "SIGTERM":
		sig = syscall.SIGTERM
	case "SIGKILL":
		sig = syscall.SIGKILL
	case "SIGINT":
		sig = syscall.SIGINT
	case "SIGSTOP", "SIGCONT", "SIGUSR1", "SIGUSR2":
		// These signals are not supported on Windows
		return nil, fmt.Errorf("invalid signal: %s (not supported on Windows)", signal)
	default:
		// Try to parse as number
		if num, err := strconv.Atoi(signal); err == nil {
			sig = syscall.Signal(num)
		} else {
			return nil, fmt.Errorf("invalid signal: %s", signal)
		}
	}
	return sig, nil
}