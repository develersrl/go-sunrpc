package sunrpc

type RpcMessageType int32

const (
	Call  RpcMessageType = 0
	Reply RpcMessageType = 1
)

type RpcCall struct {
	Xid         uint32
	MessageType RpcMessageType
	RpcVersion  uint32
	Program     uint32
	Version     uint32
	Procedure   uint32
	Cred        [2]uint32 // Dummy
	Verf        [2]uint32 // Dummy
}

func NewRpcCall(program uint32, version uint32, procedure uint32) *RpcCall {
	return &RpcCall{
		Xid:         0x07343745,
		MessageType: Call,
		RpcVersion:  2,
		Program:     program,
		Version:     version,
		Procedure:   procedure,
	}
}
