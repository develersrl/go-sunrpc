package sunrpc

import "net"

const (
	PortmapperProgram = 100000
	PortmapperVersion = 2
	PortmapperPortSet = 1
)

type PortmapperProtocol uint32

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
