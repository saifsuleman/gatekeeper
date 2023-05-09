package main

import (
	"github.com/saifsuleman/gatekeeper/config"
	"github.com/saifsuleman/gatekeeper/logger"
	"github.com/saifsuleman/gatekeeper/server"
)

func main() {
	appConfig, err := config.NewApplicationConfig("config.json")
	if err != nil {
		panic(err)
	}
	l := logger.InitializeLogger(appConfig.LoggerPath)
	proxyServer := server.NewProxyServer(appConfig, l)
	proxyServer.Listen()
}
