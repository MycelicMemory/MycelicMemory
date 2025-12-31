package benchmark

import "errors"

var (
	// ErrDirtyWorkTree is returned when git working tree has uncommitted changes
	ErrDirtyWorkTree = errors.New("git working tree is dirty - commit or stash changes first")

	// ErrRunNotFound is returned when a benchmark run is not found
	ErrRunNotFound = errors.New("benchmark run not found")

	// ErrRunAlreadyRunning is returned when trying to start a run while one is running
	ErrRunAlreadyRunning = errors.New("a benchmark run is already in progress")

	// ErrNoLoopRunning is returned when trying to stop a loop that isn't running
	ErrNoLoopRunning = errors.New("no autonomous loop is currently running")

	// ErrLoopAlreadyRunning is returned when trying to start a loop while one is running
	ErrLoopAlreadyRunning = errors.New("an autonomous loop is already running")

	// ErrPythonBridgeNotAvailable is returned when the Python bridge server is not responding
	ErrPythonBridgeNotAvailable = errors.New("Python benchmark bridge is not available - start it with 'make server' in benchmark/locomo/")

	// ErrBenchmarkFailed is returned when benchmark execution fails
	ErrBenchmarkFailed = errors.New("benchmark execution failed")

	// ErrInvalidBenchmarkType is returned for unsupported benchmark types
	ErrInvalidBenchmarkType = errors.New("invalid benchmark type - supported: locomo")

	// ErrMaxIterationsReached is returned when loop reaches max iterations
	ErrMaxIterationsReached = errors.New("maximum iterations reached")

	// ErrConverged is returned when loop converges (minimal improvement)
	ErrConverged = errors.New("loop converged - improvements are below threshold")

	// ErrNoImprovement is returned when multiple consecutive iterations show no improvement
	ErrNoImprovement = errors.New("no improvement after multiple iterations")
)
