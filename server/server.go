package server

import (
	"log"
	"net"
	"strings"

	"github.com/saifsuleman/gatekeeper/authentication"
	"github.com/saifsuleman/gatekeeper/config"
	"github.com/saifsuleman/gatekeeper/logger"
	"github.com/saifsuleman/gatekeeper/pipe"
)

// The struct for the main ProxyServer
// contains all the relevant data required to work
type ProxyServer struct {
	Address     string                           // the address this proxy server should listen on and accept incoming connections
	Redirect    string                           // the address this proxy server should pipe incoming connections to
	Connections map[net.Conn]pipe.ConnectionPipe // a map of all connections to connection pipe instances
	Auth        authentication.MultiFactorAuth   // the instance of the MultiFactorAuth object
	APIAddress  string                           // the address the API listener is listening on
}

// the main constructor for the ProxyServer struct
func NewProxyServer(config config.ApplicationConfig, logger logger.Logger) ProxyServer {
	// instantiates a new ProxyAuthHandler which is responsible for maintaining the list
	// of whitelisted IP addresses
	proxyAuthHandler, err := authentication.NewProxyAuthHandler("whitelist.json")
	// if an error is returned, throw the error which halts the program
	if err != nil {
		panic(err)
	}
	// instantiates a new MFA instance which is required for email alerts & more
	auth := authentication.NewMFA(proxyAuthHandler, logger, config.ApiWhitelist, config.DefaultApiUrl, config.Emails...)

	// constructs the struct and returns it
	return ProxyServer{
		Address:     config.ProxyAddress,
		Redirect:    config.RedirectAddress,
		Connections: map[net.Conn]pipe.ConnectionPipe{},
		Auth:        auth,
		APIAddress:  config.ApiAddress,
	}
}

// function used on the proxy server to begin listening
func (p *ProxyServer) Listen() {
	// in a goroutine it starts the MFA handler and starts the REST API listeners
	go p.Auth.Start(p.APIAddress)
	// creates a new TCP listener the proxy piping
	listener, err := net.Listen("tcp", p.Address)
	// if an error is returned, throw the error
	if err != nil {
		log.Fatalf("Error binding to address: %s\n", err)
		return
	}
	// in a while(true) loop
	for {
		// accept an incoming connection
		incoming, err := listener.Accept()

		// if an error is returned, do not throw the error, instead:
		// print the error and continue the loop
		if err != nil {
			log.Printf("Error accepting user: %s\n", err)
			continue
		}

		// in a goroutine handle the connection,
		// this is so its not thread blocking other incoming connections
		go p.handleConnection(incoming)
	}
}

// connection handler function which is called upon every connection to
// the proxy server's TCP listener
func (p *ProxyServer) handleConnection(conn net.Conn) {
	// gets the IP address of the incoming connection
	ip := GetIP(conn)

	// leverages the AuthHandler to determine whether or not this
	// IP address is whitelisted
	whitelisted := p.Auth.IsAuthenticated(ip)

	// if its not whitelisted, log this event and
	// close the connection and return
	if !whitelisted {
		log.Printf("Connection dialed from %s - IP not authenticated!\n", ip)
		_ = conn.Close()
		return
	}

	// log the successful connection
	log.Printf("Connection dialed from %s - IP authenticated!\n", ip)

	// dial TCP to the target service of this proxy (used for piping)
	redirect, err := net.Dial("tcp", p.Redirect)
	// if an error is returned, throw the error
	if err != nil {
		panic(err)
	}

	// instantiate a new connection pipe instance and pipe the incoming connection
	// and the dialed TCP connection to the target service
	connectionPipe := pipe.NewConnectionPipe(conn, redirect)

	// updates the connection map with the network connection as the key
	// and the connectionPipe as the value
	p.Connections[conn] = connectionPipe

	// as the connectionPipe.Pipe() is thread blocking, we can defer
	// the execution of deleting this from the map because we know that
	// this host function will only end once the connection pipe has been terminated
	// (it's quite smart really)
	defer delete(p.Connections, conn)

	// using our connection pipe instance, we begin piping the connection
	connectionPipe.Pipe()
}

// function to get an IP address of an existing network connection
// it takes the whole IP address and splits it at the colon and then returns the
// LHS (left-hand side) of that statement - for example: 51.146.6.229:5274 -> 51.146.6.229
func GetIP(conn net.Conn) string {
	return strings.Split(conn.RemoteAddr().String(), ":")[0]
}

func testConnectionPiping() {
	listener, err := net.Listen("tcp", ":8080")
	// start patch 1
	if err != nil {
		panic(err)
	}
	// end patch 1
	for {
		incoming, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		redirect, err := net.Dial("tcp", "127.0.0.1:3389")
		if err != nil {
			panic(err)
		}
		connectionPipe := pipe.NewConnectionPipe(incoming, redirect)
		connectionPipe.Pipe()
	}
}
