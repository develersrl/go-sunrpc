package sunrpc

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"sync"
)

// RPC program ID, version number and other stuff to speak the Portmapper protocol.
const (
	PortmapperProgram   = 100000
	PortmapperVersion   = 2
	PortmapperPortSet   = 1
	PortmapperPortUnset = 2
	PortmapperPortGet   = 3
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

type pmapMapping struct {
	Program  uint32
	Version  uint32
	Protocol PortmapperProtocol
	Port     uint32
}

var pmapInit sync.Once
var pmapClient *Client

var (
	ErrorPortmapperNotFound = errors.New("rpcbind server not found on localhost:111")

	// ErrorPortmapperServiceExists is returned by a call to PortmapperSet if there is already a
	// service registered for the specified triplet (program, version, protocol)
	ErrorPortmapperServiceExists = errors.New("RPC service is already registered")

	// ErrorPortmapperServiceDoesntExist is returned by a call to PortmapperUnset if there was no
	// service registered for the specified triplet (program, version, protocol)
	ErrorPortmapperServiceDoesntExist = errors.New("RPC service doesn't exist")
)

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

		pmapClient = NewClient("127.0.0.1:111", PortmapperProgram, PortmapperVersion, nil)
	})
}

// PortmapperAvailable returns true if we can correctly communicate with the portmapper server
func PortmapperAvailable() bool {
	PortmapperInit()

	if err := pmapClient.Call(0, nil, nil); err != nil {
		return false
	}

	return true
}

// PortmapperSet associates an RPC server with a Portmapper server running on the current host
// (i.e.: 127.0.0.1).
func PortmapperSet(program uint32, version uint32, protocol PortmapperProtocol, port uint32) error {
	PortmapperInit()

	mapping := pmapMapping{
		Program:  program,
		Version:  version,
		Protocol: protocol,
		Port:     port,
	}

	var ok bool
	if err := pmapClient.Call(PortmapperPortSet, &mapping, &ok); err != nil {
		return fmt.Errorf("cannot register to rpcbind server: %v", err)
	}

	if !ok {
		return ErrorPortmapperServiceExists
	}

	return nil
}

func PortmapperUnset(program uint32, version uint32) error {
	PortmapperInit()

	mapping := pmapMapping{
		Program: program,
		Version: version,
	}

	var ok uint32
	if err := pmapClient.Call(PortmapperPortUnset, &mapping, &ok); err != nil {
		return fmt.Errorf("cannot deregister from rpcbind server: %v", err)
	}

	if ok != 1 {
		return ErrorPortmapperServiceDoesntExist
	}

	return nil
}

func PortmapperGet(program uint32, version uint32, protocol PortmapperProtocol) (uint32, error) {
	PortmapperInit()

	mapping := pmapMapping{
		Program:  program,
		Version:  version,
		Protocol: protocol,
	}

	var port uint32
	if err := pmapClient.Call(PortmapperPortGet, &mapping, &port); err != nil {
		return 0, fmt.Errorf("cannot query rpcbind server: %v", err)
	}

	return port, nil
}
