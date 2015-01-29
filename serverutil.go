package sunrpc

import (
	"bytes"
	"io"
	"reflect"

	"github.com/davecgh/go-xdr/xdr2"
	"github.com/dropbox/godropbox/errors"
)

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
func WriteReplyMessage(w io.Writer, xid uint32, ret interface{}) (int, error) {
	var buf bytes.Buffer

	// Header
	header := Message{
		Xid:  xid,
		Type: Reply,
	}

	if _, err := xdr.Marshal(&buf, header); err != nil {
		return 0, errors.Wrap(err, "Could not write the RPC message header")
	}

	// "Accepted"
	if _, err := xdr.Marshal(&buf, ReplyBody{Type: Accepted}); err != nil {
		return 0, errors.Wrap(err, "Could not write the reply body")
	}

	// "Success"
	if _, err := xdr.Marshal(&buf, AcceptedReply{Type: Success}); err != nil {
		return 0, errors.Wrap(err, "Could not write the reply body")
	}

	// Return data
	if _, err := xdr.Marshal(&buf, ret); err != nil {
		return 0, errors.Wrap(err, "Could not write the return value")
	}

	return w.Write(buf.Bytes())
}

// callFunc Resolves and calls a real Go function given a procedure ID. The method must look
// schematically like this (but no checks are made at runtime):
//
//     func (t *T) MethodName(argType T1, replyType *T2) error
func callFunc(r io.Reader, table map[uint32]interface{}, proc uint32) (interface{}, error) {
	// Resolve function type from function table
	receiverFunc, found := table[proc]
	if !found {
		return nil, errors.Newf("Tried to call unknown procedure with id: %v", proc)
	}

	// Resolve function's type
	funcType := reflect.TypeOf(receiverFunc)

	// Deserialize arguments read from procedure call body
	funcArg := reflect.New(funcType.In(0)).Interface()

	if _, err := xdr.Unmarshal(r, &funcArg); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal the arguments to pass to the procedure")
	}

	// Call function
	funcValue := reflect.ValueOf(receiverFunc)
	funcArgValue := reflect.Indirect(reflect.ValueOf(funcArg))
	funcRetValue := reflect.New(funcType.In(1).Elem())

	funcValue.Call([]reflect.Value{funcArgValue, funcRetValue})

	// Return result computed by the actual function. This should be sent back to the remote caller.
	return funcRetValue.Interface(), nil
}
