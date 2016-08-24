package sunrpc

import (
	"io"
	"net"
	"strconv"

	"gopkg.in/Sirupsen/logrus.v0"
)

// TCPServer is an RPC server over TCP.
type TCPServer struct {
	server
}

// NewTCPServer creates a new RPC server for the given program id and program version.
func NewTCPServer(program uint32, version uint32) Server {
	return &TCPServer{
		server: newServer(program, version, logrus.Fields{"proto": "tcp"}),
	}
}

// Serve starts the RPC server.
func (s *TCPServer) Serve(addr string) error {
	// Start TCP Server
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		return err
	}

	// Bind to RPCBIND server
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}

	portAsInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	if err := s.registerToPortmapper(Tcp, portAsInt); err != nil {
		return err
	}

	// Handle incoming connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				s.server.log.WithField("err", err).Error("Unable to accept incoming connection. Ignoring")

				continue
			}

			s.server.log.WithField("remote", conn.RemoteAddr().String()).Debug("Client connected.")

			go s.handleCall(conn)
		}
	}()

	return nil
}

//
// Private
//

func (s *TCPServer) handleCall(conn net.Conn) {
	defer func() {
		s.server.log.WithField("remote", conn.RemoteAddr().String()).Debug("Closing connection.")

		conn.Close()
	}()

	for {
		// Make sure to read a whole record at a time.
		record, err := ReadRecord(conn)
		if err != nil {
			if err == io.EOF {
				return
			}
			s.server.log.WithField("err", err).Error("Unable to read a record")
			return
		}

		reply, err := s.server.handleRecord(record.Bytes())
		if err != nil {
			s.server.log.WithField("err", err).Error("handling record")
		}

		// Send response
		if err := WriteTCPReplyMessage(conn, reply.Bytes()); err != nil {
			s.server.log.Error(err)
			return
		}
	}
}
