//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/data"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/server"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/service"
	"go.uber.org/zap"
)

// wireApp initializes the application with Wire
func wireApp(*conf.Bootstrap, *zap.Logger) (*App, func(), error) {
	panic(wire.Build(
		data.NewData,
		data.NewProviderRepo,
		data.NewProviderCache,
		biz.NewProviderUsecase,
		service.NewProviderService,
		server.NewHTTPServer,
		newApp,
	))
}
