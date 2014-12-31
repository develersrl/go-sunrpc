package sunrpc

import "errors"

//
// Public Errors
//

var (
	ErrCallMessageExpected         = errors.New("rpc call: call message expected")
	ErrCannotPortmap               = errors.New("portmap: cannot set port with portmapper")
	ErrHeaderExpected              = errors.New("rpc call: header expected")
	ErrIncompleteMessage           = errors.New("rpc call: unable to read the whole message")
	ErrRPCVersion2Expected         = errors.New("rpc call: trying to read an RPC call of unsupported version")
	ErrUnsupportedMultipleFragment = errors.New("rpc call: fragmented requests are not supported")
)

//
// Private Errors
//

var (
	errUnknownFunction = errors.New("rpc call: cannot find function")
)
