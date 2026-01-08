package service

import "github.com/google/wire"

// ProviderSet is service providers for wire
var ProviderSet = wire.NewSet(NewProviderService)
