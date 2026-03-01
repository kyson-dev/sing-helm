package app

import (
	"log/slog"

	"github.com/kyson-dev/sing-helm/internal/sys/paths"
)

// Application is the central dependency holder for the entire program.
// All business components obtain their dependencies from this struct,
// avoiding global singletons.
type Application struct {
	Paths  paths.Paths
	Logger *slog.Logger
}

// New creates an Application instance by resolving paths and setting up logging.
func New(paths paths.Paths, logger *slog.Logger) *Application {
	return &Application{
		Paths:  paths,
		Logger: logger,
	}
}
