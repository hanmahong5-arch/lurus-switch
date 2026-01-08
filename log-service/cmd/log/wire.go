//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/pocketzworld/lurus-switch/log-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/log-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/log-service/internal/consumer"
	"github.com/pocketzworld/lurus-switch/log-service/internal/data"
	"github.com/pocketzworld/lurus-switch/log-service/internal/server"
	"github.com/pocketzworld/lurus-switch/log-service/internal/service"
	"go.uber.org/zap"
)

// wireApp initializes the application with Wire
func wireApp(*conf.Bootstrap, *zap.Logger) (*App, func(), error) {
	panic(wire.Build(
		data.NewData,
		data.NewLogRepo,
		biz.NewLogUsecase,
		service.NewLogService,
		server.NewHTTPServer,
		consumer.NewNATSConsumer,
		newApp,
	))
}
