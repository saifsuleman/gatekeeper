package pipe

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"testing"
)

const (
	PAYLOAD_LENGTH = 8192
)

func TestConnectionPiping(t *testing.T) {
	proxyPortChan := make(chan int)
	backendPortChan := make(chan int)

	serverToClientPayload := make([]byte, PAYLOAD_LENGTH)
	clientToServerPayload := make([]byte, PAYLOAD_LENGTH)

	_, err := rand.Read(serverToClientPayload)
	if err != nil {
		t.Error(err)
	}

	_, err = rand.Read(clientToServerPayload)
	if err != nil {
		t.Error(err)
	}

	go (func() {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Error(err)
		}
		defer listener.Close()

		backendPortChan <- listener.Addr().(*net.TCPAddr).Port

		incoming, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		defer incoming.Close()

		incoming.Write(serverToClientPayload)

		data := &bytes.Buffer{}
		_, err = io.CopyN(io.MultiWriter(data, io.Discard), incoming, PAYLOAD_LENGTH)
		if err != nil && err != io.EOF {
			t.Error(err)
		}

		if !bytes.Equal(data.Bytes(), clientToServerPayload) {
			t.Error("invalid payload sent from client -> server")
		}
	})()

	backendPort := <-backendPortChan

	go (func() {

		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Error(err)
		}
		defer listener.Close()

		proxyPortChan <- listener.Addr().(*net.TCPAddr).Port

		incoming, err := listener.Accept()
		if err != nil {
			t.Error(err)
		}
		defer incoming.Close()
		redirect, err := net.Dial("tcp", fmt.Sprintf(":%d", backendPort))
		if err != nil {
			t.Error(err)
		}
		pipe := NewConnectionPipe(incoming, redirect)
		pipe.Pipe()
	})()

	proxyPort := <-proxyPortChan

	t.Logf("Initializing connection piping test with proxy running on port %d and backend running on port %d", proxyPort, backendPort)

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", proxyPort))
	if err != nil {
		t.Error(err)
	}

	data := &bytes.Buffer{}
	_, err = io.CopyN(io.MultiWriter(data, io.Discard), conn, PAYLOAD_LENGTH)
	if err != nil && err != io.EOF {
		t.Error(err)
	}
	if !bytes.Equal(data.Bytes(), serverToClientPayload) {
		t.Errorf("invalid payload sent from server -> client\n SENT: %v\n EXPECTING: %v", data, serverToClientPayload)
	}

	conn.Write(clientToServerPayload)
}
