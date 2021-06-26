package pipe

import "net"

type ConnectionPipe struct {
	Alive bool
	Left net.Conn
	Right net.Conn
}

func NewConnectionPipe(left net.Conn, right net.Conn) ConnectionPipe {
	return ConnectionPipe{
		Alive: false,
		Left:  left,
		Right: right,
	}
}

func (cp *ConnectionPipe) pipeConnection(read net.Conn, write net.Conn) {
	buf := make([]byte, 2048)
	for cp.Alive {
		length, err := read.Read(buf)
		if err != nil || length == 0 {
			cp.Alive = false
			break
		}
		length, err = write.Write(buf[:length])
		if err != nil || length == 0 {
			cp.Alive = false
			break
		}
	}
}

func (cp *ConnectionPipe) Pipe() {
	cp.Alive = true
	go cp.pipeConnection(cp.Left, cp.Right)
	cp.pipeConnection(cp.Right, cp.Left)
}

func (cp *ConnectionPipe) Kill() {
	cp.Alive = false
}