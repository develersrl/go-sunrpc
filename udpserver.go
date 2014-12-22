package sunrpc

import (
	"bytes"
	"net"
	"reflect"
	"strconv"

	"github.com/davecgh/go-xdr/xdr2"
	"gopkg.in/sirupsen/logrus.v0"
)

const (
	MaxUdpSize = 65507
)

var (
	udpLog = logrus.WithFields(logrus.Fields{
		"package": "sunrpc",
		"server":  "udp",
	})
)

type UDPServer struct {
	program    uint32
	version    uint32
	procedures map[uint32]interface{}
}

func NewUDPServer(program uint32, version uint32) *UDPServer {
	return &UDPServer{
		program:    program,
		version:    version,
		procedures: map[uint32]interface{}{},
	}
}

func (server *UDPServer) Register(proc uint32, rcvr interface{}) {
	server.procedures[proc] = rcvr
}

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
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		return err
	}

	if err := conn.SetReadBuffer(MaxUdpSize); err != nil {
		return err
	}

	udpLog.WithFields(logrus.Fields{
		"host": host,
		"port": port,
	}).Debug("Server started")

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

	// Determine procedure call
	udpLog.WithField("procedure", call.Body.Procedure).Debug("Procedure call")

	receiverFunc, ok := server.procedures[call.Body.Procedure]

	if !ok {
		udpLog.WithField("procedure", call.Body.Procedure).Error("Cannot find procedure")

		return
	}

	// Call bound function
	funcType := reflect.TypeOf(receiverFunc)
	funcArg := reflect.New(funcType.In(0)).Interface()

	if _, err := xdr.Unmarshal(buf, &funcArg); err != nil {
		udpLog.Error(err)

		return
	}

	funcValue := reflect.ValueOf(receiverFunc)
	funcArgValue := reflect.Indirect(reflect.ValueOf(funcArg))
	funcRetValue := reflect.New(funcType.In(1).Elem())

	funcValue.Call([]reflect.Value{funcArgValue, funcRetValue})

	// Write reply to buffer (needed because we will need to use conn.WriteToUDP instead of a simple
	// Write() later on).
	//
	// FIXME: We are assuming it is always "successful".
	var replyBuf bytes.Buffer
	if _, err := WriteReplyMessage(&replyBuf, call.Header.Xid, funcRetValue.Interface()); err != nil {
		udpLog.WithField("err", err).Error("Cannot write reply to buffer")

		return
	}

	// Send reply to caller.
	if _, err := conn.WriteToUDP(replyBuf.Bytes(), callerAddr); err != nil {
		udpLog.WithFields(logrus.Fields{
			"callerAddr": callerAddr.String(),
			"err":        err,
		}).Error("Cannot send reply over UDP")

		return
	}
}
