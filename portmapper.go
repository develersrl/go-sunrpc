package sunrpc

import (
	"net"
	"runtime"
	"sync"
)

// RPC program ID, version number and other stuff to speak the Portmapper protocol.
const (
	PortmapperProgram = 100000
	PortmapperVersion = 2
	PortmapperPortSet = 1
)

// PortmapperProtocol is an enumeration denoting whether the RPC server we are registering runs over
// TCP or UDP.
type PortmapperProtocol uint32

// All connection types supported by an RPC server that can be registered to a Portmapper discovery
// service.
const (
	Tcp PortmapperProtocol = 6
	Udp PortmapperProtocol = 17
)

type mapping struct {
	Program  uint32
	Version  uint32
	Protocol PortmapperProtocol
	Port     uint32
}

var pmapInit sync.Once

// Initialize connection to portmapper.
//
// Calling this function is not strictly required, but it might be advised
// to speed up subsequent calls to PortmapperSet.
//
// NOTE: on Darwin, this may take up to 10 seconds, so you are advised to
// run this function as early as possible (in a goroutine).
func PortmapperInit() {
	pmapInit.Do(func() {
		// On Darwin, rpcbind is socket-activated using a UNIX local socket.
		// To trigger its start, we need to open the socket and read from it,
		// until a byte arrives (to signal that activation is complete).
		// Notice that (for unknown reasons), launchd can takes several seconds
		// to launch it; for instance, on OSX El Capitan, activation time is
		// about ~10 seconds.
		if runtime.GOOS == "darwin" {
			act, err := net.Dial("unix", "/var/run/portmap.socket")
			if err != nil {
				return
			}
			var data [1]byte
			act.Read(data[:])
			act.Close()
		}
	})
}

// PortmapperSet associates an RPC server with a Portmapper server running on the current host
// (i.e.: 127.0.0.1).
func PortmapperSet(program uint32, version uint32, protocol PortmapperProtocol, port uint32) error {
	PortmapperInit()

	conn, err := net.Dial("tcp", "127.0.0.1:111")
	if err != nil {
		conn, err = net.Dial("udp", "127.0.0.1:111")
		if err != nil {
			return err
		}
	}
	defer conn.Close()

	return WriteCall(conn, PortmapperProgram, PortmapperVersion, PortmapperPortSet, mapping{
		Program:  program,
		Version:  version,
		Protocol: protocol,
		Port:     port,
	})
}
