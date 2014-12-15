package sunrpc

import (
	"errors"
	"net"
	"reflect"
	"strconv"

	log "gopkg.in/sirupsen/logrus.v0"
)

var (
	ErrCannotPortmap = errors.New("cannot set port with portmapper")
)

type TCPServer struct {
	program    uint32
	version    uint32
	procedures map[uint32]interface{}
}

func NewServer(program uint32, version uint32) *TCPServer {
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
				// Ignore broken call
				continue
			}

			go server.handleCall(conn)
		}
	}()

	return nil
}

func (server *TCPServer) handleCall(conn net.Conn) {
	defer conn.Close()

	// Read message envelope
	call, err := ReadTCPCallMessage(conn)
	if err != nil {
		log.WithField("err", err).Error("Cannot read RPC Call message")

		return
	}

	if call.Program != server.program {
		log.WithFields(log.Fields{
			"expected": server.program,
			"was":      call.Program,
		}).Error("Mismatched program number")

		return
	}

	if call.Version != server.version {
		log.WithFields(log.Fields{
			"expected": server.version,
			"was":      call.Version,
		}).Error("Mismatched program version")

		return
	}

	// Determine procedure call
	receiverFunc, ok := server.procedures[call.Procedure]

	if !ok {
		log.WithField("procedure", call.Procedure).Error("Cannot find procedure")

		return
	}

	// Call bound function
	funcType := reflect.TypeOf(receiverFunc)
	funcArgType := funcType.In(0)
	funcArg := reflect.New(funcArgType).Interface()

	err = ReadArguments(conn, funcArg)
	if err != nil {
		log.Error(err)

		return
	}

	funcValue := reflect.ValueOf(receiverFunc)
	funcArgValue := reflect.ValueOf(funcArg)

	funcValue.Call([]reflect.Value{reflect.Indirect(funcArgValue)})
}
