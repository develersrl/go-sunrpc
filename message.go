package sunrpc

import "time"

//
// Authorization Data
//

// AuthFlavor is an enumeration of all supported authentication flavors.
type AuthFlavor int32

// All possible authentication flavors.
const (
	AuthFlavorNone AuthFlavor = 0
)

type OpaqueAuth struct {
	Flavor AuthFlavor
	Body   []byte // Must be between 0 and 400 bytes
}

//
// RPC Message
//

// MessageType is an enumeration of all possible RPC message types.
type MessageType int32

// All possible RPC message types.
const (
	Call  MessageType = 0
	Reply MessageType = 1
)

// Message is an RPC message header.
type Message struct {
	Xid  uint32
	Type MessageType
}

//
// Call
//

// CallBody is the body of an RPC "Call" message.
type CallBody struct {
	RPCVersion uint32
	Program    uint32
	Version    uint32
	Procedure  uint32
	Cred       OpaqueAuth
	Verf       OpaqueAuth
}

//
// Reply
//

// ReplyType is the kind of RPC "reply" message.
type ReplyType int32

// Enumeration of all possible RPC replies.
const (
	Accepted ReplyType = 0
	Denied   ReplyType = 1
)

type ReplyBody struct {
	Type ReplyType
}

// AcceptType is used to tell the client how the server accepted an RPC call.
type AcceptType int32

// Enumeration of all possible RPC "Accept" messages.
const (
	Success   AcceptType = 0
	SystemErr AcceptType = 5
)

// AcceptedReply is the
type AcceptedReply struct {
	Verf OpaqueAuth
	Type AcceptType
}

//
// Convenience: Procedure Call
//

// ProcedureCall combines the RPC Message header with RPC Call body (except for function arguments)
// for convenience during (de)serialization.
type ProcedureCall struct {
	Header Message
	Body   CallBody
}

// NewProcedureCall creates a new RPC call packet with a transaction ID derived from the current
// UNIX time stamp.
func NewProcedureCall(program uint32, version uint32, procedure uint32) *ProcedureCall {
	return &ProcedureCall{
		Header: Message{
			Xid:  uint32(time.Now().Unix()),
			Type: Call,
		},
		Body: CallBody{
			RPCVersion: 2,
			Program:    program,
			Version:    version,
			Procedure:  procedure,
		},
	}
}
