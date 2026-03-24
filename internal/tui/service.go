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

// Start enables and starts the agent (kickstart), falling back to bootstrap if needed.
func (LaunchdController) Start(label string) error {
	return service.Start(label)
}

// Stop disables and unloads the launchd agent via bootout.
func (LaunchdController) Stop(label string) error {
	return service.Stop(label)
}
