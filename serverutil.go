package sunrpc

import (
	"bytes"
	"errors"
	"io"
	"reflect"

	"github.com/rasky/go-xdr/xdr2"
)

type Server interface {
	Register(proc uint32, rcvr interface{})
	RegisterWithName(proc uint32, rcvr interface{}, name string)
	SetAuth(authFun func(proc uint32, cred interface{}) bool)
	Serve(string) error
}

// ReadProcedureCall reads an RPC "call" message from the given reader, ensuring the RPC message is
// of the "call" type and specifies version '2' of the RPC protocol.
func ReadProcedureCall(r io.Reader) (*ProcedureCall, error) {
	// Read RPC message header
	message := ProcedureCall{}

	if _, err := xdr.Unmarshal(r, &message); err != nil {
		return nil, errors.New("")
	}

	// Make sure this is a "Call" message
	if message.Header.Type != Call {
		return nil, errors.New("Expected a call message")
	}

	// We can only read RPCv2 messages
	if message.Body.RPCVersion != 2 {
		return nil, errors.New("Expected an RPC version 2 message")
	}

	return &message, nil
}

// WriteReplyMessage writes an "Accepted" RPC reply of type "Success", indicating that the procedure
// call was successful. The given return data is written right after the RPC response header.
func (s *server) WriteReplyMessage(w io.Writer, xid uint32, acceptType AcceptType, ret interface{}) error {
	var buf bytes.Buffer

	// Header
	header := Message{
		Xid:  xid,
		Type: Reply,
	}

	if _, err := xdr.Marshal(&buf, header); err != nil {
		return err
	}

	// "Accepted"
	if _, err := xdr.Marshal(&buf, ReplyBody{Type: Accepted}); err != nil {
		return err
	}

	// "Success"
	if _, err := xdr.Marshal(&buf, AcceptedReply{Type: acceptType}); err != nil {
		return err
	}

	// Return data
	if ret != nil {
		if _, err := xdr.Marshal(&buf, ret); err != nil {
			return err
		}
	}

	_, err := w.Write(buf.Bytes())
	return err
}

func (s *server) WriteReplyMessageRejectedAuth(w io.Writer, xid uint32, auth AuthStat) error {
	var buf bytes.Buffer

	// Header
	header := Message{
		Xid:  xid,
		Type: Reply,
	}

	if _, err := xdr.Marshal(&buf, header); err != nil {
		return err
	}

	// "Denied"
	if _, err := xdr.Marshal(&buf, ReplyBody{Type: Denied}); err != nil {
		return err
	}

	// "AuthError"
	if _, err := xdr.Marshal(&buf, RejectedReply{Stat: AuthError}); err != nil {
		return err
	}

	if _, err := xdr.Marshal(&buf, &auth); err != nil {
		return err
	}

	return nil
}

// callFunc Resolves and calls a real Go function given a procedure ID. The method must look
// schematically like this (but no conformance checks are performed at runtime):
//
//     func (t *T) MethodName(argType T1, replyType *T2) error
func (s *server) callFunc(r io.Reader, receiverFunc interface{}) (interface{}, error) {

	// Resolve function's type
	funcType := reflect.TypeOf(receiverFunc)

	// Deserialize arguments read from procedure call body
	funcArg := reflect.New(funcType.In(0)).Interface()

	if _, err := xdr.Unmarshal(r, &funcArg); err != nil {
		return nil, err
	}

	// Call function
	funcValue := reflect.ValueOf(receiverFunc)
	funcArgValue := reflect.Indirect(reflect.ValueOf(funcArg))
	funcRetValue := reflect.New(funcType.In(1).Elem())

	s.log.Infof("-> %+v", funcArgValue)
	funcRetError := funcValue.Call([]reflect.Value{funcArgValue, funcRetValue})[0]
	s.log.Infof("<- %+v", funcRetValue)

	if !funcRetError.IsNil() {
		return nil, funcRetError.Interface().(error)
	}

	// Return result computed by the actual function. This is what should be sent back to the remote
	// caller.
	return funcRetValue.Interface(), nil
}

func (o *OpaqueAuth) Decode() (interface{}, error) {
	switch o.Flavor {
	case AuthFlavorNone:
		return AuthNone{}, nil
	case AuthFlavorUnix:
		auth := AuthUnix{}
		_, err := xdr.Unmarshal(bytes.NewReader(o.Body), &auth)
		if err != nil {
			return nil, err
		}
		return auth, nil
	case AuthFlavorDes:
		return nil, errors.New("unsupported DES authentication")
	default:
		return nil, errors.New("unsupported authentication flavor")
	}
}

func (s *server) SetAuth(authFun func(uint32, interface{}) bool) {
	s.authFun = authFun
}
