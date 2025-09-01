package environment

import (
	"dice-sorensen-similarity-search/internal/database"
	"dice-sorensen-similarity-search/internal/logging"
)

// Env provides access to shared resources such as the database repository and logging interface.
// It is typically embedded in higher-level components that require infrastructure dependencies.
type Env struct {
	database.Repository
	logging.Logger
}

// Environment constructs a new Env instance using the provided database repository and logger.
// If either parameter is nil, a no-op implementation is substituted.
//
// param repository a database repository implementation or nil
// param logger a logging implementation or nil
// return a fully initialized *Env with fallback defaults
func Environment(repository database.Repository, logger logging.Logger) *Env {
	if repository == nil {
		repository = &database.NullRepository{}
	}

	if logger == nil {
		logger = &logging.NullLogger{}
	}

	return &Env{repository, logger}
}

// Null returns an Env with no-op implementations for both repository and logger.
// Useful for testing or as a stub in wiring graphs.
func Null() *Env {
	return Environment(nil, nil)
}
