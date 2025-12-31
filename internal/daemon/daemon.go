package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/MycelicMemory/ultrathink/internal/logging"
)

var log = logging.GetLogger("daemon")

const (
	PIDFileName   = "ultrathink.pid"
	StateFileName = "ultrathink.state"
)

// State represents the daemon state persisted to disk
type State struct {
	PID         int       `json:"pid"`
	StartTime   time.Time `json:"start_time"`
	Version     string    `json:"version"`
	RESTEnabled bool      `json:"rest_enabled"`
	RESTHost    string    `json:"rest_host"`
	RESTPort    int       `json:"rest_port"`
	MCPEnabled  bool      `json:"mcp_enabled"`
}

// Status represents the current daemon status
type Status struct {
	Running     bool          `json:"running"`
	PID         int           `json:"pid,omitempty"`
	Uptime      time.Duration `json:"uptime,omitempty"`
	Version     string        `json:"version,omitempty"`
	RESTEnabled bool          `json:"rest_enabled,omitempty"`
	RESTHost    string        `json:"rest_host,omitempty"`
	RESTPort    int           `json:"rest_port,omitempty"`
	MCPEnabled  bool          `json:"mcp_enabled,omitempty"`
}

// Daemon manages the ultrathink daemon lifecycle
type Daemon struct {
	configDir string
	version   string
}

// New creates a new Daemon instance
func New(configDir, version string) *Daemon {
	return &Daemon{
		configDir: configDir,
		version:   version,
	}
}

// PIDPath returns the path to the PID file
func (d *Daemon) PIDPath() string {
	return filepath.Join(d.configDir, PIDFileName)
}

// StatePath returns the path to the state file
func (d *Daemon) StatePath() string {
	return filepath.Join(d.configDir, StateFileName)
}

// WritePID writes the current process PID to the PID file
func (d *Daemon) WritePID() error {
	pid := os.Getpid()
	return os.WriteFile(d.PIDPath(), []byte(strconv.Itoa(pid)), 0644)
}

// ReadPID reads the PID from the PID file
func (d *Daemon) ReadPID() (int, error) {
	data, err := os.ReadFile(d.PIDPath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// RemovePID removes the PID file
func (d *Daemon) RemovePID() error {
	return os.Remove(d.PIDPath())
}

// WriteState writes the daemon state to disk
func (d *Daemon) WriteState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d.StatePath(), data, 0644)
}

// ReadState reads the daemon state from disk
func (d *Daemon) ReadState() (*State, error) {
	data, err := os.ReadFile(d.StatePath())
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// RemoveState removes the state file
func (d *Daemon) RemoveState() error {
	return os.Remove(d.StatePath())
}

// IsRunning checks if the daemon is currently running
func (d *Daemon) IsRunning() bool {
	pid, err := d.ReadPID()
	if err != nil {
		return false
	}
	return d.isProcessRunning(pid)
}

// isProcessRunning checks if a process with the given PID is running
func (d *Daemon) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// Status returns the current daemon status
func (d *Daemon) Status() *Status {
	status := &Status{Running: false}

	pid, err := d.ReadPID()
	if err != nil {
		return status
	}

	if !d.isProcessRunning(pid) {
		// Clean up stale PID file
		d.RemovePID()
		d.RemoveState()
		return status
	}

	status.Running = true
	status.PID = pid

	// Read state for additional info
	state, err := d.ReadState()
	if err == nil {
		status.Version = state.Version
		status.RESTEnabled = state.RESTEnabled
		status.RESTHost = state.RESTHost
		status.RESTPort = state.RESTPort
		status.MCPEnabled = state.MCPEnabled
		status.Uptime = time.Since(state.StartTime)
	}

	return status
}

// Start starts the daemon (saves state, writes PID)
func (d *Daemon) Start(restEnabled bool, restHost string, restPort int, mcpEnabled bool) error {
	log.Info("starting daemon", "rest_enabled", restEnabled, "mcp_enabled", mcpEnabled)

	if d.IsRunning() {
		log.Warn("daemon is already running")
		return fmt.Errorf("daemon is already running")
	}

	// Write PID file
	if err := d.WritePID(); err != nil {
		log.Error("failed to write PID file", "error", err)
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Write state
	state := &State{
		PID:         os.Getpid(),
		StartTime:   time.Now(),
		Version:     d.version,
		RESTEnabled: restEnabled,
		RESTHost:    restHost,
		RESTPort:    restPort,
		MCPEnabled:  mcpEnabled,
	}

	if err := d.WriteState(state); err != nil {
		d.RemovePID()
		log.Error("failed to write state file", "error", err)
		return fmt.Errorf("failed to write state file: %w", err)
	}

	log.Info("daemon started", "pid", state.PID, "version", d.version)
	return nil
}

// Stop stops the daemon by sending SIGTERM
func (d *Daemon) Stop() error {
	log.Info("stopping daemon")

	pid, err := d.ReadPID()
	if err != nil {
		log.Debug("no PID file found")
		return fmt.Errorf("daemon is not running (no PID file)")
	}

	if !d.isProcessRunning(pid) {
		log.Debug("stale PID file, cleaning up", "pid", pid)
		d.RemovePID()
		d.RemoveState()
		return fmt.Errorf("daemon is not running (stale PID file)")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		log.Error("failed to find process", "error", err, "pid", pid)
		return fmt.Errorf("failed to find process: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	log.Debug("sending SIGTERM", "pid", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		log.Error("failed to send SIGTERM", "error", err)
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit (with timeout)
	for i := 0; i < 50; i++ { // 5 second timeout
		if !d.isProcessRunning(pid) {
			d.RemovePID()
			d.RemoveState()
			log.Info("daemon stopped gracefully", "pid", pid)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still running
	log.Warn("daemon did not stop gracefully, sending SIGKILL", "pid", pid)
	if err := process.Signal(syscall.SIGKILL); err != nil {
		log.Error("failed to send SIGKILL", "error", err)
		return fmt.Errorf("failed to send SIGKILL: %w", err)
	}

	d.RemovePID()
	d.RemoveState()
	log.Info("daemon killed", "pid", pid)
	return nil
}

// Cleanup removes PID and state files (called on graceful shutdown)
func (d *Daemon) Cleanup() {
	d.RemovePID()
	d.RemoveState()
}

// Daemonize forks the current process and runs it as a daemon
// Returns true if we're in the child (daemon) process, false if parent
func (d *Daemon) Daemonize(args []string) (bool, error) {
	if d.IsRunning() {
		return false, fmt.Errorf("daemon is already running")
	}

	// Get the executable path
	executable, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Start the daemon process
	cmd := exec.Command(executable, args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Set process group so child doesn't get killed with parent
	setProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("failed to start daemon: %w", err)
	}

	return false, nil // Parent process returns
}

// ListProcesses returns a list of running ultrathink processes
func (d *Daemon) ListProcesses() ([]ProcessInfo, error) {
	var processes []ProcessInfo

	// Check main daemon
	status := d.Status()
	if status.Running {
		processes = append(processes, ProcessInfo{
			PID:     status.PID,
			Type:    "daemon",
			Uptime:  status.Uptime,
			Version: status.Version,
		})
	}

	return processes, nil
}

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	PID     int           `json:"pid"`
	Type    string        `json:"type"`
	Uptime  time.Duration `json:"uptime"`
	Version string        `json:"version"`
}

// KillAll kills all ultrathink processes
func (d *Daemon) KillAll() (int, error) {
	killed := 0

	// First try to stop the main daemon gracefully
	if d.IsRunning() {
		if err := d.Stop(); err == nil {
			killed++
		}
	}

	return killed, nil
}
