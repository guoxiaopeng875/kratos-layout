package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers for the main service (gRPC + HTTP with biz routes).
var ProviderSet = wire.NewSet(NewGRPCServer, NewHTTPServer)

// JobProviderSet is server providers for the job process (HTTP exposing
// /metrics and /health only).
var JobProviderSet = wire.NewSet(NewJobHTTPServer)
