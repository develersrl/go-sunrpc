package sunrpc

import (
	"net"
	"reflect"
	"strconv"

	"github.com/davecgh/go-xdr/xdr2"
	"gopkg.in/sirupsen/logrus.v0"
)

var (
	tcpLog = logrus.WithFields(logrus.Fields{
		"package": "sunrpc",
		"server":  "tcp",
	})
)

type TCPServer struct {
	program    uint32
	version    uint32
	procedures map[uint32]interface{}
}

func NewTCPServer(program uint32, version uint32) *TCPServer {
	return &TCPServer{
		program:    program,
		version:    version,
		procedures: map[uint32]interface{}{},
	}
}

func (server *TCPServer) Register(proc uint32, rcvr interface{}) {
	server.procedures[proc] = rcvr
}

func (server *TCPServer) Serve(addr string) error {
	// Start TCP Server
	listener, err := net.Listen("tcp", addr)
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

	err = PortmapperSet(server.program, server.version, Tcp, uint32(portAsInt))
	if err != nil {
		return err
	}

	// Handle incoming connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				tcpLog.WithField("err", err).Error("Unable to accept incoming connection. Ignoring")

				continue
			}

			tcpLog.WithField("from", conn.RemoteAddr().String()).Debug("Client connected")

			server.handleCall(conn)
		}
	}()

	return nil
}

func (server *TCPServer) handleCall(conn net.Conn) {
	for {
		// Read message envelope
		call, err := ReadTCPCallMessage(conn)
		if err != nil {
			tcpLog.WithField("err", err).Error("Cannot read RPC Call message")

			return
		}

		if call.Body.Program != server.program {
			tcpLog.WithFields(logrus.Fields{
				"expected": server.program,
				"was":      call.Body.Program,
			}).Error("Mismatched program number")

			return
		}

		if call.Body.Version != server.version {
			tcpLog.WithFields(logrus.Fields{
				"expected": server.version,
				"was":      call.Body.Version,
			}).Error("Mismatched program version")

			return
		}

		// Determine procedure call
		tcpLog.WithField("procedure", call.Body.Procedure).Debug("Calling procedure")

		receiverFunc, ok := server.procedures[call.Body.Procedure]

		if !ok {
			tcpLog.WithField("procedure", call.Body.Procedure).Error("Cannot find procedure")

			return
		}

		// Call bound function
		funcType := reflect.TypeOf(receiverFunc)
		funcArg := reflect.New(funcType.In(0)).Interface()

		if _, err := xdr.Unmarshal(conn, &funcArg); err != nil {
			tcpLog.Error(err)

			return
		}

		funcValue := reflect.ValueOf(receiverFunc)
		funcArgValue := reflect.Indirect(reflect.ValueOf(funcArg))
		funcRetValue := reflect.New(funcType.In(1).Elem())

		funcValue.Call([]reflect.Value{funcArgValue, funcRetValue})

		// Write reply
		// FIXME: We are assuming it is always "successful".
		if err := WriteTCPReplyMessage(conn, call.Header.Xid, funcRetValue.Interface()); err != nil {
			tcpLog.Error(err)

			return
		}
	}
}
