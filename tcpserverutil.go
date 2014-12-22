package sunrpc

import (
	"bytes"
	"encoding/binary"
	"io"
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

// ReadTCPCallMessage reads an incoming "call" message from the given reader, returning the parsed
// RPC call message structure, without the common RPC header.
func ReadTCPCallMessage(r io.Reader) (*ProcedureCall, error) {
	var marker uint32

	err := binary.Read(r, binary.LittleEndian, &marker)
	if err != nil {
		return nil, err
	}

	_, last := ParseRecordMarker(marker)

	if !last {
		return nil, ErrUnsupportedMultipleFragment
	}

	return ReadProcedureCall(r)
}

func WriteTCPReplyMessage(w io.Writer, xid uint32, ret interface{}) error {
	// Buffer reply data so that we can compute a proper record marker later on
	var buf bytes.Buffer

	size, err := WriteReplyMessage(&buf, xid, ret)
	if err != nil {
		return err
	}

	// Write the record marker
	//
	// FIXME: Assuming we are sending a single record
	record := NewRecordMarker(uint32(size), true)

	if err := binary.Write(w, binary.BigEndian, record); err != nil {
		return err
	}

	// Write the payload
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
