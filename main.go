package main

import (
	"github.com/saifsuleman/gatekeeper/server"
)

func main() {
	proxyServer := server.NewProxyServer(":7777",  "127.0.0.1:3389", ":8182", "saif@visionituk.com", "attiques@visionituk.com")
	proxyServer.Listen()
}
