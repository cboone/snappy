package tui

import "github.com/cboone/snappy/internal/service"

// ServiceController abstracts launchd service operations for testability.
type ServiceController interface {
	Status(label string) (*service.Info, error)
	Start(label string) error
	Stop(label string) error
	Install(cfg service.PlistConfig) error
	Uninstall(label string) error
	ResolveBinaryPath() (string, error)
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

// Install writes the plist and bootstraps the launchd agent.
func (LaunchdController) Install(cfg service.PlistConfig) error {
	return service.Install(cfg)
}

// Uninstall stops and removes the launchd agent.
func (LaunchdController) Uninstall(label string) error {
	return service.Uninstall(label)
}

// ResolveBinaryPath returns the canonical path of the running executable.
func (LaunchdController) ResolveBinaryPath() (string, error) {
	return service.ResolveBinaryPath()
}
