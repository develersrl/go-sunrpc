package sunrpc

import "net"

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

// PortmapperSet associates an RPC server with a Portmapper server running on the current host
// (i.e.: 127.0.0.1).
func PortmapperSet(program uint32, version uint32, protocol PortmapperProtocol, port uint32) error {
	conn, err := net.Dial("udp", "127.0.0.1:111")
	if err != nil {
		return err
	}
	defer conn.Close()

	return WriteCall(conn, PortmapperProgram, PortmapperVersion, PortmapperPortSet, mapping{
		Program:  program,
		Version:  version,
		Protocol: protocol,
		Port:     port,
	})
}
