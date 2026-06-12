//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/job"
	"github.com/go-kratos/kratos-layout/internal/server"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Server, []*conf.CronTask, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.JobProviderSet, job.ProviderSet, newApp))
}
