package sunrpc

import (
	"bytes"
	"net"
	"strconv"

	"github.com/lvillani/logrus"
)

// MaxUdpSize is the maximum size of an RPC message we accept over UDP.
const MaxUdpSize = 65507

var udpLog = logrus.WithFields(logrus.Fields{
	"package": "sunrpc",
	"server":  "udp",
})

// UDPServer is an RPC server over UDP.
type UDPServer struct {
	program    uint32
	version    uint32
	procedures map[uint32]interface{}
}

// NewUDPServer creates a new UDPServer for the given RPC program identifier and program version.
func NewUDPServer(program uint32, version uint32) *UDPServer {
	return &UDPServer{
		program:    program,
		version:    version,
		procedures: map[uint32]interface{}{},
	}
}

// Register binds a new RPC procedure ID to a function.
func (server *UDPServer) Register(proc uint32, rcvr interface{}) {
	server.procedures[proc] = rcvr
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

func (server *UDPServer) handleCall(conn *net.UDPConn) {
	// Read and buffer UDP datagram
	b := make([]byte, MaxUdpSize)

	packetSize, callerAddr, err := conn.ReadFromUDP(b)
	if err != nil {
		udpLog.WithField("err", err).Error("Cannot read UDP datagram")

		return
	}

	// Read message envelope
	buf := bytes.NewBuffer(b[0:packetSize])
	call, err := ReadProcedureCall(buf)
	if err != nil {
		udpLog.WithField("err", err).Error("Cannot parse RPC call")

		return
	}

	if call.Body.Program != server.program {
		udpLog.WithFields(logrus.Fields{
			"expected": server.program,
			"was":      call.Body.Program,
		}).Error("Mismatched program number")

		return
	}

	if call.Body.Version != server.version {
		udpLog.WithFields(logrus.Fields{
			"expected": server.version,
			"was":      call.Body.Version,
		}).Error("Mismatched program version")

		return
	}

	// Function Call
	udpLog.WithField("procedure", call.Body.Procedure).Debug("Procedure call")

	ret, err := callFunc(buf, server.procedures, call.Body.Procedure)
	if err != nil {
		udpLog.WithField("err", err).Error("Unable to perform procedure call")

		return
	}

	// Send response to client.
	//
	// We can't use a simple Write() here, thus we have to buffer our payload and then send
	// everything with WriteToUDP().
	//
	// FIXME: We are assuming it is always "successful".
	var replyBuf bytes.Buffer
	if _, err := WriteReplyMessage(&replyBuf, call.Header.Xid, ret); err != nil {
		udpLog.WithField("err", err).Error("Cannot write reply to buffer")

		return
	}

	if _, err := conn.WriteToUDP(replyBuf.Bytes(), callerAddr); err != nil {
		udpLog.WithFields(logrus.Fields{
			"callerAddr": callerAddr.String(),
			"err":        err,
		}).Error("Cannot send reply over UDP")

		return
	}
}
