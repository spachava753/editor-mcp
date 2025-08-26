//go:build unix

package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// parseSignal converts a signal string to os.Signal, with Unix-specific signals
func parseSignal(signal string) (os.Signal, error) {
	var sig os.Signal
	switch strings.ToUpper(signal) {
	case "SIGTERM":
		sig = syscall.SIGTERM
	case "SIGKILL":
		sig = syscall.SIGKILL
	case "SIGINT":
		sig = syscall.SIGINT
	case "SIGSTOP":
		sig = syscall.SIGSTOP
	case "SIGCONT":
		sig = syscall.SIGCONT
	case "SIGUSR1":
		sig = syscall.SIGUSR1
	case "SIGUSR2":
		sig = syscall.SIGUSR2
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