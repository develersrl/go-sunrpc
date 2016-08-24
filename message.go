package sunrpc

import "time"

//
// Authorization Data
//

// AuthFlavor is an enumeration of all supported authentication flavors.
type AuthFlavor int32

// All possible authentication flavors.
const (
	AuthFlavorNone AuthFlavor = iota
	AuthFlavorUnix
	AuthFlavorDes
)

type OpaqueAuth struct {
	Flavor AuthFlavor
	Body   []byte // Must be between 0 and 400 bytes
}

type AuthNone struct{}
type AuthUnix struct {
	Stamp       uint32
	MachineName string
	Uid, Gid    uint32
	Gids        []uint32
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

	InvalidReplyType ReplyType = -1
)

type ReplyBody struct {
	Type ReplyType
}

// AcceptType is used to tell the client how the server accepted an RPC call.
type AcceptType int32

// Enumeration of all possible RPC "Accept" messages.
const (
	Success      AcceptType = 0
	ProgUnavail             = 1
	ProgMismatch            = 2
	ProcUnavail             = 3
	GarbageArgs             = 4
	SystemErr               = 5
)

// AcceptedReply is the
type AcceptedReply struct {
	Verf OpaqueAuth
	Type AcceptType
}

type RejectStat uint32

const (
	RpcMismatch RejectStat = 0
	AuthError              = 1

	NoReject RejectStat = 0xFFFFFFFF
)

type AuthStat uint32

const (
	AuthBadCred AuthStat = iota
	AuthRejectedCred
	AuthBadVerf
	AUthRejectedVerf
	AuthTooWeak
)

type RejectedReply struct {
	Stat RejectStat
}

type ProgMismatchReply struct {
	Low  uint
	High uint
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

// ProcedureReply is the reply to a procedure call
type ProcedureReply struct {
	Header   Message
	Type     ReplyType `xdr:"union"`
	Accepted struct {
		Verf         OpaqueAuth
		Stat         AcceptType `xdr:"union"`
		MismatchInfo struct {
			Low, High uint32
		} `xdr:"unioncase=2"` // ProgMismatch
		// results follow here
	} `xdr:"unioncase=0"`
	Rejected struct {
		Stat         RejectStat `xdr:"union"`
		MismatchInfo struct {
			Low, High uint32
		} `xdr:"unioncase=0"` // RpcMismatch
		AuthStat AuthStat `xdr:"unioncase=0"` // AuthError
	} `xdr:"unioncase=1"`
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
