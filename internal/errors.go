package internal

import "errors"

var (
	// ErrProcessNotFound is returned when a process ID is not found in the registry
	ErrProcessNotFound = errors.New("process not found")

	// ErrProcessNotRunning is returned when trying to interact with a non-running process
	ErrProcessNotRunning = errors.New("process is not running")

	// ErrPermissionDenied is returned when the user lacks permission to perform an operation
	ErrPermissionDenied = errors.New("permission denied")

	// ErrResourceLimit is returned when hitting configured limits
	ErrResourceLimit = errors.New("resource limit exceeded")

	// ErrTimeout is returned when operations exceed timeout
	ErrTimeout = errors.New("operation timeout")

	// ErrInvalidInput is returned when parameters are invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrProcessAlreadyTerminated is returned when trying to operate on a terminated process
	ErrProcessAlreadyTerminated = errors.New("process already terminated")

	// ErrRegistryFull is returned when the process registry is full
	ErrRegistryFull = errors.New("process registry is full")
)
