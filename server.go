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
	}).Info("RPC ", s.procnames[call.Body.Procedure])
	acceptType := Success
	ret, err := s.callFunc(r, receiverFunc)
	if err != nil {
		s.log.WithField("err", err).Error("Unable to perform procedure call")
		acceptType = SystemErr
	}

	err = s.WriteReplyMessage(&reply, call.Header.Xid, acceptType, ret)
	return reply, err
}
