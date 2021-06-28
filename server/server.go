package server

import (
	"github.com/saifsuleman/gatekeeper/authentication"
	"github.com/saifsuleman/gatekeeper/config"
	"github.com/saifsuleman/gatekeeper/pipe"
	"log"
	"net"
	"strings"
)

type ProxyServer struct {
	Address     string
	Redirect    string
	Connections map[net.Conn]pipe.ConnectionPipe
	Auth        authentication.MultiFactorAuth
	APIAddress  string
}

func NewProxyServer(config config.ApplicationConfig) ProxyServer {
	proxyAuthHandler, err := authentication.NewProxyAuthHandler("whitelist.json")
	if err != nil {
		panic(err)
	}
	auth := authentication.NewMFA(proxyAuthHandler, config.DefaultApiUrl, config.Emails...)

	return ProxyServer{
		Address:     config.ProxyAddress,
		Redirect:    config.RedirectAddress,
		Connections: map[net.Conn]pipe.ConnectionPipe{},
		Auth:        auth,
		APIAddress:  config.ApiAddress,
	}
}

func (p *ProxyServer) Listen() {
	go p.Auth.Start(p.APIAddress)
	listener, err := net.Listen("tcp", p.Address)
	if err != nil {
		log.Fatalf("Error binding to address: %s\n", err)
		return
	}
	for {
		incoming, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting user: %s\n", err)
			continue
		}
		go p.handleConnection(incoming)
	}
}

func (p *ProxyServer) handleConnection(conn net.Conn) {
	ip := GetIP(conn)
	whitelisted := p.Auth.IsAuthenticated(ip)
	if !whitelisted {
		log.Printf("Connection dialed from %s - IP not authenticated!\n", ip)
		_ = conn.Close()
		return
	}
	log.Printf("Connection dialed from %s - IP authenticated!\n", ip)

	redirect, err := net.Dial("tcp", p.Redirect)
	if err != nil {
		panic(err)
	}
	connectionPipe := pipe.NewConnectionPipe(conn, redirect)
	p.Connections[conn] = connectionPipe
	defer delete(p.Connections, conn)

	connectionPipe.Pipe()
}

func GetIP(conn net.Conn) string {
	return strings.Split(conn.RemoteAddr().String(), ":")[0]
}
