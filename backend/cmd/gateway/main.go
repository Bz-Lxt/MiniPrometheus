package main

import (
	"os"

	"github.com/alkaid/miniprometheus/internal/api"
	"github.com/alkaid/miniprometheus/internal/config"
	"github.com/alkaid/miniprometheus/internal/logger"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, nil)
	if err := api.RunGateway(cfg); err != nil {
		logger.L().Error("gateway exit", "err", err.Error())
		os.Exit(1)
	}
}
