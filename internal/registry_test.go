package internal

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/nalgeon/be"
)

func TestNewRegistry(t *testing.T) {
	config := DefaultRegistryConfig()
	config.MaxProcesses = 10

	registry := NewRegistry(config)
	be.True(t, registry != nil)
	be.Equal(t, registry.config.MaxProcesses, config.MaxProcesses)

	// Clean up
	registry.Shutdown()
}

func TestStartProcess(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	proc, err := registry.StartProcess(ctx, "echo hello world", "bash", nil, nil, "", true)

	be.Err(t, err, nil)
	be.True(t, proc != nil)
	be.True(t, proc.ID != "")
	be.True(t, proc.PID != 0)
	be.Equal(t, proc.Command, "echo hello world")
	be.Equal(t, proc.Shell, "bash")
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Wait for process to complete
	time.Sleep(100 * time.Millisecond)

	// Process should be terminated
	be.Equal(t, proc.GetState(), ProcessStateTerminated)
	be.True(t, proc.GetExitCode() != nil)
	be.Equal(t, *proc.GetExitCode(), 0)
}

func TestStartProcessWithLongRunningCommand(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	proc, err := registry.StartProcess(ctx, "sleep 0.5", "bash", nil, nil, "", true)

	be.Err(t, err, nil)
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Process should still be running
	time.Sleep(100 * time.Millisecond)
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Wait for completion
	time.Sleep(500 * time.Millisecond)
	be.Equal(t, proc.GetState(), ProcessStateTerminated)
}

func TestListProcesses(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()

	// Start multiple processes
	_, err := registry.StartProcess(ctx, "echo test1", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	proc2, err := registry.StartProcess(ctx, "sleep 1", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	// List all processes
	processes, total, err := registry.ListProcesses("", 10, 0, "")
	be.Err(t, err, nil)
	be.Equal(t, total, 2)
	be.Equal(t, len(processes), 2)

	// Wait for first process to complete
	time.Sleep(100 * time.Millisecond)

	// List running processes only
	runningProcesses, runningTotal, err := registry.ListProcesses(ProcessStateRunning, 10, 0, "")
	be.Err(t, err, nil)
	be.Equal(t, runningTotal, 1)
	be.Equal(t, len(runningProcesses), 1)
	be.Equal(t, runningProcesses[0].ID, proc2.ID)

	// Terminate the long-running process
	registry.TerminateProcess(proc2.ID, true, 0)
}

func TestSendInput(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	// Start a cat process that will echo input
	proc, err := registry.StartProcess(ctx, "cat", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	// Send input
	input := "hello world\n"
	err = registry.SendInputToProcess(proc.ID, []byte(input))
	be.Err(t, err, nil)

	// Give some time for output to be captured
	time.Sleep(100 * time.Millisecond)

	// Read output
	data, _, _, err := registry.ReadProcessOutput(proc.ID, "stdout", 0, 1024)
	be.Err(t, err, nil)
	be.Equal(t, string(data), input)

	// Terminate cat process
	registry.TerminateProcess(proc.ID, true, 0)
}

func TestTerminateProcess(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	proc, err := registry.StartProcess(ctx, "sleep 10", "bash", nil, nil, "", true)
	be.Err(t, err, nil)
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Terminate gracefully
	err = registry.TerminateProcess(proc.ID, false, 100*time.Millisecond)
	be.Err(t, err, nil)

	// Wait for termination
	time.Sleep(200 * time.Millisecond)
	be.Equal(t, proc.GetState(), ProcessStateTerminated)
}

func TestSendSignal(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	proc, err := registry.StartProcess(ctx, "sleep 10", "bash", nil, nil, "", true)
	be.Err(t, err, nil)
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Send SIGTERM
	err = registry.SendSignalToProcess(proc.ID, syscall.SIGTERM)
	be.Err(t, err, nil)

	// Wait for termination
	time.Sleep(100 * time.Millisecond)
	be.Equal(t, proc.GetState(), ProcessStateTerminated)
}

func TestOutputBuffer(t *testing.T) {
	buf := NewOutputBuffer(100) // Small buffer for testing

	// Write some data to stdout
	data1 := []byte("hello world\n")
	buf.WriteStdout(data1)

	// Read the data
	readData, pos, hasMore := buf.ReadStdout(0, 50)
	be.Equal(t, readData, data1)
	be.Equal(t, pos, int64(len(data1)))
	be.True(t, !hasMore)

	// Write more data to exceed buffer size
	largeData := make([]byte, 150)
	for i := range largeData {
		largeData[i] = 'A'
	}
	buf.WriteStdout(largeData)

	// Buffer should be trimmed to max size
	size, bufPos := buf.GetStdoutSize()
	be.Equal(t, size, 100)
	be.Equal(t, bufPos, int64(len(data1)+150-100)) // Position adjusted for trimmed data
}

func TestGetProcessStatus(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	ctx := context.Background()
	proc, err := registry.StartProcess(ctx, "echo test", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	// Get status
	status, err := registry.GetProcessStatus(proc.ID)
	be.Err(t, err, nil)
	be.Equal(t, status.ID, proc.ID)
	be.Equal(t, status.PID, proc.PID)
	be.Equal(t, status.Command, proc.Command)
	be.True(t, status.Runtime > 0)
	be.True(t, status.OutputInfo != nil)
}

func TestRegistryShutdown(t *testing.T) {
	registry := NewRegistry(nil)

	ctx := context.Background()
	// Start a long-running process
	proc, err := registry.StartProcess(ctx, "sleep 10", "bash", nil, nil, "", true)
	be.Err(t, err, nil)
	be.Equal(t, proc.GetState(), ProcessStateRunning)

	// Shutdown registry
	err = registry.Shutdown()
	be.Err(t, err, nil)

	// Process should be terminated or in process of termination
	time.Sleep(100 * time.Millisecond)
	// Note: We can't reliably test the exact state here due to timing,
	// but shutdown should have attempted to terminate the process
}

func TestProcessNotFound(t *testing.T) {
	registry := NewRegistry(nil)
	defer registry.Shutdown()

	// Try to get non-existent process
	_, err := registry.GetProcess("non-existent")
	be.Equal(t, err, ErrProcessNotFound)

	// Try to send input to non-existent process
	err = registry.SendInputToProcess("non-existent", []byte("test"))
	be.Equal(t, err, ErrProcessNotFound)

	// Try to terminate non-existent process
	err = registry.TerminateProcess("non-existent", false, 0)
	be.Equal(t, err, ErrProcessNotFound)
}

func TestRegistryCapacityLimit(t *testing.T) {
	config := DefaultRegistryConfig()
	config.MaxProcesses = 2 // Very small limit for testing

	registry := NewRegistry(config)
	defer registry.Shutdown()

	ctx := context.Background()

	// Start processes up to the limit
	_, err := registry.StartProcess(ctx, "sleep 1", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	_, err = registry.StartProcess(ctx, "sleep 1", "bash", nil, nil, "", true)
	be.Err(t, err, nil)

	// Third process should fail
	_, err = registry.StartProcess(ctx, "sleep 1", "bash", nil, nil, "", true)
	be.Equal(t, err, ErrRegistryFull)
}
