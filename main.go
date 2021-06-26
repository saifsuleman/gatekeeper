package main

import (
	"github.com/saifsuleman/gatekeeper/server"
)

func main() {
	proxyServer := server.NewProxyServer(":7860", "127.0.0.1:3389")
	proxyServer.Listen()
}
