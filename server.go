package sunrpc

import (
	"bytes"
	"strconv"

	"gopkg.in/Sirupsen/logrus.v0"
)

type server struct {
	program    uint32
	version    uint32
	procedures map[uint32]interface{}
	procnames  map[uint32]string
	log        *logrus.Entry
	authFun    func(proc uint32, cred interface{}) bool
}

func newServer(program uint32, version uint32, f logrus.Fields) server {
	return server{
		program:    program,
		version:    version,
		procedures: make(map[uint32]interface{}),
		procnames:  make(map[uint32]string),
		log:        logrus.WithField("package", "sunrpc").WithFields(f),
	}
}

// Register binds a new RPC procedure ID to a function.
func (server *server) Register(proc uint32, rcvr interface{}) {
	server.procedures[proc] = rcvr
}

func (server *server) RegisterWithName(proc uint32, rcvr interface{}, name string) {
	server.procedures[proc] = rcvr
	server.procnames[proc] = name
}

func (server *server) registerToPortmapper(prot PortmapperProtocol, port int) error {
	// Check if the portmapper server is available, to return a proper high-level error
	// rather than a generic socket error.
	if !PortmapperAvailable() {
		return ErrorPortmapperNotFound
	}

	// First check if there's a mapping already. We do this because Linux rpcbind server (but not OSX)
	// is smart enough to use this call to also verify whether a registered service
	// is still alive (listening on that port), and if it doesn't, it returns zero.
	//
	// Assuming an application where the user is free to change listening port in configuration,
	// this would allow the user to run the application more than one time with different ports,
	// without getting errors, as the call to PortmapperGet() would effectively deregister the
	// previous registration automatically.
	getport, err := PortmapperGet(server.program, server.version, prot)
	switch {
	case err != nil:
		return err
	case getport == 0:
		// no service found, we need to register again
		return PortmapperSet(server.program, server.version, prot, uint32(port))
	case getport != uint32(port):
		// found a service with a different port, returns error
		return ErrorPortmapperServiceExists
	default:
		// port is what we expect, we are already registered, nothing to do
		return nil
	}
}

func (s *server) handleRecord(record []byte) (bytes.Buffer, error) {

	var reply bytes.Buffer
	r := bytes.NewReader(record)

	call, err := ReadProcedureCall(r)
	if err != nil {
		s.log.WithField("err", err).Error("Cannot read RPC Call message")
		return reply, err
	}

	if call.Body.Program != s.program {
		s.log.WithFields(logrus.Fields{
			"expected": s.program,
			"was":      call.Body.Program,
		}).Error("Mismatched program number")

		err := s.WriteReplyMessage(&reply, call.Header.Xid, ProgUnavail, nil)
		return reply, err
	}

	if call.Body.Version != s.version {
		s.log.WithFields(logrus.Fields{
			"expected": s.version,
			"was":      call.Body.Version,
		}).Error("Mismatched program version")

		ret := ProgMismatchReply{
			Low:  uint(s.version),
			High: uint(s.version),
		}
		err := s.WriteReplyMessage(&reply, call.Header.Xid, ProgMismatch, &ret)
		return reply, err
	}

	// Handle authentication (if the user requested so)
	if s.authFun != nil {
		auth, err := call.Body.Cred.Decode()
		if err != nil {
			s.log.WithField("err", err).Error("cannot decode authentication")
			err := s.WriteReplyMessageRejectedAuth(&reply, call.Header.Xid, AuthBadCred)
			return reply, err
		}

		if !s.authFun(call.Body.Procedure, auth) {
			s.log.WithFields(logrus.Fields{
				"proc": strconv.Itoa(int(call.Body.Procedure)),
				"prog": strconv.Itoa(int(call.Body.Program)),
			}).Info("authentication rejected by user")
			err := s.WriteReplyMessageRejectedAuth(&reply, call.Header.Xid, AuthBadCred)
			return reply, err
		}
	}

	// Resolve function type from function table
	receiverFunc, found := s.procedures[call.Body.Procedure]
	if !found {
		s.log.WithFields(logrus.Fields{
			"proc": strconv.Itoa(int(call.Body.Procedure)),
			"prog": strconv.Itoa(int(call.Body.Program)),
		}).Error("Unsupported procedure call")

		err := s.WriteReplyMessage(&reply, call.Header.Xid, ProcUnavail, nil)
		return reply, err
	}

	s.log.WithFields(logrus.Fields{
		"proc": strconv.Itoa(int(call.Body.Procedure)),
		"name": s.procnames[call.Body.Procedure],
	}).Debug("RPC ", s.procnames[call.Body.Procedure])
	acceptType := Success
	ret, err := s.callFunc(r, receiverFunc)
	if err != nil {
		s.log.WithField("err", err).Error("Unable to perform procedure call")
		acceptType = SystemErr
	}

	err = s.WriteReplyMessage(&reply, call.Header.Xid, acceptType, ret)
	return reply, err
}
