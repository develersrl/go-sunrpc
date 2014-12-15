package sunrpc

import (
	"errors"
	"time"
)

var (
	ErrIncompleteMessage = errors.New("rpc call: unable to read the whole message")
)

//
// RPC Message Header
//

type MessageType int32

const (
	Call  MessageType = 0
	Reply MessageType = 1
)

type Message struct {
	Xid  uint32
	Type MessageType
}

func NewMessage(t MessageType) *Message {
	return &Message{
		Xid:  uint32(time.Now().Unix()),
		Type: t,
	}
}

//
// Call
//

type CallMessage struct {
	RPCVersion uint32
	Program    uint32
	Version    uint32
	Procedure  uint32
	Cred       [2]uint32 // Dummy
	Verf       [2]uint32 // Dummy
}

//
// Reply
//

type ReplyType int32

const (
	Accepted ReplyType = 0
	Denied   ReplyType = 1
)

type ReplyMessage struct {
	Type ReplyType
}

type AcceptType int32

const (
	Success AcceptType = 0
)

type AcceptedReply struct {
	Verf [2]uint32 // Dummy
	Type AcceptType
}
