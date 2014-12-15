package sunrpc

import (
	"errors"
	"io"
	"time"

	"github.com/davecgh/go-xdr/xdr2"
)

type MessageType int32

const (
	Call  MessageType = 0
	Reply MessageType = 1
)

var (
	ErrIncompleteMessage = errors.New("rpc call: unable to read the whole message")
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

type CallMessage struct {
	RPCVersion uint32
	Program    uint32
	Version    uint32
	Procedure  uint32
	Cred       [2]uint32 // Dummy
	Verf       [2]uint32 // Dummy
}

func ReadArguments(r io.Reader, args interface{}) error {
	// Read RPC call arguments
	_, err := xdr.Unmarshal(r, &args)
	if err != nil {
		return err
	}

	return nil
}
