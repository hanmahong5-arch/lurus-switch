package server

import "github.com/google/wire"

// ProviderSet is server providers for wire
var ProviderSet = wire.NewSet(NewHTTPServer)
