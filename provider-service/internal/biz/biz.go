package biz

import "github.com/google/wire"

// ProviderSet is biz providers for wire
var ProviderSet = wire.NewSet(NewProviderUsecase)
