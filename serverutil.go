package sunrpc

import (
	"bytes"
	"io"
	"reflect"

	"github.com/davecgh/go-xdr/xdr2"
)

// ReadProcedureCall reads an RPC "call" message from the given reader, ensuring the RPC message is
// of the "call" type and specifies version '2' of the RPC protocol.
func ReadProcedureCall(r io.Reader) (*ProcedureCall, error) {
	// Read RPC message header
	message := ProcedureCall{}

	if _, err := xdr.Unmarshal(r, &message); err != nil {
		return nil, ErrHeaderExpected
	}

	// Make sure this is a "Call" message
	if message.Header.Type != Call {
		return nil, ErrCallMessageExpected
	}

	// We can only read RPCv2 messages
	if message.Body.RPCVersion != 2 {
		return nil, ErrRPCVersion2Expected
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
		return 0, err
	}

	// "Accepted"
	if _, err := xdr.Marshal(&buf, ReplyBody{Type: Accepted}); err != nil {
		return 0, err
	}

	// "Success"
	if _, err := xdr.Marshal(&buf, AcceptedReply{Type: Success}); err != nil {
		return 0, err
	}

	// Return data
	if _, err := xdr.Marshal(&buf, ret); err != nil {
		return 0, err
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
		return nil, errUnknownFunction
	}

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

	funcValue.Call([]reflect.Value{funcArgValue, funcRetValue})

	// Return result computed by the actual function. This should be sent back to the remote caller.
	return funcRetValue.Interface(), nil
}
