package server

import (
	"fmt"
	"github.com/saifsuleman/gatekeeper/authentication"
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
}

func NewProxyServer(address string, redirect string, emails ...string) ProxyServer {
	proxyAuthHandler, err := authentication.NewProxyAuthHandler("whitelist.json")
	if err != nil {
		panic(err)
	}
	auth := authentication.NewMFA(proxyAuthHandler, emails...)

	return ProxyServer{
		Address:     address,
		Redirect:    redirect,
		Connections: map[net.Conn]pipe.ConnectionPipe{},
		Auth:        auth,
	}
}

func (p *ProxyServer) Listen() {
	go p.Auth.Start(":8182")
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
	whitelisted := p.Auth.IsAuthenticated(GetIP(conn))
	if !whitelisted {
		fmt.Println("Proxy dialed - noauth!")
		return
	}
	fmt.Println("Successful proxy dialed!")

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
