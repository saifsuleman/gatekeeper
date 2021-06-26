package server

import (
	"github.com/saifsuleman/gatekeeper/pipe"
	"log"
	"net"
)

type ProxyServer struct {
	Address     string
	Redirect    string
	Connections map[net.Conn]pipe.ConnectionPipe
}

func NewProxyServer(address string, redirect string) ProxyServer {
	return ProxyServer{
		Address:     address,
		Redirect:    redirect,
		Connections: map[net.Conn]pipe.ConnectionPipe{},
	}
}

func (p *ProxyServer) Listen() {
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
	log.Printf("Connection dialed from: %s\n", conn.RemoteAddr().String())

	redirect, err := net.Dial("tcp", p.Redirect)
	if err != nil {
		panic(err)
	}
	connectionPipe := pipe.NewConnectionPipe(conn,redirect)
	p.Connections[conn] = connectionPipe
	defer delete(p.Connections, conn)

	connectionPipe.Pipe()
}
