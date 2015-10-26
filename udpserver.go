package sunrpc

import (
	"net"
	"strconv"

	"gopkg.in/Sirupsen/logrus.v0"
)

// MaxUdpSize is the maximum size of an RPC message we accept over UDP.
const MaxUdpSize = 65507

// UDPServer is an RPC server over UDP.
type UDPServer struct {
	server
}

// NewUDPServer creates a new UDPServer for the given RPC program identifier and program version.
func NewUDPServer(program uint32, version uint32) Server {
	return &UDPServer{
		server: newServer(program, version, logrus.Fields{"proto": "udp"}),
	}
}

// Serve starts the RPC server.
func (server *UDPServer) Serve(addr string) error {
	// Parse and deconstruct host and port
	host, portString, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portString)
	if err != nil {
		return err
	}

	// Start UDP Server
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		return err
	}

	if err := conn.SetReadBuffer(MaxUdpSize); err != nil {
		return err
	}

	// Bind to RPCBIND server
	if err := PortmapperSet(server.program, server.version, Udp, uint32(port)); err != nil {
		return err
	}

	go func() {
		for {
			server.handleCall(conn)
		}
	}()

	return nil
}

//
// Private
//

func (s *UDPServer) handleCall(conn *net.UDPConn) {
	// Read and buffer UDP datagram
	b := make([]byte, MaxUdpSize)

	packetSize, callerAddr, err := conn.ReadFromUDP(b)
	if err != nil {
		s.server.log.WithField("err", err).Error("Cannot read UDP datagram")

		return
	}

	reply, err := s.server.handleRecord(b[0:packetSize])
	if err != nil {
		s.server.log.WithField("err", err).Error("handling record")
	}

	if _, err := conn.WriteToUDP(reply.Bytes(), callerAddr); err != nil {
		s.server.log.WithFields(logrus.Fields{
			"callerAddr": callerAddr.String(),
			"err":        err,
		}).Error("Cannot send reply over UDP")

		return
	}
}
