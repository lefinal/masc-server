package services

import "context"

// Service is an interface for all services that can be run in app.App.
type Service interface {
	// Run the Service until the given context.Context is done.
	Run(ctx context.Context) error
}
