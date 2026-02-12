package job

import (
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/wire"
)

// Registry holds all background jobs for Kratos lifecycle management.
type Registry struct {
}

// Servers returns all jobs as transport.Server slice for kratos.Server().
func (r *Registry) Servers() []transport.Server {
	return []transport.Server{}
}

// ProviderSet is the job providers.
var ProviderSet = wire.NewSet(
	wire.Struct(new(Registry), "*"),
)
