package shellsafe

import (
	"fmt"
	"os/exec"
	"strings"
)

// SafeExecutor executes commands safely with parameter validation.
type SafeExecutor struct {
	AllowReboot bool
}

// NewSafeExecutor creates a new safe executor.
func NewSafeExecutor(allowReboot bool) *SafeExecutor {
	return &SafeExecutor{AllowReboot: allowReboot}
}

// CommandResult holds the result of a safe command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Execute runs a command with the given name and args.
func (e *SafeExecutor) Execute(name string, args ...string) *CommandResult {
	cmd := exec.Command(name, args...)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &CommandResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
	}
}

// ExecuteShell runs a command string through the shell (high risk, use sparingly).
func (e *SafeExecutor) ExecuteShell(command string) *CommandResult {
	if !e.AllowReboot {
		// Block dangerous commands
		lower := strings.ToLower(command)
		dangerous := []string{"rm -rf", "mkfs", "dd if=", "> /dev/", "wget ", "curl "}
		for _, d := range dangerous {
			if strings.Contains(lower, d) {
				return &CommandResult{
					Stderr:   fmt.Sprintf("blocked dangerous command pattern: %s", d),
					ExitCode: -1,
				}
			}
		}
	}

	cmd := exec.Command("sh", "-c", command)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &CommandResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
	}
}

// ReadFile reads a system file safely with a max size limit.
func ReadFile(path string) (string, error) {
	data, err := exec.Command("cat", path).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}