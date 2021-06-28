package main

import (
	"github.com/saifsuleman/gatekeeper/config"
	"github.com/saifsuleman/gatekeeper/logger"
	"github.com/saifsuleman/gatekeeper/server"
)

func main() {
	appConfig := config.NewApplicationConfig("config.json")
	logger.InitializeLogger(appConfig.LoggerPath)
	proxyServer := server.NewProxyServer(appConfig)
	proxyServer.Listen()
}
