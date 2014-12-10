package sunrpc

import (
	"bytes"
	"net"

	"github.com/davecgh/go-xdr/xdr2"
)

const (
	PortmapperPort    = 111
	PortmapperProgram = 100000
	PortmapperVersion = 2
	PortmapperPortSet = 1
)

type PortmapperProtocol uint32

const (
	Tcp PortmapperProtocol = 6
	Udp PortmapperProtocol = 17
)

// Field order is important
type Mapping struct {
	Call     RpcCall
	Program  uint32
	Version  uint32
	Protocol PortmapperProtocol
	Port     uint32
}

func NewMapping(program uint32, version uint32, protocol PortmapperProtocol, port uint32) *Mapping {
	return &Mapping{
		Call:     *NewRpcCall(PortmapperProgram, PortmapperVersion, PortmapperPortSet),
		Program:  program,
		Version:  version,
		Protocol: protocol,
		Port:     port,
	}
}

func PortmapperSet(program uint32, version uint32, protocol PortmapperProtocol, port uint32) error {
	message := NewMapping(program, version, protocol, port)

	conn, err := net.Dial("udp", "localhost:111")
	if err != nil {
		return err
	}
	defer conn.Close()

	var buf bytes.Buffer

	_, err = xdr.Marshal(&buf, &message)
	if err != nil {
		return err
	}

	err = conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}
