package sunrpc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/davecgh/go-xdr/xdr2"
)

var (
	ErrArgumentsExpected           = errors.New("rpc call: unable to read procedure arguments")
	ErrCallMessageBodyExpected     = errors.New("rpc call: call message body expected")
	ErrCallMessageExpected         = errors.New("rpc call: call message expected")
	ErrHeaderExpected              = errors.New("rpc call: header expected")
	ErrIncompleteMessage           = errors.New("rpc call: cannot read the whole message")
	ErrRPCVersion2Expected         = errors.New("rpc call: trying to read an RPC call of unsupported version")
	ErrUnsupportedMultipleFragment = errors.New("rpc call: fragmented requests are not supported")
)

// NewRecordMarker creates a new record marker as described in RFC 5531.
//
// "When RPC messages are passed on top of a byte stream transport protocol (like TCP), it is
// necessary to delimit one message from another in order to detect and possibly recover from
// protocol errors. This is called record marking (RM). One RPC message fits into one RM record."
//
// The first argument is the size of the subsequent RPC message, the second argument denotes whether
// this marker denotes the last record in this transmission.
//
// See also RFC 5531, Section 11: https://tools.ietf.org/html/rfc5531#section-11
func NewRecordMarker(size uint32, last bool) uint32 {
	marker := size
	marker &^= 1 << 31

	if last {
		marker ^= 0x80000000
	}

	return marker
}

// ParseRecordMarker deconstructs a record marker returning the record size and whether the given
// marker denotes the last frame of an RPC message.
func ParseRecordMarker(marker uint32) (size uint32, last bool) {
	size = marker &^ (1 << 31)
	last = (marker >> 31) == 1

	return size, last
}

func ReadCall(r io.Reader, args interface{}) error {
	var marker uint32

	err := binary.Read(r, binary.LittleEndian, &marker)
	if err != nil {
		return err
	}

	size, last := ParseRecordMarker(marker)

	if !last {
		return ErrUnsupportedMultipleFragment
	}

	read := 0

	// Read RPC message header
	message := Message{}
	bytesRead, err := xdr.Unmarshal(r, &message)
	if err != nil {
		return ErrHeaderExpected
	}

	read += bytesRead

	// Make sure this is a "Call" message
	if message.Type != Call {
		return ErrCallMessageExpected
	}

	// Read RPC call body
	callBody := CallMessage{}
	bytesRead, err = xdr.Unmarshal(r, &callBody)
	if err != nil {
		return ErrCallMessageBodyExpected
	}

	read += bytesRead

	// We can only read RPCv2 messages
	if callBody.RPCVersion != 2 {
		return ErrRPCVersion2Expected
	}

	// Read RPC call arguments
	bytesRead, err = xdr.Unmarshal(r, &args)
	if err != nil {
		return ErrArgumentsExpected
	}

	read += bytesRead

	// We must have read exactly the amount of data reported in the record marker
	if read != int(size) {
		return ErrIncompleteMessage
	}

	return nil
}

func WriteCall(w io.Writer, program uint32, version uint32, proc uint32, args interface{}) error {
	var buf bytes.Buffer

	// Write message header to the buffer
	header := Message{
		Xid:  uint32(time.Now().Unix()),
		Type: Call,
	}

	_, err := xdr.Marshal(&buf, header)
	if err != nil {
		return err
	}

	// Write call message to the buffer
	call := CallMessage{
		RPCVersion: 2,
		Program:    program,
		Version:    version,
		Procedure:  proc,
	}

	_, err = xdr.Marshal(&buf, call)
	if err != nil {
		return err
	}

	// Write procedure arguments to the buffer
	_, err = xdr.Marshal(&buf, &args)
	if err != nil {
		return err
	}

	// Write the record marker before sending the payload
	bytes := buf.Bytes()

	err = binary.Write(w, binary.LittleEndian, NewRecordMarker(uint32(len(bytes)), true))
	if err != nil {
		return err
	}

	// Send the payload
	_, err = w.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}
