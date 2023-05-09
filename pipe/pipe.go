package pipe

import "net"

// struct for an active or inactive connection pipe
type ConnectionPipe struct {
	Alive bool // represents whether or not the connection pipe is actively piping data
	Left net.Conn // left-hand-side of this connection pipe
	Right net.Conn // right-hand-side of this connection pipe
}

// main constructor function for a connection pipe, accepting
// left and right as parameters and default 'Alive' = false
func NewConnectionPipe(left net.Conn, right net.Conn) ConnectionPipe {
	return ConnectionPipe{
		Alive: false,
		Left:  left,
		Right: right,
	}
}

// function existing as a method on the ConnectionPipe to pipe connections,
// it reads from one connection and directly pipes it to the other
// this alone is NOT bidirectional
func (cp *ConnectionPipe) pipeConnection(read net.Conn, write net.Conn) {
	// creates our buffer of 2048 bytes (2KB)
	buf := make([]byte, 2048)

	// while this connection pipe is alive
	for cp.Alive {
		// reads from the 'read' connection and dumps that data into buf
		length, err := read.Read(buf)
		
		// if an error was thrown or the length of the data read is 0
		if err != nil || length == 0 {
			// kill the connection pipe and break from the loop
			cp.Alive = false
			break
		}

		// writes to the 'write' connection the contents of the buf
		// up to the length returned from the read function
		length, err = write.Write(buf[:length])

		// if an error was returned or the length successfully written is 0:
		// terminate the connection pipe and break out of the loop
		if err != nil || length == 0 {
			cp.Alive = false
			break
		}
	}
}

// function to begin piping on the connection pipe bidirectionally
func (cp *ConnectionPipe) Pipe() {
	// first, set the Alive field equal to true to represent
	// the connection pipe as now being active
	cp.Alive = true

	// in a goroutine, pipe connection from one way to the other
	go cp.pipeConnection(cp.Left, cp.Right)

	// blocking this thread context, pipe the connection in a reverse direction to the initial way
	cp.pipeConnection(cp.Right, cp.Left)
}

// function to kill a connection pipe early
func (cp *ConnectionPipe) Kill() {
	// sets the Alive field = false
	cp.Alive = false
}
