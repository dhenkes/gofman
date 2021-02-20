package gofman

import (
	"context"
)

// SetupService represents a service for managing the setup process. It should
// be called every time the application starts up to see if the setup handlers
// need to be added to the routes
type SetupService interface {
	ShouldRunSetup(ctx context.Context) (bool, error)
}
