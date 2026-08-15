package wire

import (
	healthcheckwire "github.com/jasonlabz/generate-example-project/server/wire/health_check"
)

// Modules is the application composition registry. Router only depends on
// this registry; each entry is assembled by its own module Wire package.
func Modules() []Module {
	return []Module{
		healthcheckwire.NewModule(),
	}
}
