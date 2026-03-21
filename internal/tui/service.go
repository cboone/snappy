package tui

import "github.com/cboone/snappy/internal/service"

// ServiceController abstracts launchd service operations for testability.
type ServiceController interface {
	Status(label string) (*service.Info, error)
	Start(label string) error
	Stop(label string) error
}

// LaunchdController implements ServiceController using the real service package.
type LaunchdController struct{}

// Status queries launchctl for the current state of the agent.
func (LaunchdController) Status(label string) (*service.Info, error) {
	return service.Status(label)
}

// Start sends a kickstart signal to the agent.
func (LaunchdController) Start(label string) error {
	return service.Start(label)
}

// Stop sends SIGTERM to the agent process.
func (LaunchdController) Stop(label string) error {
	return service.Stop(label)
}
