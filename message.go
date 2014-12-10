package sunrpc

type MessageType int32

const (
	Call  MessageType = 0
	Reply MessageType = 1
)

type Message struct {
	Xid  uint32
	Type MessageType
}

type CallMessage struct {
	RPCVersion uint32
	Program    uint32
	Version    uint32
	Procedure  uint32
	Cred       [2]uint32 // Dummy
	Verf       [2]uint32 // Dummy
}
